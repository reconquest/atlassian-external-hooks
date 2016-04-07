package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.hook.*;
import com.atlassian.bitbucket.hook.repository.*;
import com.atlassian.bitbucket.repository.*;
import com.atlassian.bitbucket.setting.*;
import com.atlassian.bitbucket.user.*;
import com.atlassian.bitbucket.auth.*;
import com.atlassian.bitbucket.permission.*;
import com.atlassian.bitbucket.server.*;
import com.atlassian.bitbucket.util.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collection;
import org.apache.commons.io.FilenameUtils;
import java.io.*;
import java.util.Map;
import java.util.List;
import java.util.LinkedList;
import java.util.Set;
import java.nio.file.Files;

public class ExternalPreReceiveHook
    implements PreReceiveRepositoryHook, RepositorySettingsValidator
{
    private static final Logger log = LoggerFactory.getLogger(
        ExternalPreReceiveHook.class);

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ApplicationPropertiesService properties;

    public ExternalPreReceiveHook(
        AuthenticationContext authenticationContext,
        PermissionService permissions,
        RepositoryService repoService,
        ApplicationPropertiesService properties
    ) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
        this.repoService = repoService;
        this.properties = properties;
    }

    /**
     * Call external executable as git hook.
     */
    @Override
    public boolean onReceive(
        RepositoryHookContext context,
        Collection<RefChange> refChanges,
        HookResponse hookResponse
    ) {
        Repository repo = context.getRepository();

        // compat with  < 3.2.0
        String repoPath = this.properties.getRepositoryDir(repo).getAbsolutePath();

        Settings settings = context.getSettings();
        List<String> exe = new LinkedList<String>();
        exe.add(this.getExecutable(
            settings.getString("exe"),
            settings.getBoolean("safe_path", false)).getPath());

        if (settings.getString("params") != null) {
            for (String arg : settings.getString("params").split("\r\n")) {
                exe.add(arg);
            }
        }

        ApplicationUser currentUser = authCtx.getCurrentUser();
        ProcessBuilder pb = new ProcessBuilder(exe);

        Map<String, String> env = pb.environment();
        env.put("STASH_USER_NAME", currentUser.getName());
        if (currentUser.getEmailAddress() != null) {
            env.put("STASH_USER_EMAIL", currentUser.getEmailAddress());
        } else {
            log.error("Can't get user email address. getEmailAddress() call returns null");
        }
        env.put("STASH_REPO_NAME", repo.getName());

        boolean isAdmin = permissions.hasRepositoryPermission(
            currentUser, repo, Permission.REPO_ADMIN);
        env.put("STASH_IS_ADMIN", String.valueOf(isAdmin));

        RepositoryCloneLinksRequest.Builder cloneLinksRequestBuilder =
            new RepositoryCloneLinksRequest.Builder();

        cloneLinksRequestBuilder.repository(repo);

        RepositoryCloneLinksRequest cloneLinksRequest =
            cloneLinksRequestBuilder.build();

        Set<NamedLink> cloneLinks = this.repoService.getCloneLinks(
            cloneLinksRequest
        );

        for (NamedLink link : cloneLinks) {
            env.put(
                "STASH_REPO_CLONE_" + link.getName().toUpperCase(),
                link.getHref()
            );
        }

        env.put(
            "STASH_BASE_URL",
            this.properties.getBaseUrl().toString()
        );

        env.put("STASH_PROJECT_NAME", repo.getProject().getName());
        env.put("STASH_PROJECT_KEY", repo.getProject().getKey());

        String refChangesEnv = "";
        for (RefChange refChange : refChanges) {
            refChangesEnv += " " + refChange.getRefId();
        }
        env.put("STASH_REF_CHANGES", refChangesEnv);

        pb.directory(new File(repoPath));
        pb.redirectErrorStream(true);
        try {
            Process process = pb.start();
            InputStreamReader input = new InputStreamReader(
                process.getInputStream(), "UTF-8");
            OutputStream output = process.getOutputStream();

            for (RefChange refChange : refChanges) {
                output.write(
                    (
                        refChange.getFromHash() + " " +
                        refChange.getToHash() + " " +
                        refChange.getRefId() + "\n"
                    ).getBytes("UTF-8")
                );
            }
            output.close();

            boolean trimmed = false;
            if (hookResponse != null) {
                int data;
                int count = 0;
                while ((data = input.read()) >= 0) {
                    if (count >= 65000) {
                        if (!trimmed) {
                            hookResponse.err().
                                print("\n");
                            hookResponse.err().
                                print("Hook response exceeds 65K length limit.\n");
                            hookResponse.err().
                                print("Further output will be trimmed.\n");
                            trimmed = true;
                        }
                        continue;
                    }

                    String charToWrite = Character.toString((char)data);

                    count += charToWrite.getBytes("utf-8").length;

                    hookResponse.err().print(charToWrite);
                }

            }

            return process.waitFor() == 0;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return false;
        } catch (IOException e) {
            log.error("Error running {} in {}", exe, repoPath, e);
            return false;
        }
    }

    @Override
    public void validate(
        Settings settings,
        SettingsValidationErrors errors, Repository repository
    ) {
        if (!settings.getBoolean("safe_path", false)) {
            if (!permissions.hasGlobalPermission(
                    authCtx.getCurrentUser(), Permission.SYS_ADMIN)) {
                errors.addFieldError("exe",
                    "You should be  Administrator to edit this field " +
                    "without \"safe mode\" option.");
                return;
            }
        }

        if (settings.getString("exe", "").isEmpty()) {
            errors.addFieldError("exe",
                "Executable is blank, please specify something");
            return;
        }

        File executable = this.getExecutable(
            settings.getString("exe",""),
            settings.getBoolean("safe_path", false));

        boolean isExecutable = false;
        if (executable != null) {
            try {
                isExecutable = executable.canExecute() && executable.isFile();
            } catch (SecurityException e) {
                log.error("Security exception on {}", executable.getPath(), e);
                isExecutable = false;
            }
        } else {
            errors.addFieldError("exe",
                "Specified path for executable can not be resolved.");
            return;
        }

        if (!isExecutable) {
            errors.addFieldError("exe",
                "Specified path is not executable file. Check executable flag.");
            return;
        }

        log.info("Setting executable {}", executable.getPath());
    }

    public File getExecutable(String path, boolean safeDir) {
        File executable = new File(path);
        if (safeDir) {
            path = FilenameUtils.normalize(path);
            if (path == null) {
                executable = null;
            } else {
                String safeBaseDir =
                    this.properties.getHomeDir().getAbsolutePath() +
                    "/external-hooks/";
                executable = new File(safeBaseDir, path);
            }
        }

        return executable;
    }
}

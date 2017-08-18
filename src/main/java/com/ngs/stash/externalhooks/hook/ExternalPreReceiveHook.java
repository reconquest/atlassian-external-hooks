package com.ngs.stash.externalhooks.hook;

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


public class ExternalPreReceiveHook
    implements PreRepositoryHook<RepositoryHookRequest>, RepositorySettingsValidator
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

	@Override
	public RepositoryHookResult preUpdate(PreRepositoryHookContext context, RepositoryHookRequest request) {
        return preUpdateImpl(context, request);
	}
	
	public RepositoryHookResult preUpdateImpl(RepositoryHookContext context, RepositoryHookRequest request) {
        Repository repo = request.getRepository();
        Settings settings = context.getSettings();

        // compat with < 3.2.0
        String repoPath = this.properties.getRepositoryDir(repo).getAbsolutePath();
        List<String> exe = new LinkedList<String>();

        ProcessBuilder pb = createProcessBuilder(repo, repoPath, exe, settings, request);
        
        try {
            return runExternalHooks(pb, request.getRefChanges(), "Push rejected");
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return RepositoryHookResult.rejected("error", "an error occurred");
        } catch (IOException e) {
            log.error("Error running {} in {}", exe, repoPath, e);
            return RepositoryHookResult.rejected("error", "an error occurred");
        }
	}

    public ProcessBuilder createProcessBuilder(
        Repository repo, String repoPath, List<String> exe, Settings settings, RepositoryHookRequest request
    ) {
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
        
        if (request.getScmHookDetails().isPresent()) {
        	env.putAll(request.getScmHookDetails().get().getEnvironment());
        }

        boolean isAdmin = permissions.hasRepositoryPermission(
            currentUser, repo, Permission.REPO_ADMIN);
        boolean isWrite = permissions.hasRepositoryPermission(currentUser, repo, Permission.REPO_WRITE);
        boolean isDirectAdmin = permissions.hasDirectRepositoryUserPermission(repo, Permission.REPO_ADMIN);
        boolean isDirectWrite = permissions.hasDirectRepositoryUserPermission(repo, Permission.REPO_WRITE);
        env.put("STASH_IS_ADMIN", String.valueOf(isAdmin));
        env.put("STASH_IS_WRITE", String.valueOf(isWrite));
        env.put("STASH_IS_DIRECT_ADMIN", String.valueOf(isDirectAdmin));
        env.put("STASH_IS_DIRECT_WRITE", String.valueOf(isDirectWrite));
        env.put("STASH_REPO_IS_FORK", String.valueOf(repo.isFork()));

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

        pb.directory(new File(repoPath));
        pb.redirectErrorStream(true);

        return pb;
    }

    public RepositoryHookResult runExternalHooks(
        ProcessBuilder pb,
        Collection<RefChange> refChanges,
        String summaryMessage
    ) throws InterruptedException, IOException {
        Process process = pb.start();
        InputStreamReader input = new InputStreamReader(
                                                        process.getInputStream(), "UTF-8");
        OutputStream output = process.getOutputStream();

        for (RefChange refChange : refChanges) {
            output.write(
                         (
                          refChange.getFromHash() + " " +
                          refChange.getToHash() + " " +
                          refChange.getRef().getId() + "\n"
                          ).getBytes("UTF-8")
                         );
        }
        output.close();

        boolean trimmed = false;
        int data;
        int count = 0;
        StringBuilder builder = new StringBuilder();
        while ((data = input.read()) >= 0) {
            if (count >= 65000) {
                if (!trimmed) {
                	builder.
                        append("\n");
                	builder.
                        append("Hook response exceeds 65K length limit.\n");
                	builder.
                		append("Further output will be trimmed.\n");
                    trimmed = true;
                }
                continue;
            }

            String charToWrite = Character.toString((char)data);

            count += charToWrite.getBytes("utf-8").length;

            builder.append(charToWrite);
        }
        
        int result = process.waitFor();
        
        if (result == 0) {
        	return RepositoryHookResult.accepted();
        } else {
        	return RepositoryHookResult.rejected(summaryMessage, builder.toString());
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
                    "You should be a Bitbucket System Administrator to edit this field " +
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

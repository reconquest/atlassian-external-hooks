package com.ngs.stash.externalhooks.hook;

import com.atlassian.stash.hook.*;
import com.atlassian.stash.hook.repository.*;
import com.atlassian.stash.repository.*;
import com.atlassian.stash.setting.*;
import com.atlassian.stash.env.SystemProperties;
import com.atlassian.stash.user.*;
import com.atlassian.stash.server.*;
import com.atlassian.stash.util.*;
import com.atlassian.stash.pull.*;
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
import com.atlassian.stash.user.Permission;
import com.atlassian.stash.user.PermissionService;

public class ExternalMergeCheckHook
    implements RepositoryMergeRequestCheck, RepositorySettingsValidator
{
    private static final Logger log = LoggerFactory.getLogger(
        ExternalMergeCheckHook.class);

    private StashAuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ApplicationPropertiesService properties;

    public ExternalMergeCheckHook(
        StashAuthenticationContext authenticationContext,
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
    public void check(RepositoryMergeRequestCheckContext context) {
        PullRequest pr = context.getMergeRequest().getPullRequest();
        Repository repo = pr.getToRef().getRepository();

        // compat with Stash < 3.2.0
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

        StashUser currentUser = authCtx.getCurrentUser();
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

        String summaryMsg = "Merge request failed";

        pb.directory(new File(repoPath));
        pb.redirectErrorStream(true);
        try {
            Process process = pb.start();
            InputStreamReader input = new InputStreamReader(
                process.getInputStream(), "UTF-8");
            OutputStream output = process.getOutputStream();

            output.write((
                          pr.getFromRef().getRepository().getSlug() + " " +
                          pr.getFromRef().getLatestChangeset() + " " +
                          pr.getToRef().getRepository().getSlug() + " " +
                          pr.getToRef().getLatestChangeset() + "\n"
                          ).getBytes("UTF-8"));
            output.close();

            int data;
            int count = 0;
            String response = "";
            while ((data = input.read()) >= 0) {
                response += Character.toString((char)data);
            }
            boolean Accepted = process.waitFor() == 0;
            if (!Accepted) {
                String detailedMsg = response;
                context.getMergeRequest().veto(summaryMsg, detailedMsg);
            }
            return;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            String detailedMsg = "Interrupted";
            context.getMergeRequest().veto(summaryMsg, detailedMsg);
            return;
        } catch (IOException e) {
            log.error("Error running {} in {}", exe, repoPath, e);
            String detailedMsg = "I/O error";
            context.getMergeRequest().veto(summaryMsg, detailedMsg);
            return;
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
                    "You should be Stash Administrator to edit this field " +
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

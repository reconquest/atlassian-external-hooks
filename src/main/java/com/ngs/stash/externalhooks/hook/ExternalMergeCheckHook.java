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

import com.atlassian.bitbucket.pull.*;
import static com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook.REPO_PROTOCOL.http;
import static com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook.REPO_PROTOCOL.ssh;


public class ExternalMergeCheckHook
    implements RepositoryMergeRequestCheck, RepositorySettingsValidator
{
    private static final Logger log = LoggerFactory.getLogger(
        ExternalMergeCheckHook.class);

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ApplicationPropertiesService properties;

    public ExternalMergeCheckHook(
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
    public void check(
        RepositoryMergeRequestCheckContext context
    ) {
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

        // Using the same env variables as
        // https://github.com/tomasbjerre/pull-request-notifier-for-bitbucket
        env.put("PULL_REQUEST_FROM_HASH", pr.getFromRef().getLatestCommit());
        env.put("PULL_REQUEST_FROM_ID", pr.getFromRef().getId());
        env.put("PULL_REQUEST_FROM_BRANCH", pr.getFromRef().getDisplayId());
        env.put("PULL_REQUEST_FROM_REPO_ID", pr.getFromRef().getRepository().getId() + "");
        env.put("PULL_REQUEST_FROM_REPO_NAME", pr.getFromRef().getRepository().getName() + "");
        env.put("PULL_REQUEST_FROM_REPO_PROJECT_ID", pr.getFromRef().getRepository().getProject().getId() + "");
        env.put("PULL_REQUEST_FROM_REPO_PROJECT_KEY", pr.getFromRef().getRepository().getProject().getKey());
        env.put("PULL_REQUEST_FROM_REPO_SLUG", pr.getFromRef().getRepository().getSlug() + "");
        env.put("PULL_REQUEST_FROM_SSH_CLONE_URL", cloneUrlFromRepository(ssh, pr.getFromRef().getRepository(), repoService));
        env.put("PULL_REQUEST_FROM_HTTP_CLONE_URL", cloneUrlFromRepository(http, pr.getFromRef().getRepository(), repoService));
        env.put("PULL_REQUEST_URL", getPullRequestUrl(properties, pr));
        env.put("PULL_REQUEST_ID", pr.getId() + "");
        env.put("PULL_REQUEST_VERSION", pr.getVersion() + "");
        env.put("PULL_REQUEST_AUTHOR_ID", pr.getAuthor().getUser().getId() + "");
        env.put("PULL_REQUEST_AUTHOR_DISPLAY_NAME", pr.getAuthor().getUser().getDisplayName());
        env.put("PULL_REQUEST_AUTHOR_NAME", pr.getAuthor().getUser().getName());
        env.put("PULL_REQUEST_AUTHOR_EMAIL", pr.getAuthor().getUser().getEmailAddress());
        env.put("PULL_REQUEST_AUTHOR_SLUG", pr.getAuthor().getUser().getSlug());
        env.put("PULL_REQUEST_TO_HASH", pr.getToRef().getLatestCommit());
        env.put("PULL_REQUEST_TO_ID", pr.getToRef().getId());
        env.put("PULL_REQUEST_TO_BRANCH", pr.getToRef().getDisplayId());
        env.put("PULL_REQUEST_TO_REPO_ID", pr.getToRef().getRepository().getId() + "");
        env.put("PULL_REQUEST_TO_REPO_NAME", pr.getToRef().getRepository().getName() + "");
        env.put("PULL_REQUEST_TO_REPO_PROJECT_ID", pr.getToRef().getRepository().getProject().getId() + "");
        env.put("PULL_REQUEST_TO_REPO_PROJECT_KEY", pr.getToRef().getRepository().getProject().getKey());
        env.put("PULL_REQUEST_TO_REPO_SLUG", pr.getToRef().getRepository().getSlug() + "");
        env.put("PULL_REQUEST_TO_SSH_CLONE_URL", cloneUrlFromRepository(ssh, pr.getToRef().getRepository(), repoService));
        env.put("PULL_REQUEST_TO_HTTP_CLONE_URL", cloneUrlFromRepository(http, pr.getToRef().getRepository(), repoService));
        env.put("PULL_REQUEST_TITLE", pr.getTitle());

        String summaryMsg = "Merge request failed";

        pb.directory(new File(repoPath));
        pb.redirectErrorStream(true);
        try {
            Process process = pb.start();
            InputStreamReader input = new InputStreamReader(
                process.getInputStream(), "UTF-8");
            OutputStream output = process.getOutputStream();

            output.write((
                          pr.getToRef().getLatestCommit() + " " +
                          pr.getFromRef().getLatestCommit() + " " +
                          pr.getToRef().getId() + "\n"
                          ).getBytes("UTF-8"));
            output.close();

            String hookResponse = "";
            boolean trimmed = false;
            int data;
            int count = 0;
            while ((data = input.read()) >= 0) {
                if (count >= 65000) {
                    if (!trimmed) {
                        hookResponse += "\n";
                        hookResponse += "Hook response exceeds 65K length limit.\n";
                        hookResponse += "Further output will be trimmed.\n";
                        trimmed = true;
                    }
                    continue;
                }

                String charToWrite = Character.toString((char)data);

                count += charToWrite.getBytes("utf-8").length;

                hookResponse += charToWrite;
            }

            boolean Accepted = process.waitFor() == 0;
            if (!Accepted) {
                String prePrefix = "<pre style=\"overflow: auto; white-space: nowrap;\">";
                String preSuffix = "</pre>";
                String detailedMsg = prePrefix + hookResponse.replaceAll("(\r\n|\n)", "<br/>").replaceAll(" ", " ") + preSuffix;
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

    public enum REPO_PROTOCOL {
        ssh, http
    }

    private static String cloneUrlFromRepository(
        REPO_PROTOCOL protocol,
        Repository repository,
        RepositoryService repoService
    ) {
      RepositoryCloneLinksRequest request = new RepositoryCloneLinksRequest.Builder().protocol(protocol.name())
          .repository(repository).build();
      final Set<NamedLink> cloneLinks = repoService.getCloneLinks(request);
      return cloneLinks.iterator().hasNext() ? cloneLinks.iterator().next().getHref() : "";
    }

    private static String getPullRequestUrl(
        ApplicationPropertiesService propertiesService,
        PullRequest pullRequest
    ) {
        return propertiesService.getBaseUrl() + "/projects/" + pullRequest.getToRef().getRepository().getProject().getKey()
            + "/repos/" + pullRequest.getToRef().getRepository().getSlug() + "/pull-requests/" + pullRequest.getId();
    }
}

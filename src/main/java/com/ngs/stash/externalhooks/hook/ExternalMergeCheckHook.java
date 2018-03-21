package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.hook.repository.*;
import com.atlassian.bitbucket.repository.*;
import com.atlassian.bitbucket.setting.*;
import com.atlassian.bitbucket.auth.*;
import com.atlassian.bitbucket.permission.*;
import com.atlassian.bitbucket.server.*;
import com.atlassian.bitbucket.util.*;

import static java.util.logging.Level.SEVERE;
import static java.util.logging.Level.INFO;
import java.util.logging.Logger;

import java.util.Collection;
import java.io.*;
import java.util.Map;
import java.util.List;
import java.util.LinkedList;
import java.util.Set;

import java.util.ArrayList;
import com.atlassian.bitbucket.pull.*;
import com.ngs.stash.externalhooks.hook.helpers.*;
// import com.google.common.base.Predicate;
// import static com.google.common.base.Charsets.UTF_8;
// import static com.google.common.base.Joiner.on;
// import static com.google.common.base.Throwables.propagate;
// import static com.google.common.collect.Iterables.filter;
// import static com.google.common.collect.Iterables.transform;
// import static com.google.common.collect.Lists.newArrayList;
// import static com.google.common.collect.Ordering.usingToString;
import static com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook.REPO_PROTOCOL.http;
import static com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook.REPO_PROTOCOL.ssh;


import com.atlassian.upm.api.license.PluginLicenseManager;
public class ExternalMergeCheckHook
    implements RepositoryMergeCheck, RepositorySettingsValidator
{
    private final PluginLicenseManager pluginLicenseManager;
    private static Logger log = Logger.getLogger(
        ExternalMergeCheckHook.class.getSimpleName()
    );

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ApplicationPropertiesService properties;

    public ExternalMergeCheckHook(
        AuthenticationContext authenticationContext,
        PermissionService permissions,
        RepositoryService repoService,
        ApplicationPropertiesService properties,
        PluginLicenseManager pluginLicenseManager
    ) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
        this.repoService = repoService;
        this.properties = properties;
        this.pluginLicenseManager = pluginLicenseManager;
    }

    @Override
    public RepositoryHookResult preUpdate(PreRepositoryHookContext context, PullRequestMergeHookRequest request) {
        PullRequest pr = request.getPullRequest();
        Repository repo = pr.getToRef().getRepository();
        Settings settings = context.getSettings();

        // compat with < 3.2.0
        String repoPath = this.properties.getRepositoryDir(repo).getAbsolutePath();
        List<String> exe = new LinkedList<String>();

        ProcessBuilder pb = createProcessBuilder(repo, repoPath, exe, settings, request);

        List<RefChange> refChanges = new ArrayList<RefChange>();
        refChanges.add(
            new ExternalRefChange(
                pr.getToRef().getId(),
                pr.getToRef().getLatestCommit(),
                pr.getFromRef().getLatestCommit(),
                RefChangeType.UPDATE,
                pr.getToRef()
            )
        );

        Map<String, String> env = pb.environment();

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
        // env.put("PULL_REQUEST_ACTION", prnfbPullRequestAction.getName());
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
        // env.put("PULL_REQUEST_COMMENT_TEXT", getOrEmpty(variables, PULL_REQUEST_COMMENT_TEXT));
        // env.put("PULL_REQUEST_MERGE_COMMIT", getOrEmpty(variables, PULL_REQUEST_MERGE_COMMIT));
        // env.put("PULL_REQUEST_USER_DISPLAY_NAME", applicationUser.getDisplayName());
        // env.put("PULL_REQUEST_USER_EMAIL_ADDRESS", applicationUser.getEmailAddress());
        // env.put("PULL_REQUEST_USER_ID", applicationUser.getId() + "");
        // env.put("PULL_REQUEST_USER_NAME", applicationUser.getName());
        // env.put("PULL_REQUEST_USER_SLUG", applicationUser.getSlug());
        env.put("PULL_REQUEST_TITLE", pr.getTitle());
        // env.put("PULL_REQUEST_REVIEWERS", iterableToString(transform(pr.getReviewers(), (p) -> p.getUser().getDisplayName())));
        // env.put("PULL_REQUEST_REVIEWERS_ID", iterableToString(transform(pr.getReviewers(), (p) -> Integer.toString(p.getUser().getId()))));
        // env.put("PULL_REQUEST_REVIEWERS_SLUG", iterableToString(transform(pr.getReviewers(), (p) -> p.getUser().getSlug())));
        // env.put("PULL_REQUEST_REVIEWERS_APPROVED_COUNT", Integer.toString(newArrayList(filter(pr.getReviewers(), isApproved)).size()));
        // env.put("PULL_REQUEST_PARTICIPANTS_APPROVED_COUNT", Integer.toString(newArrayList(filter(pr.getParticipants(), isApproved)).size()));

        String summaryMsg = "Merge request failed";

        try {
            return runExternalHooks(pb, refChanges, summaryMsg);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            String detailedMsg = "Interrupted";
            return RepositoryHookResult.rejected(summaryMsg, detailedMsg);
        } catch (IOException e) {
            log.log(SEVERE, "Error running "+exe+" in "+repoPath, e);
            String detailedMsg = "I/O Error";
            return RepositoryHookResult.rejected(summaryMsg, detailedMsg);
        }
    }

    public ProcessBuilder createProcessBuilder(
        Repository repo, String repoPath, List<String> exe, Settings settings, RepositoryHookRequest request
    ) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx,
            this.permissions, this.repoService, this.properties, this.pluginLicenseManager);
        return impl.createProcessBuilder(repo, repoPath, exe, settings, request);
    }

    public RepositoryHookResult runExternalHooks(
        ProcessBuilder pb,
        Collection<RefChange> refChanges,
        String summaryMessage
    ) throws InterruptedException, IOException {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx,
            this.permissions, this.repoService, this.properties, this.pluginLicenseManager);
        return impl.runExternalHooks(pb, refChanges, summaryMessage);
    }

    @Override
    public void validate(
        Settings settings,
        SettingsValidationErrors errors,
        Repository repository
    ) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx,
            this.permissions, this.repoService, this.properties, this.pluginLicenseManager);
        impl.validate(settings, errors, repository);
    }

    public enum REPO_PROTOCOL {
        ssh, http
    }

    // private static final Predicate<PullRequestParticipant> isApproved = new Predicate<PullRequestParticipant>() {
    //         @Override
    //         public boolean apply(PullRequestParticipant input) {
    //             return input.isApproved();
    //         }
    //     };

    // private static String iterableToString(Iterable<String> slist) {
    //     List<String> sorted = usingToString().sortedCopy(slist);
    //     return on(',').join(sorted);
    // }

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

package com.ngs.stash.externalhooks.hook.listeners;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.atlassian.bitbucket.event.pull.PullRequestDeletedEvent;
import com.atlassian.event.api.EventListener;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.hook.data.PullRequestCheck;
import net.java.ao.Query;

import javax.inject.Inject;
import javax.inject.Named;

/**
 * A pull request event listener
 */
@Named("PullRequestListener")
public class PullRequestListener {
    private final ActiveObjects activeObjects;

    @Inject
    public PullRequestListener(
        @ComponentImport final ActiveObjects activeObjects) {
        this.activeObjects = activeObjects;
    }

    /**
     * An event lister for delete pull request events
     * @param event
     */
    @EventListener
    public void onDeleteEvent(final PullRequestDeletedEvent event) {
        // Delete connected pull request check entries
        this.activeObjects.delete(
            this.activeObjects.find(
                PullRequestCheck.class,
                Query.select().where(
                    "PROJECT_ID = ? AND REPOSITORY_ID = ? AND PULL_REQUEST_ID = ?",
                    event.getPullRequest().getToRef().getRepository().getProject().getId(),
                    event.getPullRequest().getToRef().getRepository().getId(),
                    event.getPullRequest().getId()
                )
            )
        );
    }
}

package com.ngs.stash.externalhooks.hook.listeners;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.atlassian.bitbucket.event.project.ProjectDeletedEvent;
import com.atlassian.bitbucket.event.repository.RepositoryDeletedEvent;
import com.atlassian.event.api.EventListener;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.hook.data.PullRequestCheck;
import net.java.ao.Query;

import javax.inject.Inject;
import javax.inject.Named;

/**
 * A listener for repository events
 */
@Named("RepositoryListener")
public class RepositoryListener {
    private final ActiveObjects activeObjects;

    @Inject
    public RepositoryListener(@ComponentImport ActiveObjects activeObjects) {
        this.activeObjects = activeObjects;
    }

    /**
     * A listener for the repository delete event
     * @param event
     */
    @EventListener
    public void onDeleteEvent(RepositoryDeletedEvent event) {
        // Delete connected pull request check entries
        this.activeObjects.delete(
            this.activeObjects.find(
                PullRequestCheck.class,
                Query.select().where(
                    "PROJECT_ID = ? AND REPOSITORY_ID = ?",
                    event.getRepository().getProject().getId(),
                    event.getRepository().getId()
                )
            )
        );
    }
}

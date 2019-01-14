package com.ngs.stash.externalhooks.hook.data;

import net.java.ao.Entity;
import net.java.ao.schema.NotNull;
import net.java.ao.schema.StringLength;
import net.java.ao.schema.Table;

/**
 * Informations about what checks were already carried out on a
 * pull request to avoid duplicated checks
 */
@Table("PRChecks")
public interface PullRequestCheck extends Entity {

    /**
     * The project id of the pull request
     *
     * @return project id
     */
    @NotNull
    int getProjectId();

    void setProjectId(int projectId);

    /**
     * The repository id of the pull request
     *
     * @return repository id
     */
    @NotNull
    int getRepositoryId();

    void setRepositoryId(int repositoryId);

    /**
     * The pull request id
     * @return pull request id
     */
    @NotNull
    long getPullRequestId();

    void setPullRequestId(long pullRequestId);

    /**
     * The version of the pull request
     * @return version number
     */
    @NotNull
    int getVersion();

    void setVersion(int version);

    /**
     * Wether the last check resulted in an acceptance
     * @return true: last check was accepted, false: last check was rejected
     */
    boolean getWasAccepted();

    void setWasAccepted(boolean wasExcepted);

    /**
     * If a comment was added for the last check and, optionally, the PR was declined
     * @return true: comment/decline was carried out
     */
    boolean getWasHandled();

    void setWasHandled(boolean wasHandled);

    /**
     * Get the summary string from the last rejection
     * @return summary
     */
    String getLastExceptionSummary();

    @StringLength(StringLength.UNLIMITED)
    void setLastExceptionSummary(String lastExceptionSummary);

    /**
     * Get the detail string from the last rejection
     * @return summary
     */
    String getLastExceptionDetail();

    @StringLength(StringLength.UNLIMITED)
    void setLastExceptionDetail(String lastExceptionSummary);


    /**
     * Get timestamp of executable file used to run check
     * @return timestamp
     */
    @NotNull
    long getExecutableVersion();

    void setExecutableVersion(long executableVersion);
}

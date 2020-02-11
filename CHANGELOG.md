# 9.1.0

Fix need for re-configuration of hooks in personal repositories after the
add-on enable/disable lifecycle (e.g., after BB restart).

Indirectly fixes migration problem for hooks in personal repositories from BB
4.14.4 to 6.10 along with add-on upgrade.

# 9.0.1

Add organization url to add-on manifest file.

It resolves problem on Manage Apps admin page on some installations.

# 9.0.0

Add global triggers' configuration which is accessible from Bitbucket
Administration Panel.

Now users with System Admin role can select events which will trigger pre-,
post-receive & merge check hooks.

Events available for configuration:

* push to repo,
* web UI: branch create/delete,
* web UI: tag create/delete,
* web UI: file edit,
* web UI: pull request merge check,
* internal: merge event from other plugins.

See documentation for more information:

https://external-hooks.reconquest.io/docs/triggers/

# 8.0.0

Bug fixes & minor improvements.

Additional fixes for:

https://github.com/reconquest/atlassian-external-hooks/issues/100

# 7.5.0

Fix bug causing inherited hooks to be force-enabled.

# 7.3.0

Fix BB upgrade problem (BB 6.5.1 -> 6.6.0).

https://github.com/reconquest/atlassian-external-hooks/issues/100

# 7.2.0

Revert change made in 6.3.0: do not invoke pre- & post-receive hook on pull
request merge check.

This feature is already covered by Merge Check Hook.

# 7.1.0

Added 'async' option for post-receive hook configuration.

By default post-receive hooks are not running in async mode, which means that
`git push` process will wait until post-receive hook completes.

Pre-receive hook with enabled 'async' option will run in background, making
possible to start time-consuming tasks such as CI.

Note, that it's not possible to return any output back to the user invoking
`git push` from 'async' post-receive hook.

# 7.0.0

The new feature — _Asynchronous_ — has been added to the **Post Receive Hook**
setup. The field is useful for users, who recently upgraded from
an old version of the add-on and after that
the Post Receive Hook is executed in synchronous mode,
that causes the git push command executes with delays.

<img src="https://external-hooks.reconquest.io/img/7.0.0-new-feature.png"/>

# 6.3.2

Pre- & post-receive hooks are extended to be triggered of the following events
made from BB UI:

* file edit.

# 6.3.0

Pre- & post-receive hooks are extended to be triggered of the following events
made from BB UI:

* tag create,
* tag delete,
* branch create,
* branch delete,
* pull request merge.

# 6.2.0

Pre- and Post-Receive Hooks will now always pass theirs' output to user no
matter which exit code was returned from script.

Following environment variables are now marked as deprecated and their
alternatives should be considered to be used instead. No immediate change
required.

* `STASH_USER_NAME` → `BB_USER_DISPLAY_NAME`
* `STASH_USER_NAME` → `BB_USER_DISPLAY_NAME`
* `STASH_USER_EMAIL` → `BB_USER_EMAIL`
* `STASH_REPO_NAME` → `BB_REPO_SLUG`
* `STASH_REPO_IS_FORK` → `BB_REPO_IS_FORK`
* `STASH_PROJECT_KEY` → `BB_PROJECT_KEY`
* `STASH_BASE_URL` → `BB_BASE_URL`
* `STASH_REPO_CLONE_SSH` → `BB_REPO_CLONE_SSH`
* `STASH_REPO_CLONE_HTTP` → `BB_REPO_CLONE_HTTP` 

Following environment variables were removed without replacement due
limitations in Bitbucket Server starting from 6.2.0.

* `STASH_PROJECT_NAME`
* `STASH_IS_DIRECT_WRITE`
* `STASH_IS_DIRECT_ADMIN`
* `PULL_REQUEST_FROM_HASH`
* `PULL_REQUEST_FROM_ID`
* `PULL_REQUEST_FROM_BRANCH`
* `PULL_REQUEST_FROM_REPO_ID`
* `PULL_REQUEST_FROM_REPO_NAME`
* `PULL_REQUEST_FROM_REPO_PROJECT_ID`
* `PULL_REQUEST_FROM_REPO_PROJECT_KEY`
* `PULL_REQUEST_FROM_REPO_SLUG`
* `PULL_REQUEST_FROM_SSH_CLONE_URL`
* `PULL_REQUEST_FROM_HTTP_CLONE_URL`
* `PULL_REQUEST_URL`
* `PULL_REQUEST_ID`
* `PULL_REQUEST_TITLE`
* `PULL_REQUEST_VERSION`
* `PULL_REQUEST_AUTHOR_ID`
* `PULL_REQUEST_AUTHOR_DISPLAY_NAME`
* `PULL_REQUEST_AUTHOR_NAME`
* `PULL_REQUEST_AUTHOR_EMAIL`
* `PULL_REQUEST_AUTHOR_SLUG`
* `PULL_REQUEST_TO_HASH`
* `PULL_REQUEST_TO_ID`
* `PULL_REQUEST_TO_BRANCH`
* `PULL_REQUEST_TO_REPO_ID`
* `PULL_REQUEST_TO_REPO_NAME`
* `PULL_REQUEST_TO_REPO_PROJECT_ID`
* `PULL_REQUEST_TO_REPO_PROJECT_KEY`
* `PULL_REQUEST_TO_REPO_SLUG`
* `PULL_REQUEST_TO_SSH_CLONE_URL`
* `PULL_REQUEST_TO_HTTP_CLONE_URL`

Merge Check will no longer add comments to Pull Requests or automatically
reject them and no such configuration is possible.

The following configuration is **not available**:

<img src="https://external-hooks.reconquest.io/img/7.0.0-new-feature.png"/>

If your workflow requires this feature, please contact us.

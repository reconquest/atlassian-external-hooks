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

See documentation for more information: https://external-hooks.reconquest.io/docs/triggers/

# 8.0.0

Bug fixes & minor improvements.

Additional fixes for https://github.com/reconquest/atlassian-external-hooks/issues/100

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

Merge Check will no longer add comments to Pull Requests or automatically
reject them and no such configuration is possible.

Pre- and Post-Receive Hooks will now always pass theirs' output to user no
matter which exit code was returned from script.

## Bitbucket 6.2.0 and higher

### 12.0.1 (2021-11-05)

* Fix compatibility of the global hooks feature and Oracle/Postgers databases.

### 12.0.0 (2021-10-20)

* New feature: [global hooks](https://external-hooks.reconquest.io/docs/global_hooks/).
* Fixed a bug where configuring a project hook after a repository hook led to having two hooks at the same
    time, while doing the same in a different order did not lead to having two hooks. Now
    configuring both project and repository hooks means a repository hook *overrides* project hook as
    it was supposted to.

### 11.1.0 (2020-11-13)

Fix back-compatibility of STASH_USER_NAME. It was equal to BB_USER_DISPLAY_NAME by a mistake.

* STASH_USER_NAME is now equal to BB_USER_NAME.
* STASH_USER_DISPLAY_NAME is now equal to BB_USER_DISPLAY_NAME.

### 11.0.0 (2020-10-22)

The Diagnostics page in the admin panel added.

This page provides a few specific features:
* Change the logging level of the add-on.
* Download dump of HookScripts usable for debugging purposes.
* Remove all existing hooks from the add-on. This button is not going to be used very often, it's useful for emergency cases only.

### 10.2.2 (2020-10-20)

Fix error when the user without project admin access but with repository admin
access changes inherited hook.

Previously, hook state change from Inherited to Enabled (or Disabled) and back
to Inherited by such user caused Project level hook script to be ignored
completely. The internal permission model in Bitbucket caused this error.

### 10.2.1 (2020-05-15)

* Change the severity of log messages during startup from Warning to Info.
* Fix grammar typos

### 10.2.0 (2020-04-15)

Properly override project-level hooks with repository hooks.

Now, for any given repository, if there are both project-level and repository-level
hooks are enabled, only repository-level hook will be executed, completely
overriding project-level settings.

### 10.1.0 (2020-03-06)

Fix bug:
* project level hooks not triggering with inheritance on first commit/push to
    new repository

Original issue: https://github.com/reconquest/atlassian-external-hooks/issues/109

### 10.0.0 (2020-02-28)

Allow to disable hooks on repository level bypassing project level hooks.

Previously, due to internal changes in Bitbucket made in 6+ version, it was not
possible to disable hooks on specific repository while having project level
hook enabled.

### 9.1.0 (2020-03-02)

Fix need for re-configuration of hooks in personal repositories after the
add-on enable/disable life-cycle (e.g., after BB restart).

Indirectly fixes migration problem for hooks in personal repositories from BB
4.14.4 to 6.10 along with add-on upgrade.

### 9.0.1 (2020-02-06)

Add organization URL to add-on manifest file.

It resolves problem on Manage Apps admin page on some installations.

### 9.0.0 (2019-11-29)

Add global triggers' configuration which is accessible from Bitbucket
Administration Panel.

Now users with System Admin role can select events which will trigger pre-,
post-receive & merge check hooks.

Events available for configuration:

* push to repository,
* web UI: branch create/delete,
* web UI: tag create/delete,
* web UI: file edit,
* web UI: pull request merge check,
* internal: merge event from other plugins.

See documentation for more information:

https://external-hooks.reconquest.io/docs/triggers/

### 8.0.0 (2019-11-11)

Bug fixes & minor improvements.

Additional fixes for:

https://github.com/reconquest/atlassian-external-hooks/issues/100

### 7.5.0 (2019-11-05)

Fix bug causing inherited hooks to be force-enabled.

### 7.3.0 (2019-10-31)

Fix BB upgrade problem (BB 6.5.1 -> 6.6.0).

https://github.com/reconquest/atlassian-external-hooks/issues/100

### 7.2.0 (2019-10-04)

Revert change made in 6.3.0: do not invoke pre- & post-receive hook on pull
request merge check.

This feature is already covered by Merge Check Hook.

### 7.1.0 (2019-09-11)

Added 'async' option for post-receive hook configuration.

By default post-receive hooks are not running in async mode, which means that
`git push` process will wait until post-receive hook completes.

Pre-receive hook with enabled 'async' option will run in background, making
possible to start time-consuming tasks such as CI.

Note, that it's not possible to return any output back to the user invoking
`git push` from 'async' post-receive hook.

### 7.0.0 (2019-09-09)

The new feature — _Asynchronous_ — has been added to the **Post Receive Hook**
setup. The field is useful for users, who recently upgraded from
an old version of the add-on and after that
the Post Receive Hook is executed in synchronous mode,
that causes the git push command executes with delays.

<img src="https://external-hooks.reconquest.io/img/7.0.0-new-feature.png"/>

### 6.3.2 (2019-07-20)

Pre- & post-receive hooks are extended to be triggered of the following events
made from BB UI:

* file edit.

### 6.3.1 (2019-06-12)

Pre- & post-receive hooks are extended to be triggered of the following events
made from BB UI:

* tag create,
* tag delete,
* branch create,
* branch delete,
* pull request merge.

### 6.2.0 (2019-05-16)

Pre- and Post-Receive Hooks will now always pass theirs' output to user no
matter which exit code was returned from script.

Following environment variables are now marked as deprecated and their
alternatives should be considered to be used instead. No immediate change
required.

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

## Bitbucket 5

### 4.8.1 (2020-03-02)

Backport: do not trigger pre-receive on merge check/rebase dry run.

### 4.8 (2019-04-17)

Bitbucket Data Center compatibility.

### 4.7 (2019-01-14)

Two new options for merge check to control cache & tooltip.

### 4.6 (2018-11-21)

Merge checks: update results if merge check file changed.

### 4.5 (2018-11-05)

Fix Merge Check to work with git refs in forked repositories.

Fix Java Exception in Merge Check Hook.

Merge Checks can be configured to add comments to Pull Requests.

### 4.4 (2018-07-25)

Enable project-level hooks configuration.

### 4.3 (2018-04-04)

Compatibility with latest Bitbuckets versions & support offer.

**This is first supported version. Previous add-on versions are free to use and not supported.**

## Unsupported add-on version

### 3.4 (2017-06-13)

Add STASH_IS_DRY_RUN environment variable.

### 3.3 (2017-06-07)

Compatibility with Bitbucket 5+.

Data Center support & additional environment variables.

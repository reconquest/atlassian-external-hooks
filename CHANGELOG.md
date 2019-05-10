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

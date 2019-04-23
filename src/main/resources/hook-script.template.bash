#!/bin/bash

# STASH_ variables are deprectaed and will be removed in further releases
export STASH_USER_NAME="$BB_USER_DISPLAY_NAME"
export STASH_USER_NAME="$BB_USER_DISPLAY_NAME"
export STASH_USER_EMAIL="$BB_USER_EMAIL"
export STASH_REPO_NAME="$BB_REPO_SLUG"
export STASH_REPO_IS_FORK="$BB_REPO_IS_FORK"
export STASH_PROJECT_KEY="$BB_PROJECT_KEY"
export STASH_BASE_URL="$BB_BASE_URL"
export STASH_REPO_CLONE_SSH="$BB_REPO_CLONE_SSH"
export STASH_REPO_CLONE_HTTP="$BB_REPO_CLONE_HTTP"

# There is no way to obtain STASH_PROJECT_NAME (human-readable name of the
# project repository belongs to) since Bitbucket 6.2.0+
export STASH_PROJECT_NAME=
export STASH_IS_DIRECT_WRITE=
export STASH_IS_DIRECT_ADMIN=

if [[ "$BB_USER_PERMISSIONS" == "SYS_ADMIN"
    || "$BB_USER_PERMISSIONS" == "ADMIN"
    || "$BB_USER_PERMISSIONS" == "REPO_ADMIN" ]]; then
    export STASH_IS_ADMIN="true"
    export STASH_IS_WRITE="true"
else
    export STASH_IS_ADMIN="false"

    if [[ "$BB_USER_PERMISSIONS" == "REPO_WRITE" ]]; then
        export STASH_IS_WRITE="true"
    fi
fi

# end of hook-script.template.bash resource

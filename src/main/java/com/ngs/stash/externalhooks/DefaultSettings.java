package com.ngs.stash.externalhooks;

import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.BRANCH_CREATE;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.BRANCH_DELETE;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.FILE_EDIT;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.PULL_REQUEST_MERGE;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.REPO_PUSH;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.TAG_CREATE;
import static com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger.TAG_DELETE;

import java.util.Arrays;
import java.util.List;

import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;

public class DefaultSettings {
  public static final List<RepositoryHookTrigger> PreReceiveHookTriggers =
      Arrays.asList(REPO_PUSH, FILE_EDIT, TAG_DELETE, TAG_CREATE, BRANCH_DELETE, BRANCH_CREATE);

  public static final List<RepositoryHookTrigger> PostReceiveHookTriggers = Arrays.asList(
      REPO_PUSH,
      FILE_EDIT,
      TAG_DELETE,
      TAG_CREATE,
      BRANCH_DELETE,
      BRANCH_CREATE,
      PULL_REQUEST_MERGE);

  public static final List<RepositoryHookTrigger> MergeCheckHookTriggers =
      Arrays.asList(PULL_REQUEST_MERGE);
}

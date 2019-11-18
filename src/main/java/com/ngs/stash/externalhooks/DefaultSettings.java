package com.ngs.stash.externalhooks;

import java.util.Arrays;
import java.util.List;

import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger;

public class DefaultSettings {
  public static final List<RepositoryHookTrigger> PreReceiveHookTriggers = Arrays.asList(
      StandardRepositoryHookTrigger.REPO_PUSH,
      StandardRepositoryHookTrigger.FILE_EDIT,
      StandardRepositoryHookTrigger.TAG_DELETE,
      StandardRepositoryHookTrigger.TAG_CREATE,
      StandardRepositoryHookTrigger.BRANCH_DELETE,
      StandardRepositoryHookTrigger.BRANCH_CREATE);

  public static final List<RepositoryHookTrigger> PostReceiveHookTriggers = Arrays.asList(
      StandardRepositoryHookTrigger.REPO_PUSH,
      StandardRepositoryHookTrigger.FILE_EDIT,
      StandardRepositoryHookTrigger.TAG_DELETE,
      StandardRepositoryHookTrigger.TAG_CREATE,
      StandardRepositoryHookTrigger.BRANCH_DELETE,
      StandardRepositoryHookTrigger.BRANCH_CREATE,
      StandardRepositoryHookTrigger.PULL_REQUEST_MERGE);

  public static final List<RepositoryHookTrigger> MergeCheckHookTriggers =
      Arrays.asList(StandardRepositoryHookTrigger.PULL_REQUEST_MERGE);
}

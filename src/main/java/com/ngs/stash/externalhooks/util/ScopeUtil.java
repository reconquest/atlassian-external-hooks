package com.ngs.stash.externalhooks.util;

import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.scope.ScopeType;

public class ScopeUtil {
  public static boolean isInheritedEnabled(RepositoryHook hook, Scope scope) {
    return hook.getScope().getType() != scope.getType();
  }

  public static boolean isDirectEnabled(RepositoryHook hook, RepositoryScope scope) {
    return hook.isEnabled() && hook.getScope().getType() == scope.getType();
  }

  public static boolean isRepository(Scope scope) {
    return scope.getType().equals(ScopeType.REPOSITORY);
  }

  public static boolean isProject(Scope scope) {
    return scope.getType().equals(ScopeType.PROJECT);
  }

  public static String toString(Scope scope) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      Repository repository = ((RepositoryScope) scope).getRepository();
      return String.format(
          "repository=%s/%s", repository.getProject().getKey(), repository.getSlug());
    }

    if (scope.getType().equals(ScopeType.PROJECT)) {
      Project project = ((ProjectScope) scope).getProject();
      return String.format("project=%s", project.getKey());
    }

    return "";
  }
}

package com.ngs.stash.externalhooks.util;

import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.user.ApplicationUser;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.bitbucket.user.UserSearchRequest;
import com.atlassian.bitbucket.user.UserService;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.bitbucket.util.PageRequest;
import com.atlassian.bitbucket.util.PageRequestImpl;

public class Walker {
  private RepositoryService repositoryService;
  private ProjectService projectService;
  private UserService userService;
  private SecurityService securityService;

  public Walker(
      SecurityService securityService,
      UserService userService,
      ProjectService projectService,
      RepositoryService repositoryService) {
    this.securityService = securityService;
    this.userService = userService;
    this.projectService = projectService;
    this.repositoryService = repositoryService;
  }

  public void walk(Callback callback) {
    walkProjects(callback);

    walkUsers(callback);
  }

  private void walkProjects(Callback callback) {
    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Project> projects = this.projectService.findAll(page);
      if (projects.getSize() == 0) {
        break;
      }

      projects.stream().forEach(project -> walk(project, callback));

      page = projects.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  private void walkUsers(Callback callback) {
    int start = 0;
    int limit = 10;
    UserSearchRequest searchRequest = (new UserSearchRequest.Builder()).build();
    while (true) {
      //
      PageRequest page = new PageRequestImpl(start, limit);
      Page<ApplicationUser> users = this.userService.search(searchRequest, page);
      if (users.getSize() == 0) {
        break;
      }

      users.stream().forEach((user) -> {
        walk(user, callback);
      });

      PageRequest nextPage = users.getNextPageRequest();
      if (nextPage == null) {
        break;
      }

      start = nextPage.getStart();
      limit = nextPage.getLimit();
    }
  }

  private void walk(ApplicationUser user, Callback callback) {
    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Repository> repos = this.repositoryService.findByOwner(user, page);
      if (repos.getSize() == 0) {
        break;
      }

      repos.stream().forEach(repo -> callback.onRepository(repo));

      page = repos.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  private void walk(Project project, Callback callback) {
    callback.onProject(project);

    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Repository> repos = this.repositoryService.findByProjectId(project.getId(), page);
      if (repos.getSize() == 0) {
        break;
      }

      repos.stream().forEach(repo -> callback.onRepository(repo));

      page = repos.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  public interface Callback {
    void onProject(Project project);

    void onRepository(Repository repository);

    // there is no onUser method because this Callback is expected to be used in combination with
    // Hooks Settings and there is no User Scope for Hooks.
  }
}

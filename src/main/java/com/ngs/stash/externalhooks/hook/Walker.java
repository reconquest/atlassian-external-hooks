package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.bitbucket.util.PageRequest;
import com.atlassian.bitbucket.util.PageRequestImpl;

public class Walker {
  private RepositoryService repositoryService;
  private ProjectService projectService;

  public Walker(ProjectService projectService, RepositoryService repositoryService) {
    this.projectService = projectService;
    this.repositoryService = repositoryService;
  }

  public void walk(Callback callback) {
    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Project> projects = this.projectService.findAll(page);
      if (projects.getSize() == 0) {
        break;
      }

      for (Project project : projects.getValues()) {
        walk(project, callback);
      }

      page = projects.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  public void walk(Project project, Callback callback) {
    callback.onProject(project);

    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Repository> repos = this.repositoryService.findByProjectId(project.getId(), page);
      if (repos.getSize() == 0) {
        break;
      }

      for (Repository repository : repos.getValues()) {
        callback.onRepository(repository);
      }

      page = repos.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  public interface Callback {
    void onProject(Project project);

    void onRepository(Repository repository);
  }
}

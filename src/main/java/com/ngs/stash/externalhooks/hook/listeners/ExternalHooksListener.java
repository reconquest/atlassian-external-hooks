package com.ngs.stash.externalhooks.hook.listeners;

import java.io.IOException;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import javax.inject.Inject;
import javax.inject.Named;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSearchRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookType;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.bitbucket.util.PageRequest;
import com.atlassian.bitbucket.util.PageRequestImpl;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.scheduler.JobRunner;
import com.atlassian.scheduler.JobRunnerRequest;
import com.atlassian.scheduler.JobRunnerResponse;
import com.atlassian.scheduler.SchedulerService;
import com.atlassian.scheduler.SchedulerServiceException;
import com.atlassian.scheduler.config.JobConfig;
import com.atlassian.scheduler.config.JobId;
import com.atlassian.scheduler.config.JobRunnerKey;
import com.atlassian.scheduler.config.Schedule;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.hook.Walker;
import com.ngs.stash.externalhooks.hook.factory.ExternalHooksFactory;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@Named("ExternalHooksListener")
public class ExternalHooksListener implements JobRunner {
  private static Logger log = LoggerFactory.getLogger(ExternalHooksListener.class.getSimpleName());

  private static String statusHookScripts = "internal:hook-scripts";

  private final int jobInterval = 2000;
  private final JobId jobId = JobId.of("external-hooks-enable-job");

  private SchedulerService schedulerService;
  private HookScriptService hookScriptService;
  private RepositoryHookService repositoryHookService;
  private ProjectService projectService;
  private SecurityService securityService;

  private PluginSettings pluginSettings;

  private ExternalHooksFactory factory;
  private Walker walker;

  @Inject
  public ExternalHooksListener(
      @ComponentImport RepositoryService repositoryService,
      @ComponentImport SchedulerService schedulerService,
      @ComponentImport HookScriptService hookScriptService,
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport ProjectService projectService,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport SecurityService securityService,
      @ComponentImport AuthenticationContext authenticationContext,
      @ComponentImport("permissions") PermissionService permissions,
      @ComponentImport PluginLicenseManager pluginLicenseManager,
      @ComponentImport ClusterService clusterService,
      @ComponentImport StorageService storageService)
      throws IOException {
    this.schedulerService = schedulerService;
    this.hookScriptService = hookScriptService;
    this.repositoryHookService = repositoryHookService;
    this.projectService = projectService;
    this.securityService = securityService;

    this.factory = new ExternalHooksFactory(
        repositoryService,
        schedulerService,
        hookScriptService,
        repositoryHookService,
        projectService,
        pluginSettingsFactory,
        securityService,
        authenticationContext,
        permissions,
        pluginLicenseManager,
        clusterService,
        storageService);

    this.walker = new Walker(projectService, repositoryService);

    this.pluginSettings = pluginSettingsFactory.createGlobalSettings();
  }

  @PostConstruct
  public void init() throws SchedulerServiceException {
    log.info("Registering Job for creating HookScripts (plugin enabled / bitbucket restarted)");

    JobRunnerKey runner = JobRunnerKey.of("external-hooks-enable");

    this.schedulerService.registerJobRunner(runner, this);

    this.schedulerService.scheduleJob(this.jobId, JobConfig.forJobRunnerKey(runner)
        .withSchedule(Schedule.forInterval(this.jobInterval, null)));
  }

  @PreDestroy
  public void destroy() {
    int deleted = this.securityService
        .withPermission(Permission.SYS_ADMIN, "External Hook Plugin: Uninstall repo hooks")
        .call(() -> this.hookScriptService.deleteByPluginKey(ExternalHookScript.PLUGIN_KEY));

    log.info("Successfully deleted {} HookScripts", deleted);

    this.setHookScriptsDestroyed();
  }

  public JobRunnerResponse runJob(JobRunnerRequest request) {
    log.info("Started job for creating HookScripts");

    securityService
        .withPermission(Permission.SYS_ADMIN, "External Hook Plugin: creating HookScripts")
        .call(() -> createHookScriptsForEverything());

    return JobRunnerResponse.success();
  }

  protected boolean isPluginLoaded() {
    Page<Project> projects = this.projectService.findAll(new PageRequestImpl(0, 1));
    if (projects.getSize() == 0) {
      // there is nothing to do, no projects means no job to do
      return true;
    }

    RepositoryHookSearchRequest.Builder searchBuilder = new RepositoryHookSearchRequest.Builder(
            new ProjectScope(projects.getValues().iterator().next()))
        .type(RepositoryHookType.PRE_RECEIVE);

    Page<RepositoryHook> page = repositoryHookService.search(
        searchBuilder.build(), new PageRequestImpl(0, PageRequest.MAX_PAGE_LIMIT));

    boolean found = false;
    for (RepositoryHook hook : page.getValues()) {
      if (hook.getDetails().getKey().startsWith(ExternalHookScript.PLUGIN_KEY)) {
        found = true;
        break;
      }
    }

    return found;
  }

  private Object createHookScriptsForEverything() {
    if (!this.isPluginLoaded()) {
      log.warn("Plugin is not yet completely loaded, waiting");
      return null;
    }

    Object created = pluginSettings.get(statusHookScripts);
    if (created == null) {
      walker.walk(new Walker.Callback() {
        @Override
        public void onProject(Project project) {
          factory.install(new ProjectScope(project));
        }

        @Override
        public void onRepository(Repository repository) {
          factory.install(new RepositoryScope(repository));
        }
      });

      this.setHookScriptsCreated();
    } else {
      log.warn("HooksScripts are already created, unscheduling the job");
    }

    this.schedulerService.unscheduleJob(this.jobId);

    return null;
  }

  private void setHookScriptsCreated() {
    pluginSettings.put(statusHookScripts, "created");
    log.warn("HookScripts created successfully");
  }

  private void setHookScriptsDestroyed() {
    try {
      pluginSettings.remove(statusHookScripts);
    } catch (IllegalArgumentException e) {
      log.warn("Plugin was disabled but HookScripts were not created.");
    }
  }
}

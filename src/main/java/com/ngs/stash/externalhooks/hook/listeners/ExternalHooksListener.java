package com.ngs.stash.externalhooks.hook.listeners;

import java.io.IOException;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import javax.inject.Inject;
import javax.inject.Named;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.GetRepositoryHookSettingsRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSearchRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSettings;
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
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
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
import com.ngs.stash.externalhooks.hook.ExternalAsyncPostReceiveHook;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook;
import com.ngs.stash.externalhooks.hook.ExternalPreReceiveHook;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@Named("ExternalHooksListener")
public class ExternalHooksListener implements JobRunner {
  private static Logger log = LoggerFactory.getLogger(ExternalHooksListener.class.getSimpleName());

  private static String statusHookScripts = "internal:hook-scripts";

  private final int jobInterval = 2000;
  private final JobId jobId = JobId.of("external-hooks-enable-job");

  @ComponentImport private RepositoryService repositoryService;
  @ComponentImport private SchedulerService schedulerService;
  @ComponentImport private HookScriptService hookScriptService;
  @ComponentImport private RepositoryHookService repoHookService;
  @ComponentImport private ProjectService projectService;
  @ComponentImport private PluginSettingsFactory pluginSettingsFactory;
  @ComponentImport private SecurityService securityService;
  @ComponentImport private AuthenticationContext authenticationContext;

  @ComponentImport("permissions")
  private PermissionService permissions;

  @ComponentImport private PluginLicenseManager pluginLicenseManager;

  @ComponentImport private ClusterService clusterService;
  @ComponentImport private StorageService storageProperties;

  private PluginSettings pluginSettings;

  private ExternalHookScript hookPreReceive;
  private ExternalHookScript hookPostReceive;
  private ExternalHookScript hookMergeCheck;

  @Inject
  public ExternalHooksListener(
      RepositoryService repositoryService,
      SchedulerService schedulerService,
      HookScriptService hookScriptService,
      RepositoryHookService repoHookService,
      ProjectService projectService,
      PluginSettingsFactory pluginSettingsFactory,
      SecurityService securityService,
      AuthenticationContext authenticationContext,
      PermissionService permissions,
      PluginLicenseManager pluginLicenseManager,
      ClusterService clusterService,
      StorageService storageProperties)
      throws IOException {
    this.repositoryService = repositoryService;
    this.schedulerService = schedulerService;
    this.hookScriptService = hookScriptService;
    this.repoHookService = repoHookService;
    this.projectService = projectService;
    this.pluginSettingsFactory = pluginSettingsFactory;
    this.securityService = securityService;
    this.authenticationContext = authenticationContext;
    this.permissions = permissions;
    this.pluginLicenseManager = pluginLicenseManager;
    this.clusterService = clusterService;
    this.storageProperties = storageProperties;

    this.pluginSettings = pluginSettingsFactory.createGlobalSettings();

    this.hookPreReceive = ExternalPreReceiveHook.getExternalHookScript(
        authenticationContext,
        permissions,
        pluginLicenseManager,
        clusterService,
        storageProperties,
        hookScriptService,
        pluginSettingsFactory,
        securityService);

    this.hookPostReceive = ExternalAsyncPostReceiveHook.getExternalHookScript(
        authenticationContext,
        permissions,
        pluginLicenseManager,
        clusterService,
        storageProperties,
        hookScriptService,
        pluginSettingsFactory,
        securityService);

    this.hookMergeCheck = ExternalMergeCheckHook.getExternalHookScript(
        authenticationContext,
        permissions,
        pluginLicenseManager,
        clusterService,
        storageProperties,
        hookScriptService,
        pluginSettingsFactory,
        securityService);
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

    Page<RepositoryHook> page = repoHookService.search(
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
      this.createHookScriptsForProjects();
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

  protected void createHookScriptsForProjects() {
    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Project> projects = this.projectService.findAll(page);
      if (projects.getSize() == 0) {
        break;
      }

      for (Project project : projects.getValues()) {
        this.createHookScripts(new ProjectScope(project));

        this.createHookScriptsForRepositories(project);
      }

      page = projects.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  protected void createHookScriptsForRepositories(Project project) {
    PageRequest page = new PageRequestImpl(0, 10);

    while (true) {
      Page<Repository> repos = this.repositoryService.findByProjectId(project.getId(), page);
      if (repos.getSize() == 0) {
        break;
      }

      for (Repository repository : repos.getValues()) {
        this.createHookScripts(new RepositoryScope(repository));
      }

      page = repos.getNextPageRequest();
      if (page == null) {
        break;
      }
    }
  }

  protected void createHookScripts(Scope scope) {
    log.warn("Creating HookScripts in scope: {}", scope.toString());

    RepositoryHookSearchRequest.Builder searchBuilder =
        new RepositoryHookSearchRequest.Builder(scope);

    Page<RepositoryHook> page = repoHookService.search(
        searchBuilder.build(), new PageRequestImpl(0, PageRequest.MAX_PAGE_LIMIT));

    Integer created = 0;
    for (RepositoryHook hook : page.getValues()) {
      String hookKey = hook.getDetails().getKey();
      if (!hookKey.startsWith(ExternalHookScript.PLUGIN_KEY)) {
        continue;
      }

      if (!hook.isEnabled()) {
        continue;
      }

      if (!hook.isConfigured()) {
        continue;
      }

      if (hook.getScope().getType() != scope.getType()) {
        log.warn(
            "Hook {} is enabled & configured (inherited: {} {})",
            hookKey,
            hook.getScope().getType(),
            hook.getScope().getResourceId().orElse(-1));
        continue;
      }

      GetRepositoryHookSettingsRequest.Builder getSettingsBuilder =
          new GetRepositoryHookSettingsRequest.Builder(scope, hookKey);

      RepositoryHookSettings hookSettings =
          this.repoHookService.getSettings(getSettingsBuilder.build());

      if (hookSettings == null) {
        log.warn("Hook {} has no settings, can't be enabled", hookKey);
        return;
      }

      Settings settings = hookSettings.getSettings();

      try {
        if (hookKey.equals(hookPreReceive.getHookKey())) {
          log.warn("Creating PRE_RECEIVE HookScript for {}", hookKey);
          this.hookPreReceive.install(settings, scope);
        } else if (hookKey.equals(hookPostReceive.getHookKey())) {
          log.warn("Creating POST_RECEIVE HookScript for {}", hookKey);
          this.hookPostReceive.install(settings, scope);
        } else if (hookKey.equals(hookMergeCheck.getHookKey())) {
          log.warn("Creating MERGE_CHECK HookScript for {}", hookKey);
          this.hookMergeCheck.install(settings, scope);
        } else {
          log.warn("Unexpected hook key: {}", hookKey);
        }

        created++;
      } catch (Exception e) {
        log.error("Unable to install hook script {}: {}", hookKey, e.toString());
      }
    }

    log.warn("Created {} HookScripts in scope: {}", created, scope.toString());
  }
}

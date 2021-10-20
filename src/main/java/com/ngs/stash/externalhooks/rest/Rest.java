package com.ngs.stash.externalhooks.rest;

import java.io.IOException;
import java.io.Serializable;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.Consumes;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.GlobalScope;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsBuilder;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.bitbucket.user.UserService;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.scheduler.JobRunner;
import com.atlassian.scheduler.JobRunnerRequest;
import com.atlassian.scheduler.JobRunnerResponse;
import com.atlassian.scheduler.SchedulerService;
import com.atlassian.scheduler.SchedulerServiceException;
import com.atlassian.scheduler.config.JobConfig;
import com.atlassian.scheduler.config.JobId;
import com.atlassian.scheduler.config.JobRunnerKey;
import com.atlassian.scheduler.config.RunMode;
import com.atlassian.scheduler.config.Schedule;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.ExternalHooksSettings;
import com.ngs.stash.externalhooks.GlobalHooks;
import com.ngs.stash.externalhooks.HookInstaller;
import com.ngs.stash.externalhooks.HooksFactory;
import com.ngs.stash.externalhooks.SimpleSettingsBuilder;
import com.ngs.stash.externalhooks.SimpleSettingsValidationErrors;
import com.ngs.stash.externalhooks.ao.FactoryState;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;
import com.ngs.stash.externalhooks.dao.ExternalHooksSettingsDao;
import com.ngs.stash.externalhooks.dao.FactoryStateDao;
import com.ngs.stash.externalhooks.dao.GlobalHookSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.Walker;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.atlassian.util.concurrent.atomic.AtomicInteger;

@Path("/")
public class Rest implements JobRunner {
  private static final Logger log = LoggerFactory.getLogger(Rest.class);

  private SchedulerService schedulerService;
  private PermissionService permissionService;
  private SecurityService securityService;

  private FactoryStateDao factoryStateDao;
  private ExternalHooksSettingsDao settingsDao;
  private Walker walker;
  private GlobalHookSettingsDao globalHookSettingsDao;
  // private RepositoryHookService repositoryHookService;
  private HookInstaller hookInstaller;
  private HooksFactory hooksFactory;

  public Rest(
      @ComponentImport AuthenticationContext authenticationContext,
      @ComponentImport GlobalHookSettingsDao globalHookSettingsDao,
      @ComponentImport HookInstaller hookInstaller,
      @ComponentImport UserService userService,
      @ComponentImport ActiveObjects ao,
      @ComponentImport RepositoryService repositoryService,
      @ComponentImport SchedulerService schedulerService,
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport ProjectService projectService,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport HookScriptService hookScriptService,
      @ComponentImport PluginLicenseManager pluginLicenseManager,
      @ComponentImport ClusterService clusterService,
      @ComponentImport SecurityService securityService,
      @ComponentImport("permissions") PermissionService permissionService,
      @ComponentImport StorageService storageService)
      throws IOException {
    this.globalHookSettingsDao = globalHookSettingsDao;
    this.permissionService = permissionService;
    this.schedulerService = schedulerService;
    this.securityService = securityService;
    // this.repositoryHookService = repositoryHookService;
    this.hookInstaller = hookInstaller;
    //
    // Unfortunately, no way to @ComponentImport it because Named() used here.
    // Consider it to replace with lifecycle aware listener.
    this.hooksFactory = new HooksFactory(
        repositoryHookService,
        new HookInstaller(
            userService,
            projectService,
            repositoryService,
            repositoryHookService,
            authenticationContext,
            permissionService,
            pluginLicenseManager,
            clusterService,
            storageService,
            hookScriptService,
            pluginSettingsFactory,
            securityService));

    this.settingsDao = new ExternalHooksSettingsDao(pluginSettingsFactory);

    this.factoryStateDao = new FactoryStateDao(ao);

    this.walker = new Walker(userService, projectService, repositoryService);
  }

  private boolean isSystemAdmin() {
    return permissionService.hasGlobalPermission(Permission.SYS_ADMIN);
  }

  @GET
  @Produces({MediaType.APPLICATION_JSON})
  @Path("/settings")
  public Response getSettings() {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    return Response.ok(settingsDao.getSettings()).build();
  }

  @GET
  @Produces({MediaType.APPLICATION_JSON})
  @Path("/settings/default")
  public Response getDefaultSettings() {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    return Response.ok(settingsDao.getDefaultSettings()).build();
  }

  @PUT
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  @Path("/settings")
  public Response updateSettings(ExternalHooksSettings settings) {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    settingsDao.save(settings);

    return Response.ok().build();
  }

  @GET
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  @Path("/factory/state/{id}")
  public Response getFactoryState(@PathParam("id") Integer id) {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    FactoryState state = factoryStateDao.find(id);

    return Response.ok(new FactoryStateResponse(state)).build();
  }

  @POST
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  @Path("/factory/hooks")
  public Response applySettings() {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    FactoryState state = factoryStateDao.create();

    scheduleCreatingHooks(state.getID());

    return Response.ok(new FactoryStateResponse(state.getID())).build();
  }

  @PUT
  @Path("/global-hooks/{hookKey}")
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  public Response putGlobalHookSettings(
      @PathParam("hookKey") String hookKey, GlobalHookSettingsSchema schema) {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    if (!hookKey.startsWith(Const.PLUGIN_KEY)) {
      return Response.status(404).build();
    }

    if (schema.enabled) {
      ExternalHookScript script = this.hookInstaller.getScript(hookKey);
      if (script == null) {
        return Response.status(404).build();
      }

      SettingsBuilder settingsBuilder = new SimpleSettingsBuilder();
      settingsBuilder.add("safe_path", schema.safePath);
      settingsBuilder.add("async", schema.async);
      if (schema.exe != null) {
        settingsBuilder.add("exe", schema.exe);
      }
      if (schema.params != null) {
        settingsBuilder.add("params", schema.params);
      }

      Settings scriptSettings = settingsBuilder.build();

      SimpleSettingsValidationErrors errors = new SimpleSettingsValidationErrors();
      script.validate(scriptSettings, errors, new GlobalScope());

      if (!errors.isEmpty()) {
        return Response.ok(new FormValidationErrors(errors)).build();
      }
    }

    GlobalHookSettings settings = this.globalHookSettingsDao.get(hookKey);
    if (settings == null) {
      settings = this.globalHookSettingsDao.create();
      settings.setHook(hookKey);
    }

    settings.setExe(schema.exe);
    settings.setParams(schema.params);
    settings.setSafePath(schema.safePath);
    settings.setAsync(schema.async);

    settings.setEnabled(schema.enabled);
    settings.setFilterPersonalRepositories(schema.filterPersonalRepositories);

    settings.save();

    return Response.ok(new HashMap<String, String>()).build();
  }

  @GET
  @Path("/global-hooks/{hookKey}")
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  public Response getGlobalHookSettings(@PathParam("hookKey") String hookKey) {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    GlobalHookSettings settings = this.globalHookSettingsDao.get(hookKey);
    GlobalHookSettingsSchema schema = new GlobalHookSettingsSchema();
    if (settings == null) {
      return Response.ok(schema).build();
    }
    schema.safePath = settings.getSafePath();
    schema.exe = settings.getExe();
    schema.params = settings.getParams();
    schema.async = settings.getAsync();
    schema.enabled = settings.getEnabled();
    schema.filterPersonalRepositories = settings.getFilterPersonalRepositories();
    return Response.ok(schema).build();
  }

  private void scheduleCreatingHooks(int stateId) {
    JobRunnerKey runner = JobRunnerKey.of("external-hooks-factory-runner");
    JobId id = JobId.of("external-hooks-factory-job");

    this.schedulerService.registerJobRunner(runner, this);

    Map<String, Serializable> parameters = new HashMap<String, Serializable>();
    parameters.put("state_id", stateId);

    JobConfig job = JobConfig.forJobRunnerKey(runner)
        .withSchedule(Schedule.runOnce(new Date()))
        .withRunMode(RunMode.RUN_ONCE_PER_CLUSTER)
        .withParameters(parameters);

    try {
      this.schedulerService.scheduleJob(id, job);
    } catch (SchedulerServiceException e) {
      log.error("Unable to schedule re-creating External Hooks", e);
    }
  }

  public JobRunnerResponse runJob(JobRunnerRequest request) {
    JobConfig config = request.getJobConfig();
    Map<String, Serializable> parameters = config.getParameters();
    int stateId = (int) parameters.get("state_id");

    FactoryState state = factoryStateDao.find(Integer.valueOf(stateId));
    if (state == null) {
      log.error("scheduled factory state not found: {}", stateId);
      return JobRunnerResponse.failed("scheduled factory state not found");
    }

    securityService
        .withPermission(Permission.SYS_ADMIN, "External Hook Factory: create hooks")
        .call(() -> {
          createHooks(state);
          return null;
        });

    return JobRunnerResponse.success();
  }

  private void createHooks(FactoryState state) {
    GlobalHooks globalHooks = new GlobalHooks(globalHookSettingsDao.find());
    state.setStarted(true);
    state.save();

    AtomicInteger total = new AtomicInteger();

    walker.walk(new Walker.Callback() {
      @Override
      public void onProject(Project project) {
        total.incrementAndGet();
      }

      @Override
      public void onRepository(Repository repository) {
        total.incrementAndGet();
      }
    });

    state.setTotal(total.get());
    state.save();

    // adding a small delay in order to spread the cpu/io load if bb instance
    // has a lot of hooks installed
    int millisDelay = 10;

    AtomicInteger current = new AtomicInteger();
    walker.walk(new Walker.Callback() {
      @Override
      public void onProject(Project project) {
        hooksFactory.apply(new ProjectScope(project), globalHooks);

        state.setCurrent(current.incrementAndGet());
        state.save();

        delay(millisDelay);
      }

      @Override
      public void onRepository(Repository repository) {
        hooksFactory.apply(new RepositoryScope(repository), globalHooks);

        state.setCurrent(current.incrementAndGet());
        state.save();

        delay(millisDelay);
      }
    });

    state.setFinished(true);
    state.save();
  }

  private void delay(int ms) {
    try {
      Thread.sleep(ms);
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
    }
  }
}

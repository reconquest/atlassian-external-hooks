package com.ngs.stash.externalhooks.rest;

import java.io.IOException;
import java.io.Serializable;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;

import javax.inject.Inject;
import javax.ws.rs.Consumes;
import javax.ws.rs.GET;
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
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.plugin.spring.scanner.annotation.component.Scanned;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.sal.api.user.UserManager;
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
import com.ngs.stash.externalhooks.ExternalHooksSettings;
import com.ngs.stash.externalhooks.ao.FactoryState;
import com.ngs.stash.externalhooks.dao.FactoryStateDao;
import com.ngs.stash.externalhooks.hook.factory.ExternalHooksFactory;
import com.ngs.stash.externalhooks.util.Walker;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.atlassian.util.concurrent.atomic.AtomicInteger;

@Path("/")
@Scanned
public class Rest implements JobRunner {
  private static final Logger log = LoggerFactory.getLogger(Rest.class);

  private UserManager userManager;

  private SchedulerService schedulerService;
  private AuthenticationContext authenticationContext;
  private PermissionService permissionService;
  private RepositoryService repositoryService;
  private ProjectService projectService;
  private SecurityService securityService;
  private PluginSettingsFactory pluginSettingsFactory;

  private ExternalHooksFactory factory;
  private FactoryStateDao factoryStateDao;

  @Inject
  public Rest(
      @ComponentImport ActiveObjects ao,
      @ComponentImport UserManager userManager,
      @ComponentImport RepositoryService repositoryService,
      @ComponentImport SchedulerService schedulerService,
      @ComponentImport HookScriptService hookScriptService,
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport ProjectService projectService,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport SecurityService securityService,
      @ComponentImport AuthenticationContext authenticationContext,
      @ComponentImport("permissions") PermissionService permissionService,
      @ComponentImport PluginLicenseManager pluginLicenseManager,
      @ComponentImport ClusterService clusterService,
      @ComponentImport StorageService storageService)
      throws IOException {
    this.permissionService = permissionService;
    this.projectService = projectService;
    this.repositoryService = repositoryService;
    this.schedulerService = schedulerService;
    this.securityService = securityService;
    this.pluginSettingsFactory = pluginSettingsFactory;

    this.userManager = userManager;

    this.factory = new ExternalHooksFactory(
        repositoryService,
        schedulerService,
        hookScriptService,
        repositoryHookService,
        projectService,
        pluginSettingsFactory,
        securityService,
        authenticationContext,
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService);

    this.factoryStateDao = new FactoryStateDao(ao);
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

    ExternalHooksSettings settings = new ExternalHooksSettings(pluginSettingsFactory);

    return Response.ok(settings).build();
  }

  @PUT
  @Produces({MediaType.APPLICATION_JSON})
  @Consumes({MediaType.APPLICATION_JSON})
  @Path("/settings")
  public Response updateSettings(ExternalHooksSettings settings) {
    if (!isSystemAdmin()) {
      return Response.status(401).build();
    }

    settings.save(this.pluginSettingsFactory);

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

  @GET
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

  private void scheduleCreatingHooks(int stateId) {
    JobRunnerKey runner = JobRunnerKey.of("external-hooks-factory-runner");
    JobId id = JobId.of("external-hooks-factory-job");

    this.schedulerService.registerJobRunner(runner, this);

    Map<String, Serializable> parameters = new HashMap<String, Serializable>();
    parameters.put("state_id", stateId);

    JobConfig job = JobConfig.forJobRunnerKey(runner)
        .withSchedule(Schedule.runOnce(new Date()))
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
    state.setStarted(true);
    state.save();

    AtomicInteger total = new AtomicInteger();

    Walker walker = new Walker(projectService, repositoryService);
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

    AtomicInteger current = new AtomicInteger();
    walker.walk(new Walker.Callback() {
      @Override
      public void onProject(Project project) {
        factory.install(new ProjectScope(project));
        // uncomment for debugging purposes
        // sleep();

        state.setCurrent(current.incrementAndGet());
        state.save();
      }

      @Override
      public void onRepository(Repository repository) {
        factory.install(new RepositoryScope(repository));
        // uncomment for debugging purposes
        // sleep();

        state.setCurrent(current.incrementAndGet());
        state.save();
      }
    });

    state.setFinished(true);
    state.save();
  }

  @SuppressWarnings("unused")
  private void sleep() {
    try {
      Thread.sleep(100);
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
    }
  }
}

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
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
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
import com.atlassian.scheduler.config.Schedule;
import com.ngs.stash.externalhooks.ExternalHooksSettings;
import com.ngs.stash.externalhooks.HooksCoordinator;
import com.ngs.stash.externalhooks.HooksFactory;
import com.ngs.stash.externalhooks.ao.FactoryState;
import com.ngs.stash.externalhooks.dao.ExternalHooksSettingsDao;
import com.ngs.stash.externalhooks.dao.FactoryStateDao;
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
  private HooksFactory hooksFactory;
  private Walker walker;

  public Rest(
      @ComponentImport HooksFactory hooksFactory,
      @ComponentImport HooksCoordinator hooksCoordinator,
      @ComponentImport UserService userService,
      @ComponentImport ActiveObjects ao,
      @ComponentImport RepositoryService repositoryService,
      @ComponentImport SchedulerService schedulerService,
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport ProjectService projectService,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport SecurityService securityService,
      @ComponentImport("permissions") PermissionService permissionService)
      throws IOException {
    this.hooksFactory = hooksFactory;
    this.permissionService = permissionService;
    this.schedulerService = schedulerService;
    this.securityService = securityService;

    this.settingsDao = new ExternalHooksSettingsDao(pluginSettingsFactory);

    this.factoryStateDao = new FactoryStateDao(ao);

    this.walker = new Walker(securityService, userService, projectService, repositoryService);
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
        hooksFactory.install(new ProjectScope(project));

        state.setCurrent(current.incrementAndGet());
        state.save();

        delay(millisDelay);
      }

      @Override
      public void onRepository(Repository repository) {
        hooksFactory.install(new RepositoryScope(repository));

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

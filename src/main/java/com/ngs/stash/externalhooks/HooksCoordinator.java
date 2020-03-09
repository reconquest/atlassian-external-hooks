package com.ngs.stash.externalhooks;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookEnabledEvent;
import com.atlassian.bitbucket.event.repository.RepositoryCreatedEvent;
import com.atlassian.bitbucket.hook.repository.GetRepositoryHookSettingsRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSettings;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.project.ProjectType;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.scope.ScopeType;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.bitbucket.user.UserService;
import com.atlassian.event.api.EventListener;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.ngs.stash.externalhooks.dao.ExternalHooksSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.ScopeUtil;
import com.ngs.stash.externalhooks.util.Walker;

public class HooksCoordinator {
  private RepositoryHookService repositoryHookService;

  private Map<String, ExternalHookScript> scripts = new HashMap<>();
  private Walker walker;

  public HooksCoordinator(
      @ComponentImport UserService userService,
      @ComponentImport ProjectService projectService,
      @ComponentImport RepositoryService repositoryService,
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport AuthenticationContext authenticationContext,
      @ComponentImport("permissions") PermissionService permissionService,
      @ComponentImport PluginLicenseManager pluginLicenseManager,
      @ComponentImport ClusterService clusterService,
      @ComponentImport StorageService storageService,
      @ComponentImport HookScriptService hookScriptService,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport SecurityService securityService)
      throws IOException {
    this.repositoryHookService = repositoryHookService;

    ExternalHooksSettingsDao settingsDao = new ExternalHooksSettingsDao(pluginSettingsFactory);

    this.scripts.put(Const.PRE_RECEIVE_HOOK_ID, new ExternalHookScript(
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService,
        Const.PRE_RECEIVE_HOOK_ID,
        HookScriptType.PRE,
        () -> settingsDao.getPreReceiveHookTriggers()));

    this.scripts.put(Const.POST_RECEIVE_HOOK_ID, new ExternalHookScript(
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService,
        Const.POST_RECEIVE_HOOK_ID,
        HookScriptType.POST,
        () -> settingsDao.getPostReceiveHookTriggers()));

    this.scripts.put(Const.MERGE_CHECK_HOOK_ID, new ExternalHookScript(
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService,
        Const.MERGE_CHECK_HOOK_ID,
        HookScriptType.PRE,
        () -> settingsDao.getMergeCheckHookTriggers()));

    this.walker = new Walker(securityService, userService, projectService, repositoryService);
  }

  private ExternalHookScript getScript(String idOrKey) {
    if (idOrKey.startsWith(Const.PLUGIN_KEY)) {
      // +1 stands for : after plugin key
      return scripts.get(idOrKey.substring(Const.PLUGIN_KEY.length() + 1));
    }

    return scripts.get(idOrKey);
  }

  @EventListener
  public void onHookEnabled(RepositoryHookEnabledEvent event) {
    ExternalHookScript script = getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    enable(event.getScope(), script);
  }

  @EventListener
  public void onHookDisabled(RepositoryHookDisabledEvent event) {
    ExternalHookScript script = getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      disable((RepositoryScope) scope, script);
    } else if (ScopeUtil.isProject(scope)) {
      disable((ProjectScope) scope, script);
    }
  }

  // This event is triggered when repository hook transfered from 'Enabled' to
  // 'Inherited' state. It means that the repository doesn't have any its own
  // hook, but might have project's hook.
  //
  // Also, triggered when the state changed from 'Disabled' to 'Inherited'
  @EventListener
  public void onHookInherited(RepositoryHookDeletedEvent event) {
    ExternalHookScript script = getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      inherit((RepositoryScope) scope, script);
    }
  }

  @EventListener
  public void onRepositoryCreated(RepositoryCreatedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    scripts.forEach((hookId, script) -> {
      inherit(scope, script);
    });
  }

  public void validate(
      @Nonnull String hookId,
      @Nonnull Settings settings,
      @Nonnull SettingsValidationErrors errors,
      @Nonnull Scope scope) {
    ExternalHookScript script = scripts.get(hookId);
    if (script == null) {
      return;
    }

    script.validate(settings, errors, scope);
  }

  public void enable(Scope scope, String hookKey) {
    ExternalHookScript script = getScript(hookKey);
    if (script == null) {
      return;
    }

    enable(scope, script);
  }

  public void enable(Scope scope, ExternalHookScript script) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      enable((RepositoryScope) scope, script);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      enable((ProjectScope) scope, script);
    } else {
      // Do nothing with ScopeType.GLOBAL
    }
  }

  public void enable(ProjectScope scope, ExternalHookScript script) {
    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(scope);

    GetRepositoryHookSettingsRequest request =
        (new GetRepositoryHookSettingsRequest.Builder(scope, script.getHookKey())).build();

    RepositoryHookSettings projectSettings = repositoryHookService.getSettings(request);
    if (projectSettings == null) {
      return;
    }

    walker.walk(scope.getProject(), (repository) -> {
      RepositoryScope repositoryScope = new RepositoryScope(repository);
      RepositoryHook hook = repositoryHookService.getByKey(repositoryScope, script.getHookKey());
      //
      // isEnabled also covers 'inherited' case
      //
      if (hook.isEnabled()) {
        script.install(projectSettings.getSettings(), scope, repositoryScope);
      }
    });
  }

  public void enable(RepositoryScope scope, ExternalHookScript script) {
    GetRepositoryHookSettingsRequest request =
        (new GetRepositoryHookSettingsRequest.Builder(scope, script.getHookKey())).build();

    RepositoryHookSettings settings = repositoryHookService.getSettings(request);
    script.install(settings.getSettings(), scope);
  }

  public void disable(ProjectScope scope, ExternalHookScript script) {
    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(scope);

    walker.walk(scope.getProject(), (repository) -> {
      // RepositoryHook.isEnabled returns true when hook is in state 'enabled (inherited)'
      script.uninstall(scope, new RepositoryScope(repository));
    });
  }

  public void disable(RepositoryScope scope, ExternalHookScript script) {
    script.uninstall(scope);

    if (scope.getProject().getType().equals(ProjectType.PERSONAL)) {
      return;
    }

    ProjectScope projectScope = new ProjectScope(scope.getProject());
    RepositoryHook projectHook = repositoryHookService.getByKey(projectScope, script.getHookKey());
    if (projectHook.isEnabled()) {
      script.uninstall(projectScope, scope);
    }
  }

  public void inherit(RepositoryScope scope, ExternalHookScript script) {
    // here is a problem: Inherited state can be obtained by two ways:
    // * enabled → inherited which means we need to disable repository scope and
    //      just don't touch project hook, it's already here
    // * disabled → inherited which means the project/repository hooks were
    //    completely disabled and now only project scope hook should be turned
    //    on. The problem is that RepositoryHook.isEnabled() already returns
    //    true at this point.

    disable(scope, script);

    ProjectScope projectScope = new ProjectScope(scope.getProject());
    RepositoryHook projectHook = repositoryHookService.getByKey(projectScope, script.getHookKey());
    if (projectHook.isEnabled()) {
      GetRepositoryHookSettingsRequest request =
          (new GetRepositoryHookSettingsRequest.Builder(scope, script.getHookKey())).build();
      RepositoryHookSettings projectSettings = repositoryHookService.getSettings(request);
      if (projectSettings == null) {
        return;
      }

      script.install(projectSettings.getSettings(), projectScope, scope);
    }
  }
}

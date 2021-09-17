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
import com.atlassian.bitbucket.event.repository.RepositoryDeletedEvent;
import com.atlassian.bitbucket.hook.repository.GetRepositoryHookSettingsRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSettings;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.ProjectService;
import com.atlassian.bitbucket.project.ProjectType;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.GlobalScope;
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
import com.ngs.stash.externalhooks.dao.GlobalHookSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.ScopeUtil;
import com.ngs.stash.externalhooks.util.Walker;

public class HooksCoordinator {
  private RepositoryHookService repositoryHookService;

  private Map<String, ExternalHookScript> scripts = new HashMap<>();
  private Walker walker;
  private SecurityService securityService;
  private GlobalHookSettingsDao globalHookSettingsDao;

  public HooksCoordinator(
      @ComponentImport GlobalHookSettingsDao globalHookSettingsDao,
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
    this.globalHookSettingsDao = globalHookSettingsDao;
    this.repositoryHookService = repositoryHookService;
    this.securityService = securityService;

    ExternalHooksSettingsDao settingsDao = new ExternalHooksSettingsDao(pluginSettingsFactory);

    this.scripts.put(
        Const.PRE_RECEIVE_HOOK_ID,
        new ExternalHookScript(
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

    this.scripts.put(
        Const.POST_RECEIVE_HOOK_ID,
        new ExternalHookScript(
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

    this.scripts.put(
        Const.MERGE_CHECK_HOOK_ID,
        new ExternalHookScript(
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

    this.walker = new Walker(userService, projectService, repositoryService);
  }

  public ExternalHookScript getScript(String idOrKey) {
    if (idOrKey.startsWith(Const.PLUGIN_KEY)) {
      // +1 stands for : after plugin key
      return scripts.get(idOrKey.substring(Const.PLUGIN_KEY.length() + 1));
    }

    return scripts.get(idOrKey);
  }

  @EventListener
  public void onHookEnabled(RepositoryHookEnabledEvent event) {
    ExternalHookScript script = getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    enable(event.getScope(), script);

    GlobalHooks globalHooks = new GlobalHooks(this.globalHookSettingsDao.find());
    if (globalHooks.isEnabled(script.getHookKey())) {
      enable(event.getScope(), script, globalHooks.getSettings(script.getHookKey()));
    }
  }

  @EventListener
  public void onHookDisabled(RepositoryHookDisabledEvent event) {
    ExternalHookScript script = getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
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
    GlobalHooks globalHooks = new GlobalHooks(this.globalHookSettingsDao.find());
    scripts.forEach((hookId, script) -> {
      inherit(scope, script);

      if (globalHooks.isEnabled(script.getHookKey())) {
        enable(scope, script, globalHooks.getSettings(script.getHookKey()));
      }
    });
  }

  @EventListener
  public void onRepositoryDeleted(RepositoryDeletedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    scripts.forEach((hookId, script) -> {
      script.uninstall(scope);
      script.uninstall(new GlobalScope(), scope);
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

  public void enable(Scope scope, String hookKey, Settings globalSettings) {
    ExternalHookScript script = getScript(hookKey);
    if (script == null) {
      return;
    }

    enable(scope, script, globalSettings);
  }

  public void disable(Scope scope, String hookKey, GlobalScope globalScope) {
    ExternalHookScript script = getScript(hookKey);
    if (script == null) {
      return;
    }

    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      disable((RepositoryScope) scope, script, globalScope);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      disable((ProjectScope) scope, script, globalScope);
    }
  }

  public void enable(Scope scope, ExternalHookScript script) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      enable((RepositoryScope) scope, script);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      enable((ProjectScope) scope, script);
    } else {
      // Do nothing with ScopeType.GLOBAL
      //
      // ScopeType.GLOBAL shall not be here since there are no global hooks
      // actually
    }
  }

  public void enable(Scope scope, ExternalHookScript script, Settings globalSettings) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      enable((RepositoryScope) scope, script, globalSettings);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      enable((ProjectScope) scope, script, globalSettings);
    } else {
      // Do nothing with ScopeType.GLOBAL
      //
      // ScopeType.GLOBAL shall not be here since there are no global hooks
      // actually
    }
  }

  public void enable(ProjectScope projectScope, ExternalHookScript script) {
    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(projectScope);

    RepositoryHookSettings projectSettings = repositoryHookService.getSettings(
        (new GetRepositoryHookSettingsRequest.Builder(projectScope, script.getHookKey())).build());
    if (projectSettings == null) {
      return;
    }

    walker.walk(projectScope.getProject(), (repository) -> {
      RepositoryScope repositoryScope = new RepositoryScope(repository);
      // repositoryHookService.getByKey will return project wide's hook but with
      // projectScope.getType() = PROJECT
      RepositoryHook hook = repositoryHookService.getByKey(repositoryScope, script.getHookKey());
      //
      // isEnabled also covers 'inherited' case
      //
      if (ScopeUtil.isInheritedEnabled(hook, repositoryScope)) {
        //  uninstall on repository level
        script.uninstall(repositoryScope);

        script.install(projectSettings.getSettings(), projectScope, repositoryScope);
      }
    });
  }

  public void enable(
      ProjectScope projectScope, ExternalHookScript script, Settings globalSettings) {
    if (globalSettings == null) {
      throw new RuntimeException("empty settings for " + script.getHookKey());
    }

    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(projectScope);

    walker.walk(projectScope.getProject(), (repository) -> {
      RepositoryScope repositoryScope = new RepositoryScope(repository);
      script.install(globalSettings, new GlobalScope(), repositoryScope);
    });
  }

  public void enable(RepositoryScope scope, ExternalHookScript script) {
    GetRepositoryHookSettingsRequest request =
        (new GetRepositoryHookSettingsRequest.Builder(scope, script.getHookKey())).build();

    RepositoryHookSettings settings = repositoryHookService.getSettings(request);
    script.install(settings.getSettings(), scope);

    securityService
        .withPermission(
            Permission.PROJECT_ADMIN,
            scope.getProject(),
            "atlassian-external-hooks: look for project hook")
        .call(() -> {
          // Disable project-wide hook on this specified repository because 'enabled'
          // hook on Repository means that it's overwritten and two hooks at the same
          // time is not obvious for customers
          script.uninstall(new ProjectScope(scope.getProject()), scope);

          return null;
        });
  }

  public void enable(RepositoryScope scope, ExternalHookScript script, Settings globalSettings) {
    if (globalSettings == null) {
      throw new RuntimeException("empty settings for " + script.getHookKey());
    }

    script.install(globalSettings, new GlobalScope(), scope);
  }

  public void disable(ProjectScope _projetScope, ExternalHookScript _script, GlobalScope _hooks) {
    // this is kind of a dirty hack but we don't really need to uninstall hooks
    // like we do it in enable() method since we know that this method is called
    // by Walker which will also call HooksCoordinator with RepositoryScope (for
    // each repository) and we don't have project-wide hook scripts (they are
    // repository-wide).
    return;
  }

  public void disable(RepositoryScope scope, ExternalHookScript script, GlobalScope _hooks) {
    script.uninstall(new GlobalScope(), scope);
    return;
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

    // The user might not have ADMIN privileges to the project while having
    // ADMIN privileges on the repository
    securityService
        .withPermission(
            Permission.PROJECT_ADMIN,
            scope.getProject(),
            "atlassian-external-hooks: look for project hook")
        .call(() -> {
          ProjectScope projectScope = new ProjectScope(scope.getProject());
          RepositoryHook projectHook =
              repositoryHookService.getByKey(projectScope, script.getHookKey());
          if (projectHook.isEnabled()) {
            script.uninstall(projectScope, scope);
          }

          return null;
        });
  }

  public void inherit(RepositoryScope scope, ExternalHookScript script) {
    // here is a problem: Inherited state can be obtained by two ways:
    // * enabled → inherited which means we need to disable repository scope and
    //      just don't touch project hook, it's already here
    // * disabled → inherited which means the project/repository hooks were
    //    completely disabled and now only project scope hook should be turned
    //    on. The problem is that RepositoryHook.isEnabled() already returns
    //    true at this point.
    if (scope.getProject().getType().equals(ProjectType.PERSONAL)) {
      return;
    }

    disable(scope, script);

    // The user might not have ADMIN privileges to the project while having
    // ADMIN privileges on the repository
    securityService
        .withPermission(
            Permission.PROJECT_ADMIN,
            scope.getProject(),
            "atlassian-external-hooks: look for project hook")
        .call(() -> {
          ProjectScope projectScope = new ProjectScope(scope.getProject());
          RepositoryHook projectHook =
              repositoryHookService.getByKey(projectScope, script.getHookKey());
          if (projectHook.isEnabled()) {
            GetRepositoryHookSettingsRequest request =
                (new GetRepositoryHookSettingsRequest.Builder(scope, script.getHookKey())).build();
            RepositoryHookSettings projectSettings = repositoryHookService.getSettings(request);
            if (projectSettings == null) {
              return null;
            }

            script.install(projectSettings.getSettings(), projectScope, scope);
          }
          return null;
        });
  }
}

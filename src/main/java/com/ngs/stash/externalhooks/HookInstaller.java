package com.ngs.stash.externalhooks;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
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
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;
import com.ngs.stash.externalhooks.dao.ExternalHooksSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.ScopeUtil;
import com.ngs.stash.externalhooks.util.Walker;

public class HookInstaller {
  private RepositoryHookService repositoryHookService;

  private Map<String, ExternalHookScript> scripts = new HashMap<>();
  private Walker walker;
  private SecurityService securityService;

  public HookInstaller(
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

  public Map<String, ExternalHookScript> getScripts() {
    return this.scripts;
  }

  public ExternalHookScript getScript(String idOrKey) {
    if (idOrKey.startsWith(Const.PLUGIN_KEY)) {
      // +1 stands for : after plugin key
      return scripts.get(idOrKey.substring(Const.PLUGIN_KEY.length() + 1));
    }

    return scripts.get(idOrKey);
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

  public boolean enable(Scope scope, String hookKey) {
    ExternalHookScript script = getScript(hookKey);
    if (script == null) {
      return false;
    }

    return enable(scope, script);
  }

  public boolean enable(Scope scope, String hookKey, GlobalHooks globalHooks) {
    ExternalHookScript script = getScript(hookKey);
    if (script == null) {
      return false;
    }

    return enable(scope, script, globalHooks);
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

  public boolean enable(Scope scope, ExternalHookScript script) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      return enable((RepositoryScope) scope, script);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      return enable((ProjectScope) scope, script);
    } else {
      // Do nothing with ScopeType.GLOBAL
      //
      // ScopeType.GLOBAL shall not be here since there are no global hooks
      // actually
      return false;
    }
  }

  public boolean enable(Scope scope, ExternalHookScript script, GlobalHooks globalHooks) {
    if (scope.getType().equals(ScopeType.REPOSITORY)) {
      return enable((RepositoryScope) scope, script, globalHooks);
    } else if (scope.getType().equals(ScopeType.PROJECT)) {
      return enable((ProjectScope) scope, script, globalHooks);
    } else {
      // Do nothing with ScopeType.GLOBAL
      //
      // ScopeType.GLOBAL shall not be here since there are no global hooks
      // actually
      return false;
    }
  }

  public boolean enable(ProjectScope projectScope, ExternalHookScript script) {
    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(projectScope);

    RepositoryHookSettings projectSettings = repositoryHookService.getSettings(
        (new GetRepositoryHookSettingsRequest.Builder(projectScope, script.getHookKey())).build());
    if (projectSettings == null) {
      return false;
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

    return true;
  }

  public boolean enable(
      ProjectScope projectScope, ExternalHookScript script, GlobalHooks globalHooks) {
    // cover legacy hook scripts created only on project level
    script.uninstallLegacy(projectScope);

    walker.walk(projectScope.getProject(), (repository) -> {
      RepositoryScope repositoryScope = new RepositoryScope(repository);
      enable(repositoryScope, script, globalHooks);
    });

    return true;
  }

  public boolean enable(RepositoryScope scope, ExternalHookScript script) {
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

    return true;
  }

  public boolean enable(RepositoryScope scope, ExternalHookScript script, GlobalHooks globalHooks) {
    GlobalHookSettings globalHook = globalHooks.getHook(script.getHookKey());
    FilterPersonalRepositories filter = FilterPersonalRepositories.fromId(globalHook.getFilterPersonalRepositories());
    if (filter == null) {
      filter = FilterPersonalRepositories.DISABLED;
    }

    boolean isPersonal = ((RepositoryScope) scope).getProject().getType() == ProjectType.PERSONAL;

    if (filter == FilterPersonalRepositories.DISABLED
        || (filter == FilterPersonalRepositories.ONLY_PERSONAL && isPersonal) 
        || (filter == FilterPersonalRepositories.EXCLUDE_PERSONAL && !isPersonal)) {
      Settings globalSettings = globalHooks.getSettings(script.getHookKey());
      if (globalSettings == null) {
        throw new RuntimeException("empty settings for " + script.getHookKey());
      }

      script.install(globalSettings, new GlobalScope(), scope);
      return true;
    }

    return false;
  }

  public void disable(ProjectScope _projetScope, ExternalHookScript _script, GlobalScope _hooks) {
    // this is kind of a dirty hack but we don't really need to uninstall hooks
    // like we do it in enable() method since we know that this method is called
    // by Walker which will also call HookInstaller with RepositoryScope (for
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

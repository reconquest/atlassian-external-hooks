package com.ngs.stash.externalhooks.hook.factory;

import java.io.IOException;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.GetRepositoryHookSettingsRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSearchRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSettings;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.project.ProjectService;
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
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.scheduler.SchedulerService;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.ngs.stash.externalhooks.hook.ExternalAsyncPostReceiveHook;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook;
import com.ngs.stash.externalhooks.hook.ExternalPreReceiveHook;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExternalHooksFactory {
  private static Logger log = LoggerFactory.getLogger(ExternalHooksFactory.class.getSimpleName());

  private RepositoryHookService repositoryHookService;

  private ExternalHookScript hookPreReceive;
  private ExternalHookScript hookPostReceive;
  private ExternalHookScript hookMergeCheck;

  public ExternalHooksFactory(
      RepositoryService repositoryService,
      SchedulerService schedulerService,
      HookScriptService hookScriptService,
      RepositoryHookService repositoryHookService,
      ProjectService projectService,
      PluginSettingsFactory pluginSettingsFactory,
      SecurityService securityService,
      AuthenticationContext authenticationContext,
      PermissionService permissionService,
      PluginLicenseManager pluginLicenseManager,
      ClusterService clusterService,
      StorageService storageService)
      throws IOException {
    this.repositoryHookService = repositoryHookService;

    this.hookPreReceive = ExternalPreReceiveHook.getExternalHookScript(
        authenticationContext,
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService);

    this.hookPostReceive = ExternalAsyncPostReceiveHook.getExternalHookScript(
        authenticationContext,
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService);

    this.hookMergeCheck = ExternalMergeCheckHook.getExternalHookScript(
        authenticationContext,
        permissionService,
        pluginLicenseManager,
        clusterService,
        storageService,
        hookScriptService,
        pluginSettingsFactory,
        securityService);
  }

  /**
   * Re-creates Atlassian {@link HookScript} for every {@link RepositoryHook}. Works with both
   * {@link ProjectScope} and {@link RepositoryScope}
   *
   * @param scope
   */
  public void install(Scope scope) {
    log.warn("Creating HookScripts in scope: {}", scope.toString());

    RepositoryHookSearchRequest.Builder searchBuilder =
        new RepositoryHookSearchRequest.Builder(scope);

    Page<RepositoryHook> page = repositoryHookService.search(
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
          this.repositoryHookService.getSettings(getSettingsBuilder.build());

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

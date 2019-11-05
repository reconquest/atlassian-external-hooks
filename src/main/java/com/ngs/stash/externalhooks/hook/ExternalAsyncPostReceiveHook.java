package com.ngs.stash.externalhooks.hook;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookEnabledEvent;
import com.atlassian.bitbucket.hook.repository.GetRepositoryHookSettingsRequest;
import com.atlassian.bitbucket.hook.repository.PostRepositoryHook;
import com.atlassian.bitbucket.hook.repository.PostRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.RepositoryHookRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSettings;
import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.setting.SettingsValidator;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.event.api.EventListener;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;

public class ExternalAsyncPostReceiveHook
    implements PostRepositoryHook<RepositoryHookRequest>, SettingsValidator {

  private ExternalHookScript externalHookScript;
  private RepositoryHookService repositoryHookService;

  public ExternalAsyncPostReceiveHook(
      AuthenticationContext authenticationContext,
      PermissionService permissions,
      PluginLicenseManager pluginLicenseManager,
      ClusterService clusterService,
      StorageService storageProperties,
      HookScriptService hookScriptService,
      PluginSettingsFactory pluginSettingsFactory,
      RepositoryHookService repositoryHookService,
      SecurityService securityService)
      throws IOException {
    this.repositoryHookService = repositoryHookService;

    List<RepositoryHookTrigger> triggers = new ArrayList<RepositoryHookTrigger>();
    triggers.add(StandardRepositoryHookTrigger.REPO_PUSH);
    triggers.add(StandardRepositoryHookTrigger.FILE_EDIT);
    triggers.add(StandardRepositoryHookTrigger.TAG_DELETE);
    triggers.add(StandardRepositoryHookTrigger.TAG_CREATE);
    triggers.add(StandardRepositoryHookTrigger.BRANCH_DELETE);
    triggers.add(StandardRepositoryHookTrigger.BRANCH_CREATE);
    triggers.add(StandardRepositoryHookTrigger.PULL_REQUEST_MERGE);

    this.externalHookScript = new ExternalHookScript(
        authenticationContext,
        permissions,
        pluginLicenseManager,
        clusterService,
        storageProperties,
        hookScriptService,
        pluginSettingsFactory,
        securityService,
        "external-post-receive-hook",
        HookScriptType.POST,
        triggers);
  }

  @Override
  public void validate(
      @Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
    this.externalHookScript.validate(settings, errors, scope);
  }

  @EventListener
  public void onRepositoryHookDisabledEvent(RepositoryHookDisabledEvent event) {
    if (!this.externalHookScript.hookId.equals(event.getRepositoryHookKey())) {
      return;
    }

    this.externalHookScript.deleteHookScriptByKey(event.getRepositoryHookKey(), event.getScope());
  }

  @EventListener
  public void onRepositoryHookEnabledEvent(RepositoryHookEnabledEvent event) {
    if (!this.externalHookScript.hookId.equals(event.getRepositoryHookKey())) {
      return;
    }

    GetRepositoryHookSettingsRequest request = (new GetRepositoryHookSettingsRequest.Builder(
            event.getScope(), event.getRepositoryHookKey()))
        .build();

    RepositoryHookSettings hookSettings = this.repositoryHookService.getSettings(request);

    this.externalHookScript.install(hookSettings.getSettings(), event.getScope());
  }

  @EventListener
  public void onRepositoryHookSettingsDeletedEvent(RepositoryHookDeletedEvent event) {
    if (!this.externalHookScript.hookId.equals(event.getRepositoryHookKey())) {
      return;
    }

    this.externalHookScript.deleteHookScriptByKey(event.getRepositoryHookKey(), event.getScope());
  }

  @Override
  public void postUpdate(
      @Nonnull PostRepositoryHookContext postRepositoryHookContext,
      @Nonnull RepositoryHookRequest repositoryHookRequest) {}
}

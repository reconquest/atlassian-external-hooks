package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.hook.repository.PreRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.PullRequestMergeHookRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookResult;
import com.atlassian.bitbucket.hook.repository.RepositoryMergeCheck;
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

import javax.annotation.Nonnull;
import javax.inject.Named;

@Named("ExternalMergeCheckHook")
public class ExternalMergeCheckHook
    implements RepositoryMergeCheck, SettingsValidator
{
    private ExternalHookScript externalHookScript;

    public ExternalMergeCheckHook(
            AuthenticationContext authenticationContext,
            PermissionService permissions,
            PluginLicenseManager pluginLicenseManager,
            ClusterService clusterService,
            StorageService storageProperties,
            HookScriptService hookScriptService,
            PluginSettingsFactory pluginSettingsFactory,
            SecurityService securityService
    ) {
        externalHookScript = new ExternalHookScript(authenticationContext, permissions, pluginLicenseManager, clusterService, storageProperties,
                hookScriptService, pluginSettingsFactory,  securityService,"external-merge-check-hook", HookScriptType.PRE, StandardRepositoryHookTrigger.PULL_REQUEST_MERGE);
    }

    @Override
    public RepositoryHookResult preUpdate(PreRepositoryHookContext context, PullRequestMergeHookRequest request) {
        return RepositoryHookResult.accepted();
    }

    @Override
    public void validate(@Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
        this.externalHookScript.validate(settings, errors, scope);
    }

    @EventListener
    public void onRepositoryHookSettingsChangedEvent(RepositoryHookDisabledEvent event) {
        this.externalHookScript.deleteHookScript(event.getRepositoryHookKey());
    }

    @EventListener
    public void onRepositoryHookSettingsChangedEvent(RepositoryHookDeletedEvent event) {
        this.externalHookScript.deleteHookScript(event.getRepositoryHookKey());
    }
}

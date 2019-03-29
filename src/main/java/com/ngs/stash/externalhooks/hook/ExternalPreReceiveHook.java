package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.hook.repository.PreRepositoryHook;
import com.atlassian.bitbucket.hook.repository.PreRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.RepositoryHookRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookResult;
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

public class ExternalPreReceiveHook
        implements PreRepositoryHook<RepositoryHookRequest>, SettingsValidator {

    private ExternalHookScript externalHookScript;

    public ExternalPreReceiveHook(
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
                hookScriptService, pluginSettingsFactory,  securityService,"external-pre-receive-hook", HookScriptType.PRE, StandardRepositoryHookTrigger.REPO_PUSH);
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

    @Nonnull
    @Override
    public RepositoryHookResult preUpdate(@Nonnull PreRepositoryHookContext preRepositoryHookContext, @Nonnull RepositoryHookRequest repositoryHookRequest) {
        return RepositoryHookResult.accepted();
    }
}

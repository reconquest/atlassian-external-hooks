package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.PostRepositoryHook;
import com.atlassian.bitbucket.hook.repository.PostRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.RepositoryHookRequest;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.repository.RepositoryService;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.server.ApplicationPropertiesService;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.setting.SettingsValidator;
import com.atlassian.cache.CacheFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;

import javax.annotation.Nonnull;
import java.util.logging.Logger;

public class ExternalAsyncPostReceiveHook
        implements PostRepositoryHook<RepositoryHookRequest>, SettingsValidator {
    private final PluginLicenseManager pluginLicenseManager;

    private static Logger log = Logger.getLogger(
            ExternalAsyncPostReceiveHook.class.getSimpleName()
    );

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ClusterService clusterService;
    private ApplicationPropertiesService properties;
    private StorageService storageProperties;
    private CacheFactory cacheFactory;


    public ExternalAsyncPostReceiveHook(
            AuthenticationContext authenticationContext,
            PermissionService permissions,
            RepositoryService repoService,
            ApplicationPropertiesService properties,
            PluginLicenseManager pluginLicenseManager,
            ClusterService clusterService,
            StorageService storageProperties,
            CacheFactory cacheFactory

    ) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
        this.repoService = repoService;
        this.properties = properties;
        this.clusterService = clusterService;
        this.pluginLicenseManager = pluginLicenseManager;
        this.storageProperties = storageProperties;
        this.cacheFactory = cacheFactory;
    }

    @Override
    public void postUpdate(PostRepositoryHookContext context, RepositoryHookRequest request) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(
                this.authCtx, this.permissions, this.repoService, this.properties,
                this.pluginLicenseManager, this.clusterService, this.storageProperties, this.cacheFactory
        );

        impl.preUpdateImpl(context, request);
    }

    @Override
    public void validate(@Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx,
                this.permissions, this.repoService, this.properties,
                this.pluginLicenseManager, this.clusterService, this.storageProperties, this.cacheFactory
        );

        impl.validate(settings, errors, scope);
    }
}

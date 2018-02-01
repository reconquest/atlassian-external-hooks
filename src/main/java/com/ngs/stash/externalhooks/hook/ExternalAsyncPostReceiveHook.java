package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.hook.repository.*;
import com.atlassian.bitbucket.repository.*;
import com.atlassian.bitbucket.setting.*;
import com.atlassian.bitbucket.auth.*;
import com.atlassian.bitbucket.permission.*;
import com.atlassian.bitbucket.server.*;

import static java.util.logging.Level.SEVERE;
import static java.util.logging.Level.INFO;
import java.util.logging.Logger;


public class ExternalAsyncPostReceiveHook
    implements PostRepositoryHook<RepositoryHookRequest>, RepositorySettingsValidator
{
    private static Logger log = Logger.getLogger(
        ExternalAsyncPostReceiveHook.class.getSimpleName()
    );

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private RepositoryService repoService;
    private ApplicationPropertiesService properties;

    public ExternalAsyncPostReceiveHook(
        AuthenticationContext authenticationContext,
        PermissionService permissions,
        RepositoryService repoService,
        ApplicationPropertiesService properties
    ) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
        this.repoService = repoService;
        this.properties = properties;
    }

	@Override
	public void postUpdate(PostRepositoryHookContext context, RepositoryHookRequest request) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(
                this.authCtx, this.permissions, this.repoService, this.properties);

        impl.preUpdateImpl(context, request);
	}

    @Override
    public void validate(
        Settings settings,
        SettingsValidationErrors errors,
        Repository repository
    ) {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx,
            this.permissions, this.repoService, this.properties);

        impl.validate(settings, errors, repository);
    }
}

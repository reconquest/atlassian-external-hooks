package com.ngs.stash.externalhooks.hook;

import com.atlassian.stash.hook.repository.*;
import com.atlassian.stash.repository.*;
import com.atlassian.stash.setting.*;
import com.atlassian.stash.env.SystemProperties;
import com.atlassian.stash.user.*;
import com.ngs.stash.externalhooks.hook.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import java.util.Collection;
import java.io.*;
import java.util.Map;
import java.util.List;
import java.util.LinkedList;


public class ExternalPostReceiveHook implements AsyncPostReceiveRepositoryHook, RepositorySettingsValidator
{
    private static final Logger log = LoggerFactory.getLogger(ExternalPostReceiveHook.class);

    private StashAuthenticationContext authCtx;
    private PermissionService permissions;
    public ExternalPostReceiveHook(StashAuthenticationContext authenticationContext, PermissionService permissions) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
    }

    /**
     * Call external executable as git hook.
     */
    @Override
    public void postReceive(RepositoryHookContext context, Collection<RefChange> refChanges)
    {
        ExternalPreReceiveHook impl = new ExternalPreReceiveHook(this.authCtx, this.permissions);
        impl.onReceive(context, refChanges, null);
    }

    @Override
    public void validate(Settings settings, SettingsValidationErrors errors, Repository repository)
    {
        if (!permissions.hasGlobalPermission(authCtx.getCurrentUser(), Permission.SYS_ADMIN)) {
            errors.addFieldError("exe", "You should be Stash Administrator to edit this field.");
            return;
        }

        if (settings.getString("exe", "").isEmpty()) {
            errors.addFieldError("exe", "Executable is blank, please specify something");
        }
    }
}

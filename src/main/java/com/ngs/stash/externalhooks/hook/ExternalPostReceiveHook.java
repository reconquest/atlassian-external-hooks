package com.ngs.stash.externalhooks.hook;

import com.atlassian.stash.hook.repository.*;
import com.atlassian.stash.repository.*;
import com.atlassian.stash.setting.*;
import com.atlassian.stash.env.SystemProperties;
import com.atlassian.stash.user.*;
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
        Repository repo = context.getRepository();
        String repo_path = System.getProperty(SystemProperties.HOME_DIR_SYSTEM_PROPERTY) +
            "/data/repositories/" + repo.getId();

        List<String> exe = new LinkedList<String>();
        exe.add(context.getSettings().getString("exe"));
        if (context.getSettings().getString("params") != null) {
            for (String arg : context.getSettings().getString("params").split("\r\n")) {
                exe.add(arg);
            }
        }

        StashUser currentUser = authCtx.getCurrentUser();
        boolean isAdmin = permissions.hasRepositoryPermission(currentUser, repo, Permission.REPO_ADMIN);
        ProcessBuilder pb = new ProcessBuilder(exe);
        Map<String, String> env = pb.environment();
        env.put("STASH_USER_NAME", currentUser.getName());
        env.put("STASH_USER_EMAIL", currentUser.getEmailAddress());
        env.put("STASH_REPO_NAME", repo.getName());
        env.put("STASH_IS_ADMIN", String.valueOf(isAdmin));
        pb.directory(new File(repo_path));
        pb.redirectErrorStream(true);
        try {
            Process process = pb.start();
            InputStream input = process.getInputStream();
            OutputStream output = process.getOutputStream();

            for (RefChange refChange : refChanges) {
                output.write((refChange.getFromHash() + " " +
                    refChange.getToHash() + " " +
                    refChange.getRefId() + "\n").getBytes("UTF-8"));
            }
            output.close();

            process.waitFor();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        } catch (IOException e) {
            log.error("Error running {} in {}", exe, repo_path, e);
        }
    }

    @Override
    public void validate(Settings settings, SettingsValidationErrors errors, Repository repository)
    {
        if (settings.getString("exe", "").isEmpty())
        {
            errors.addFieldError("exe", "Executable is blank, please specify something");
        }
    }
}

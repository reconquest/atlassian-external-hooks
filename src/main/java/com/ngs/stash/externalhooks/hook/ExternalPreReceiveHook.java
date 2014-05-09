package com.ngs.stash.externalhooks.hook;

import com.atlassian.stash.hook.*;
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
import com.atlassian.stash.user.Permission;
import com.atlassian.stash.user.PermissionService;

public class ExternalPreReceiveHook implements PreReceiveRepositoryHook, RepositorySettingsValidator
{
    private static final Logger log = LoggerFactory.getLogger(ExternalPreReceiveHook.class);
    private StashAuthenticationContext authCtx;
    private PermissionService permissions;
    public ExternalPreReceiveHook(StashAuthenticationContext authenticationContext, PermissionService permissions) {
        this.authCtx = authenticationContext;
        this.permissions = permissions;
    }

    /**
     * Call external executable as git hook.
     */
    @Override
    public boolean onReceive(RepositoryHookContext context, Collection<RefChange> refChanges, HookResponse hookResponse)
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
            InputStreamReader input = new InputStreamReader(process.getInputStream(), "UTF-8");
            OutputStream output = process.getOutputStream();

            for (RefChange refChange : refChanges) {
                output.write((refChange.getFromHash() + " " +
                    refChange.getToHash() + " " +
                    refChange.getRefId() + "\n").getBytes("UTF-8"));
            }
            output.close();

            if (hookResponse != null) {
                int data;
                int count = 0;
                boolean clipOutput = context.getSettings().getBoolean("clipOutput", true);
                while ((data = input.read()) >= 0) {
                    if (clipOutput && count >= 65000) {
                        hookResponse.err().print("\n");
                        hookResponse.err().print("Hook response exceeds 65K length limit.\n");
                        hookResponse.err().print("Further output will be trimmed.\n");

                        process.destroy();
                        //if no setting, default is to reject
                        boolean rejectWhenClipped = context.getSettings().getBoolean("rejectWhenClipped", true);
                        return rejectWhenClipped;
                    }

                    String char_to_write = Character.toString((char)data);

                    count += char_to_write.getBytes("utf-8").length;

                    hookResponse.err().print(char_to_write);
                }

            }

            return process.waitFor() == 0;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return false;
        } catch (IOException e) {
            log.error("Error running {} in {}", exe, repo_path, e);
            return false;
        }
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

package com.ngs.stash.externalhooks.hook;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptConfig;
import com.atlassian.bitbucket.hook.script.HookScriptCreateRequest;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptSetConfigurationRequest;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.hook.script.HookScriptUpdateRequest;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.atlassian.upm.api.license.entity.PluginLicense;
import com.atlassian.upm.api.util.Option;
import org.apache.commons.io.FilenameUtils;

import javax.annotation.Nonnull;
import java.io.File;
import java.io.FileInputStream;
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExternalHookScript {

    public static final String PLUGIN_ID = "com.ngs.stash.externalhooks.external-hooks";

    private final PluginLicenseManager pluginLicenseManager;

    private static Logger log = LoggerFactory.getLogger(
            ExternalHookScript.class.getSimpleName()
    );

    private AuthenticationContext authCtx;
    private PermissionService permissions;
    private ClusterService clusterService;
    private StorageService storageProperties;
    private HookScriptService hookScriptService;
    private PluginSettings pluginSettings;
    private String hookComponentId;
    private String hookId;
    private HookScriptType hookScriptType;
    private RepositoryHookTrigger repositoryHookTrigger;
    private SecurityService securityService;

    public ExternalHookScript(
            AuthenticationContext authenticationContext,
            PermissionService permissions,
            PluginLicenseManager pluginLicenseManager,
            ClusterService clusterService,
            StorageService storageProperties,
            HookScriptService hookScriptService,
            PluginSettingsFactory pluginSettingsFactory,
            SecurityService securityService,
            String hookComponentId,
            HookScriptType hookScriptType,
            RepositoryHookTrigger repositoryHookTrigger) {

        this.authCtx = authenticationContext;
        this.permissions = permissions;
        this.storageProperties = storageProperties;
        this.pluginLicenseManager = pluginLicenseManager;
        this.clusterService = clusterService;
        this.hookScriptService = hookScriptService;
        this.pluginSettings = pluginSettingsFactory.createGlobalSettings();
        this.hookComponentId = hookComponentId;
        this.hookId = PLUGIN_ID + ":" + hookComponentId;
        this.hookScriptType = hookScriptType;
        this.repositoryHookTrigger = repositoryHookTrigger;
        this.securityService = securityService;
    }

    public void validate(@Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
        if (!this.isLicenseValid()) {
            errors.addFieldError("exe",
                    "License for External Hooks is expired.");
            return;
        }

        if (this.clusterService.isAvailable() && !settings.getBoolean("safe_path", false)) {
            errors.addFieldError("exe",
                    "Bitbucket is running in DataCenter mode. You must use \"safe mode\" option.");
            return;
        }

        if (!settings.getBoolean("safe_path", false)) {
            if (!permissions.hasGlobalPermission(
                    authCtx.getCurrentUser(), Permission.SYS_ADMIN)) {
                errors.addFieldError("exe",
                        "You should be a Bitbucket System Administrator to edit this field " +
                                "without \"safe mode\" option.");
                return;
            }
        }

        if (settings.getString("exe", "").isEmpty()) {
            errors.addFieldError("exe",
                    "Executable is blank, please specify something");
            return;
        }

        File executable = this.getExecutable(
                settings.getString("exe", ""),
                settings.getBoolean("safe_path", false));

        if ((executable == null) || (!executable.exists())) {
            errors.addFieldError("exe",
                    "Executable does not exist");
            return;
        }

        boolean isExecutable;
        try {
            isExecutable = executable.canExecute() && executable.isFile();
        } catch (SecurityException e) {
            log.error("Security exception on " + executable.getPath(), e);
            isExecutable = false;
        }

        if (!isExecutable) {
            errors.addFieldError("exe",
                    "Specified path is not executable file. Check executable flag.");
            return;
        }

        StringBuilder scriptBuilder = new StringBuilder();
        scriptBuilder.append("#!/bin/bash").append("\n\n");

        scriptBuilder.append(executable);

        String params = settings.getString("params");
        if (params != null) {
            params = params.trim();
            if (params.length() != 0) {
                for (String arg : settings.getString("params").split("\r\n")) {
                    if (arg.length() != 0) {
                        scriptBuilder.append(" ").append(arg);
                    }
                }
            }
        }

        scriptBuilder.append("\n\n");
        String script = scriptBuilder.toString();

        HookScript hookScript = null;

        Object id = pluginSettings.get(this.hookId);
        if (id != null) {
            Optional<HookScript> maybeHookScript = hookScriptService.findById(Long.valueOf(id.toString()));
            if (maybeHookScript.isPresent()) {
                hookScript = maybeHookScript.get();
            } else {
                log.warn("Settings had id {} stored, but hook was already gone", id);
                pluginSettings.remove(this.hookId);
            }
        }

        if (hookScript == null) {
            HookScriptCreateRequest.Builder test = new HookScriptCreateRequest.Builder(this.hookComponentId, PLUGIN_ID, this.hookScriptType)
                    .content(script);
            HookScriptCreateRequest hookScriptCreateRequest = test.build();

            hookScript = securityService.withPermission(Permission.SYS_ADMIN, "External Hook Plugin: Allow repo admins to set hooks").call(
                    () -> hookScriptService.create(hookScriptCreateRequest));
            pluginSettings.put(this.hookId, String.valueOf(hookScript.getId()));

            HookScriptSetConfigurationRequest.Builder configBuilder = new HookScriptSetConfigurationRequest.Builder(hookScript, scope);
            configBuilder.trigger(this.repositoryHookTrigger);
            HookScriptSetConfigurationRequest hookScriptSetConfigurationRequest = configBuilder.build();
            hookScriptService.setConfiguration(hookScriptSetConfigurationRequest);

            log.info("Successfully created HookScript with id: {}", hookScript.getId());
        } else {
            HookScriptUpdateRequest hookScriptUpdateRequest = new HookScriptUpdateRequest.Builder(hookScript).content(script).build();
            HookScript updatedHookScript = securityService.withPermission(Permission.SYS_ADMIN, "External Hook Plugin: Allow repo admins to update hooks").call(
                    () -> hookScriptService.update(hookScriptUpdateRequest));

            log.info("Successfully updated HookScript. id: {}, version: {}", updatedHookScript.getId(), updatedHookScript.getVersion());
        }
    }

    public File getExecutable(String path, boolean safeDir) {
        File executable = new File(path);
        if (safeDir) {
            path = FilenameUtils.normalize(path);
            if (path == null) {
                executable = null;
            } else {
                String safeBaseDir =
                        getHomeDir().getAbsolutePath() +
                                "/external-hooks/";
                executable = new File(safeBaseDir, path);
            }
        }

        return executable;
    }

    private File getHomeDir() {
        if (this.clusterService.isAvailable()) {
            return this.storageProperties.getSharedHomeDir().toFile();
        } else {
            return this.storageProperties.getHomeDir().toFile();
        }
    }

    public boolean isLicenseValid() {
        Option<PluginLicense> licenseOption = pluginLicenseManager.getLicense();
        if (!licenseOption.isDefined()) {
            return true;
        }

        PluginLicense pluginLicense = licenseOption.get();
        return pluginLicense.isValid();
    }

    public void deleteHookScript(String hookKey) {
        if (!this.hookId.equals(hookKey)) {
            return;
        }

        Object id = pluginSettings.get(this.hookId);
        if (id != null) {
            Optional<HookScript> maybeHookScript = hookScriptService.findById(Long.valueOf(id.toString()));
            if (maybeHookScript.isPresent()) {
                securityService.withPermission(Permission.SYS_ADMIN, "External Hook Plugin: Allow repo admins to update hooks").call(
                        () -> {
                            hookScriptService.delete(maybeHookScript.get());
                            return null;
                        });
                log.info("Successfully deleted HookScript with id: {}", id);
            } else {
                log.warn("Attempting to deleted HookScript with id: {}, but it is already gone", id);
            }
            pluginSettings.remove(this.hookId);
        }
    }
}

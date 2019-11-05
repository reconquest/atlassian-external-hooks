package com.ngs.stash.externalhooks.hook;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.List;
import java.util.Optional;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.auth.AuthenticationContext;
import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptCreateRequest;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptSetConfigurationRequest;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.plugin.util.ClassLoaderUtils;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.atlassian.upm.api.license.entity.PluginLicense;
import com.atlassian.upm.api.util.Option;
import com.google.common.base.Charsets;
import com.google.common.escape.Escaper;
import com.google.common.escape.Escapers;

import org.apache.commons.io.FilenameUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExternalHookScript {
  public static final String PLUGIN_KEY = "com.ngs.stash.externalhooks.external-hooks";
  private final PluginLicenseManager pluginLicenseManager;
  private static Logger log = LoggerFactory.getLogger(ExternalHookScript.class.getSimpleName());
  public final Escaper SHELL_ESCAPE;
  private AuthenticationContext authCtx;
  private PermissionService permissions;
  private ClusterService clusterService;
  private StorageService storageProperties;
  private HookScriptService hookScriptService;
  private PluginSettings pluginSettings;
  private String hookComponentId;
  public String hookId;
  private HookScriptType hookScriptType;
  private List<RepositoryHookTrigger> repositoryHookTriggers;
  private SecurityService securityService;
  private String hookScriptTemplate;

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
      List<RepositoryHookTrigger> repositoryHookTriggers)
      throws IOException {
    this.authCtx = authenticationContext;
    this.permissions = permissions;
    this.storageProperties = storageProperties;
    this.pluginLicenseManager = pluginLicenseManager;
    this.clusterService = clusterService;
    this.hookScriptService = hookScriptService;
    this.pluginSettings = pluginSettingsFactory.createGlobalSettings();
    this.hookComponentId = hookComponentId;
    this.hookId = PLUGIN_KEY + ":" + hookComponentId;
    this.hookScriptType = hookScriptType;
    this.repositoryHookTriggers = repositoryHookTriggers;
    this.securityService = securityService;

    final Escapers.Builder builder = Escapers.builder();
    builder.addEscape('\'', "'\"'\"'");
    SHELL_ESCAPE = builder.build();

    this.hookScriptTemplate = this.getResource("hook-script.template.bash");
  }

  private String getResource(String name) throws IOException {
    InputStream resource = ClassLoaderUtils.getResourceAsStream(name, this.getClass());
    if (resource == null) {
      throw new IllegalArgumentException("file is not found");
    }

    StringBuilder stringBuilder = new StringBuilder();
    String line = null;

    try (BufferedReader bufferedReader =
        new BufferedReader(new InputStreamReader(resource, Charsets.UTF_8))) {
      while ((line = bufferedReader.readLine()) != null) {
        stringBuilder.append(line).append("\n");
      }
    }

    return stringBuilder.toString();
  }

  public void validate(
      @Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
    if (!this.isLicenseDefined()) {
      errors.addFieldError("exe", "External Hooks Add-on is Unlicensed.");
      return;
    }

    if (!this.isLicenseValid()) {
      errors.addFieldError("exe", "License for External Hooks is expired.");
      return;
    }

    if (this.clusterService.isAvailable() && !settings.getBoolean("safe_path", false)) {
      errors.addFieldError(
          "exe", "Bitbucket is running in DataCenter mode. You must use \"safe mode\" option.");
      return;
    }

    if (!settings.getBoolean("safe_path", false)) {
      if (!permissions.hasGlobalPermission(Permission.SYS_ADMIN)) {
        errors.addFieldError(
            "exe", "You should be a Bitbucket System Administrator to edit this field "
                + "without \"safe mode\" option.");
        return;
      }
    }

    if (settings.getString("exe", "").isEmpty()) {
      errors.addFieldError("exe", "Executable is blank, please specify something");
      return;
    }

    File executable =
        this.getExecutable(settings.getString("exe", ""), settings.getBoolean("safe_path", false));

    if ((executable == null) || (!executable.isFile())) {
      errors.addFieldError("exe", "Executable does not exist");
      return;
    }

    boolean isExecutable;
    try {
      isExecutable = executable.canExecute();
    } catch (SecurityException e) {
      log.error("Security exception on " + executable.getPath(), e);
      isExecutable = false;
    }

    if (!isExecutable) {
      errors.addFieldError("exe", "Specified path is not executable file. Check executable flag.");
      return;
    }
  }

  public void install(@Nonnull Settings settings, @Nonnull Scope scope) {
    File executable =
        this.getExecutable(settings.getString("exe", ""), settings.getBoolean("safe_path", false));

    Boolean async = settings.getBoolean("async", false);

    StringBuilder scriptBuilder = new StringBuilder();
    scriptBuilder.append(this.hookScriptTemplate).append("\n\n");

    if (async) {
      // dumping stdin to a temporary file
      scriptBuilder.append("stdin=\"$(mktemp)\"\n");
      scriptBuilder.append("cat >\"$stdin\"\n");
      // subshell start
      scriptBuilder.append("(\n");
      // deleting stdin after finishing the job
      scriptBuilder.append("    trap \"rm \\\"$stdin\\\"\" EXIT\n");
      // just an indentation for script and params
      scriptBuilder.append("    ");
    }
    scriptBuilder.append("'").append(SHELL_ESCAPE.escape(executable.toString())).append("'");

    String params = settings.getString("params");
    if (params != null) {
      params = params.trim();
      if (params.length() != 0) {
        for (String arg : settings.getString("params").split("\r\n")) {
          if (arg.length() != 0) {
            scriptBuilder.append(" '").append(SHELL_ESCAPE.escape(arg)).append('\'');
          }
        }
      }
    }

    if (async) {
      scriptBuilder.append(" <\"$stdin\"\n");

      // subshell end: closing all fds and starting subshell in background
      scriptBuilder.append(") >/dev/null 2>&1 <&- &\n");
    }

    scriptBuilder.append("\n");

    String script = scriptBuilder.toString();

    HookScript hookScript = null;

    String hookId = getHookId(scope);
    Object id = pluginSettings.get(hookId);
    if (id != null) {
      Optional<HookScript> maybeHookScript =
          hookScriptService.findById(Long.valueOf(id.toString()));
      if (maybeHookScript.isPresent()) {
        hookScript = maybeHookScript.get();
      } else {
        log.warn("Settings had id {} stored, but hook was already gone", id);
        pluginSettings.remove(hookId);
      }
    }

    if (hookScript != null) {
      this.deleteHookScript(hookScript);
    }

    HookScriptCreateRequest.Builder test = new HookScriptCreateRequest.Builder(
            this.hookComponentId, PLUGIN_KEY, this.hookScriptType)
        .content(script);
    HookScriptCreateRequest hookScriptCreateRequest = test.build();

    hookScript = securityService
        .withPermission(
            Permission.SYS_ADMIN, "External Hook Plugin: Allow repo admins to set hooks")
        .call(() -> hookScriptService.create(hookScriptCreateRequest));
    pluginSettings.put(hookId, String.valueOf(hookScript.getId()));

    HookScriptSetConfigurationRequest.Builder configBuilder =
        new HookScriptSetConfigurationRequest.Builder(hookScript, scope);
    configBuilder.triggers(this.repositoryHookTriggers);
    HookScriptSetConfigurationRequest hookScriptSetConfigurationRequest = configBuilder.build();
    hookScriptService.setConfiguration(hookScriptSetConfigurationRequest);

    log.info("Successfully created HookScript with id: {}", hookScript.getId());
  }

  public File getExecutable(String path, boolean safeDir) {
    File executable = new File(path);
    if (safeDir) {
      path = FilenameUtils.normalize(path);
      if (path == null) {
        executable = null;
      } else {
        String safeBaseDir = getHomeDir().getAbsolutePath() + "/external-hooks/";
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
      return false;
    }

    PluginLicense pluginLicense = licenseOption.get();
    return pluginLicense.isValid();
  }

  public boolean isLicenseDefined() {
    Option<PluginLicense> licenseOption = pluginLicenseManager.getLicense();
    return licenseOption.isDefined();
  }

  public void deleteHookScriptByKey(String hookKey, Scope scope) {
    if (!this.hookId.equals(hookKey)) {
      return;
    }

    String hookId = this.getHookId(scope);
    Object id = pluginSettings.get(hookId);
    if (id != null) {
      Optional<HookScript> maybeHookScript =
          hookScriptService.findById(Long.valueOf(id.toString()));
      if (maybeHookScript.isPresent()) {
        HookScript hookScript = maybeHookScript.get();
        deleteHookScript(hookScript);
        log.info("Successfully deleted HookScript with id: {}", id);
      } else {
        log.warn("Attempting to delete HookScript with id: {}, but it is already gone", id);
      }
      pluginSettings.remove(hookId);
    }
  }

  private String getHookId(Scope scope) {
    StringBuilder builder = new StringBuilder(this.hookId);
    builder.append(":").append(scope.getType().getId());
    if (scope.getResourceId().isPresent()) {
      builder.append(":").append(scope.getResourceId().get());
    }
    return builder.toString();
  }

  private void deleteHookScript(HookScript hookScript) {
    securityService
        .withPermission(
            Permission.SYS_ADMIN, "External Hooks Plugin: Allow repo admins to update hooks")
        .call(() -> {
          hookScriptService.delete(hookScript);
          return null;
        });
  }
}

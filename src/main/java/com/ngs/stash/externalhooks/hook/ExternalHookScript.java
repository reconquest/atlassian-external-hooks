package com.ngs.stash.externalhooks.hook;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptCreateRequest;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.hook.script.HookScriptSetConfigurationRequest;
import com.atlassian.bitbucket.hook.script.HookScriptType;
import com.atlassian.bitbucket.permission.Permission;
import com.atlassian.bitbucket.permission.PermissionService;
import com.atlassian.bitbucket.scope.GlobalScope;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.scope.ScopeType;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.user.SecurityService;
import com.atlassian.plugin.util.ClassLoaderUtils;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.google.common.base.Charsets;
import com.google.common.escape.Escaper;
import com.google.common.escape.Escapers;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.LicenseValidator;
import com.ngs.stash.externalhooks.util.ScopeUtil;

import org.apache.commons.io.FilenameUtils;
import org.apache.commons.lang3.tuple.Pair;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExternalHookScript {
  private static Logger log = LoggerFactory.getLogger(ExternalHookScript.class);

  private final Escaper SHELL_ESCAPE;
  private PermissionService permissionService;
  private ClusterService clusterService;
  private StorageService storageService;
  private HookScriptService hookScriptService;
  private PluginSettings pluginSettings;
  private String hookId;
  private HookScriptType hookScriptType;
  private HookTriggersGetter getRepositoryHookTriggers;
  private SecurityService securityService;
  private String hookScriptTemplate;
  private LicenseValidator license;
  private String hookKey;

  public ExternalHookScript(
      PermissionService permissionService,
      PluginLicenseManager pluginLicenseManager,
      ClusterService clusterService,
      StorageService storageService,
      HookScriptService hookScriptService,
      PluginSettingsFactory pluginSettingsFactory,
      SecurityService securityService,
      String hookId,
      HookScriptType hookScriptType,
      HookTriggersGetter getRepositoryHookTriggers)
      throws IOException {
    this.permissionService = permissionService;
    this.storageService = storageService;
    this.clusterService = clusterService;
    this.hookScriptService = hookScriptService;
    this.pluginSettings = pluginSettingsFactory.createGlobalSettings();
    this.hookId = hookId;
    this.hookKey = Const.PLUGIN_KEY + ":" + hookId;
    this.hookScriptType = hookScriptType;
    this.securityService = securityService;
    this.getRepositoryHookTriggers = getRepositoryHookTriggers;

    final Escapers.Builder builder = Escapers.builder();
    builder.addEscape('\'', "'\"'\"'");
    SHELL_ESCAPE = builder.build();

    this.hookScriptTemplate = this.getResource("hook-script.template.bash");

    this.license = new LicenseValidator(pluginLicenseManager, storageService, clusterService);
  }

  public String getHookKey() {
    return hookKey;
  }

  public String getHookId() {
    return hookId;
  }

  private String getResource(String name) throws IOException {
    InputStream resource = ClassLoaderUtils.getResourceAsStream(name, this.getClass());
    if (resource == null) {
      throw new IllegalArgumentException("resource file not found: " + name);
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
    if (!this.license.isDefined()) {
      errors.addFieldError("exe", "External Hooks Add-on is Unlicensed.");
      return;
    }

    if (!this.license.isValid()) {
      errors.addFieldError("exe", "License for External Hooks is expired.");
      return;
    }

    if (this.clusterService.isAvailable() && !settings.getBoolean("safe_path", false)) {
      errors.addFieldError(
          "exe", "Bitbucket is running in DataCenter mode. You must use \"safe mode\" option.");
      return;
    }

    if (!settings.getBoolean("safe_path", false)) {
      if (!permissionService.hasGlobalPermission(Permission.SYS_ADMIN)) {
        errors.addFieldError(
            "exe",
            "You should be a Bitbucket System Administrator to edit this field "
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
      log.error("security exception on " + executable.getPath(), e);
      isExecutable = false;
    }

    if (!isExecutable) {
      errors.addFieldError("exe", "Specified path is not executable file. Check executable flag.");
      return;
    }
  }

  // Should be used only for uninstalling legacy ProjectScope scripts.
  public void uninstallLegacy(ProjectScope scope) {
    String pluginSettingsPath = getLegacyPluginSettingsPath(scope);

    DeletionResult result = deleteHookScript(pluginSettingsPath);
    if (result != DeletionResult.MISSING_ID) {
      // Unlike other methods we don't want to spam this message because this
      // function will be called on every next plugin version, unfortunately.
      log.debug(
          "deleting legacy project hook script {} on {}: {}",
          hookId,
          ScopeUtil.toString(scope),
          result.getMessage());

      pluginSettings.remove(pluginSettingsPath);
    }
  }

  public void uninstall(ProjectScope parent, RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(parent, scope);

    DeletionResult result = deleteHookScript(pluginSettingsPath);

    log.debug(
        "deleting project hook script {} of {} on {}: {}",
        hookId,
        ScopeUtil.toString(parent),
        ScopeUtil.toString(scope),
        result.getMessage());

    if (result != DeletionResult.MISSING_ID) {
      pluginSettings.remove(pluginSettingsPath);
    }
  }

  public void uninstall(GlobalScope parentScope, RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(parentScope,scope);

    DeletionResult result = deleteHookScript(pluginSettingsPath);

    log.debug(
        "deleting global hook script {} on repository {}: {}",
        hookId,
        ScopeUtil.toString(scope),
        result.getMessage());

    if (result != DeletionResult.MISSING_ID) {
      pluginSettings.remove(pluginSettingsPath);
    }
  }

  public void uninstall(RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(scope);

    DeletionResult result = deleteHookScript(pluginSettingsPath);

    log.debug(
        "deleting repository hook script {} on {}: {}",
        hookId,
        ScopeUtil.toString(scope),
        result.getMessage());

    if (result != DeletionResult.MISSING_ID) {
      pluginSettings.remove(pluginSettingsPath);
    }
  }

  public void install(
      @Nonnull Settings settings, @Nonnull ProjectScope parent, @Nonnull RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(parent, scope);
    Pair<HookScript, List<RepositoryHookTrigger>> result =
        install(pluginSettingsPath, settings, scope);

    log.debug(
        "created project hook script {} of {} with id: {} on {}; triggers: {}",
        hookId,
        ScopeUtil.toString(parent),
        result.getLeft().getId(),
        ScopeUtil.toString(scope),
        listTriggers(result.getRight()));
  }

  public void install(
      @Nonnull Settings settings,
      @Nonnull GlobalScope globalParent,
      @Nonnull RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(globalParent, scope);
    Pair<HookScript, List<RepositoryHookTrigger>> result =
        install(pluginSettingsPath, settings, scope);

    log.debug(
        "created repository hook script of global configuration with id: {} on {}; triggers: {}",
        hookId,
        result.getLeft().getId(),
        listTriggers(result.getRight()));
  }

  public void install(@Nonnull Settings settings, @Nonnull RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(scope);
    Pair<HookScript, List<RepositoryHookTrigger>> result =
        install(pluginSettingsPath, settings, scope);

    log.debug(
        "created repository hook script {} with id: {} on {}; triggers: {}",
        hookId,
        result.getLeft().getId(),
        ScopeUtil.toString(scope),
        listTriggers(result.getRight()));
  }

  private Pair<HookScript, List<RepositoryHookTrigger>> install(
      String pluginSettingsPath, @Nonnull Settings settings, @Nonnull RepositoryScope scope) {
    deleteHookScript(pluginSettingsPath);

    HookScript hookScript = create(settings);

    pluginSettings.put(pluginSettingsPath, String.valueOf(hookScript.getId()));

    List<RepositoryHookTrigger> triggers = getRepositoryHookTriggers.get();

    HookScriptSetConfigurationRequest.Builder configBuilder =
        new HookScriptSetConfigurationRequest.Builder(hookScript, scope);
    configBuilder.triggers(triggers);

    HookScriptSetConfigurationRequest configRequest = configBuilder.build();
    hookScriptService.setConfiguration(configRequest);

    return Pair.of(hookScript, triggers);
  }

  private DeletionResult deleteHookScript(String pluginSettingsPath) {
    Object id = pluginSettings.get(pluginSettingsPath);
    if (id != null) {
      log.debug("delete hook script: {}", id);

      Optional<HookScript> maybeHookScript =
          hookScriptService.findById(Long.valueOf(id.toString()));
      if (maybeHookScript.isPresent()) {
        return securityService
            .withPermission(Permission.SYS_ADMIN, "atlassian-external-hooks: delete hook script")
            .call(() -> {
              hookScriptService.delete(maybeHookScript.get());
              return DeletionResult.OK;
            });
      } else {
        return DeletionResult.MISSING_SCRIPT;
      }
    }

    return DeletionResult.MISSING_ID;
  }

  private HookScript create(Settings settings) {
    String script = getScriptContents(settings);

    HookScriptCreateRequest.Builder builder = new HookScriptCreateRequest.Builder(
            this.hookId, Const.PLUGIN_KEY, this.hookScriptType)
        .content(script);
    HookScriptCreateRequest hookScriptCreateRequest = builder.build();

    return securityService
        .withPermission(
            Permission.SYS_ADMIN, "atlassian-external-hooks: create low-level hook script")
        .call(() -> hookScriptService.create(hookScriptCreateRequest));
  }

  private String getScriptContents(Settings settings) {
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

    return scriptBuilder.toString();
  }

  private String listTriggers(List<RepositoryHookTrigger> list) {
    return "["
        + list.stream().map(trigger -> trigger.getId()).collect(Collectors.joining(", "))
        + "]";
  }

  private File getExecutable(String path, boolean safeDir) {
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
      return this.storageService.getSharedHomeDir().toFile();
    } else {
      return this.storageService.getHomeDir().toFile();
    }
  }

  private String getPluginSettingsPath(ProjectScope parent, RepositoryScope scope) {
    StringBuilder builder = new StringBuilder(this.hookKey);
    builder.append(":").append(ScopeType.PROJECT.getId());
    builder.append(":").append(parent.getResourceId().orElse(-1));

    builder.append(":").append(ScopeType.REPOSITORY.getId());
    builder.append(":").append(scope.getResourceId().orElse(-1));

    return builder.toString();
  }

  private String getPluginSettingsPath(GlobalScope parent, RepositoryScope scope) {
    StringBuilder builder = new StringBuilder(this.hookKey);
    builder.append(":").append(ScopeType.GLOBAL.getId());
    builder.append(":").append("global");

    builder.append(":").append(ScopeType.REPOSITORY.getId());
    builder.append(":").append(scope.getResourceId().orElse(-1));

    return builder.toString();
  }

  private String getPluginSettingsPath(RepositoryScope scope) {
    StringBuilder builder = new StringBuilder(this.hookKey);
    builder.append(":").append(ScopeType.REPOSITORY.getId());
    builder.append(":").append(scope.getResourceId().orElse(-1));

    return builder.toString();
  }

  private String getLegacyPluginSettingsPath(ProjectScope scope) {
    StringBuilder builder = new StringBuilder(this.hookKey);
    builder.append(":").append(scope.getType().getId());
    if (scope.getResourceId().isPresent()) {
      builder.append(":").append(scope.getResourceId().get());
    }
    return builder.toString();
  }

  public interface HookTriggersGetter {
    List<RepositoryHookTrigger> get();
  }

  private enum DeletionResult {
    MISSING_ID,
    MISSING_SCRIPT,
    OK;

    public String getMessage() {
      if (this == MISSING_ID) {
        return "was not installed";
      }

      if (this == MISSING_SCRIPT) {
        return "hook script already gone";
      }

      return "success";
    }
  }
}

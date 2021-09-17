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
    delete(getLegacyPluginSettingsPath(scope), new CallbackLogDelete() {
      @Override
      public void onSuccess(String hookKey, Long id) {
        log.debug(
            "deleted legacy/project hook script {}: id={} {}",
            hookKey,
            id,
            ScopeUtil.toString(scope));
      }

      @Override
      public void onMissingId(String hookKey) {
        //
      }

      @Override
      public void onMissingScript(String hookKey, Long id) {
        log.debug(
            "did not delete legacy/project hook script {}: id={} {} because the script is"
                + " already gone",
            hookKey,
            id,
            ScopeUtil.toString(scope));
      }
    });
  }

  public void uninstall(ProjectScope parent, RepositoryScope scope) {
    delete(getPluginSettingsPath(parent, scope), new CallbackLogDelete() {
      @Override
      public void onSuccess(String hookKey, Long id) {
        log.debug(
            "deleted project/repository hook script {}: id={} {} {}",
            hookKey,
            id,
            ScopeUtil.toString(parent),
            ScopeUtil.toString(scope));
      }

      @Override
      public void onMissingId(String hookKey) {
        //
      }

      @Override
      public void onMissingScript(String hookKey, Long id) {
        log.debug(
            "did not delete project/repository hook script {}: id={} {} {}"
                + " because the script is already gone",
            hookKey,
            id,
            ScopeUtil.toString(parent),
            ScopeUtil.toString(scope));
      }
    });
  }

  public void uninstall(GlobalScope parentScope, RepositoryScope scope) {
    delete(getPluginSettingsPath(parentScope, scope), new CallbackLogDelete() {
      @Override
      public void onSuccess(String hookKey, Long id) {
        log.debug(
            "deleted global/repository hook script {}: id={} {}",
            hookKey,
            id,
            ScopeUtil.toString(scope));
      }

      @Override
      public void onMissingId(String hookKey) {
        //
      }

      @Override
      public void onMissingScript(String hookKey, Long id) {
        log.debug(
            "did not delete global/repository hook script {}: id={} {} because the"
                + " script is already gone",
            hookKey,
            id,
            ScopeUtil.toString(scope));
      }
    });
  }

  public void uninstall(RepositoryScope scope) {
    delete(getPluginSettingsPath(scope), new CallbackLogDelete() {
      @Override
      public void onSuccess(String hookKey, Long id) {
        log.debug(
            "deleted repository hook script {}: id={} {}", hookKey, id, ScopeUtil.toString(scope));
      }

      @Override
      public void onMissingId(String hookKey) {
        //
      }

      @Override
      public void onMissingScript(String hookKey, Long id) {
        log.debug(
            "did not delete repository hook script {}: id={} {} because the script is"
                + " already gone",
            hookKey,
            id,
            ScopeUtil.toString(scope));
      }
    });
  }

  private void delete(String path, CallbackLogDelete logger) {
    Optional<Long> id = readHookScriptId(path);
    if (id.isPresent()) {
      boolean deleted = deleteHookScript(id.get());
      if (deleted) {
        logger.onSuccess(hookKey, id.get());
      } else {
        logger.onMissingScript(hookKey, id.get());
      }

      pluginSettings.remove(path);
    } else {
      logger.onMissingId(hookKey);
    }
  }

  public void install(
      @Nonnull Settings settings, @Nonnull ProjectScope parent, @Nonnull RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(parent, scope);
    Pair<HookScript, List<RepositoryHookTrigger>> result =
        install(pluginSettingsPath, settings, scope);

    log.debug(
        "created project/repository hook script {}: id={} {} {} triggers={}",
        hookId,
        result.getLeft().getId(),
        ScopeUtil.toString(parent),
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
        "created global/repository hook script {}: id={} {} triggers={}",
        hookId,
        result.getLeft().getId(),
        ScopeUtil.toString(scope),
        listTriggers(result.getRight()));
  }

  public void install(@Nonnull Settings settings, @Nonnull RepositoryScope scope) {
    String pluginSettingsPath = getPluginSettingsPath(scope);
    Pair<HookScript, List<RepositoryHookTrigger>> result =
        install(pluginSettingsPath, settings, scope);

    log.debug(
        "created repository hook script {}: id={} {} triggers={}",
        hookId,
        result.getLeft().getId(),
        ScopeUtil.toString(scope),
        listTriggers(result.getRight()));
  }

  private Pair<HookScript, List<RepositoryHookTrigger>> install(
      String pluginSettingsPath, @Nonnull Settings settings, @Nonnull RepositoryScope scope) {
    Optional<Long> hookScriptId = readHookScriptId(pluginSettingsPath);
    if (hookScriptId.isPresent()) {
      deleteHookScript(hookScriptId.get());
    }

    HookScript hookScript = create(pluginSettingsPath, settings);

    pluginSettings.put(pluginSettingsPath, String.valueOf(hookScript.getId()));

    List<RepositoryHookTrigger> triggers = getRepositoryHookTriggers.get();

    HookScriptSetConfigurationRequest.Builder configBuilder =
        new HookScriptSetConfigurationRequest.Builder(hookScript, scope);
    configBuilder.triggers(triggers);

    HookScriptSetConfigurationRequest configRequest = configBuilder.build();
    hookScriptService.setConfiguration(configRequest);

    return Pair.of(hookScript, triggers);
  }

  public Optional<Long> readHookScriptId(String pluginSettingsPath) {
    Object id = pluginSettings.get(pluginSettingsPath);
    if (id != null) {
      return Optional.of(Long.valueOf(id.toString()));
    }

    return Optional.empty();
  }

  private Optional<HookScript> getHookScript(Long id) {
    return hookScriptService.findById(id);
  }

  private boolean deleteHookScript(Long id) {
    Optional<HookScript> maybeHookScript = getHookScript(id);
    if (maybeHookScript.isPresent()) {
      return securityService
          .withPermission(Permission.SYS_ADMIN, "atlassian-external-hooks: delete hook script")
          .call(() -> {
            hookScriptService.delete(maybeHookScript.get());
            return true;
          });
    }

    return false;
  }

  private HookScript create(String tag, Settings settings) {
    String script = getScriptContents(tag, settings);

    HookScriptCreateRequest.Builder builder = new HookScriptCreateRequest.Builder(
            this.hookId, Const.PLUGIN_KEY, this.hookScriptType)
        .content(script);
    HookScriptCreateRequest hookScriptCreateRequest = builder.build();

    return securityService
        .withPermission(
            Permission.SYS_ADMIN, "atlassian-external-hooks: create low-level hook script")
        .call(() -> hookScriptService.create(hookScriptCreateRequest));
  }

  private String getScriptContents(String tag, Settings settings) {
    File executable =
        this.getExecutable(settings.getString("exe", ""), settings.getBoolean("safe_path", false));

    Boolean async = settings.getBoolean("async", false);

    StringBuilder scriptBuilder = new StringBuilder();
    scriptBuilder.append(this.hookScriptTemplate).append("\n\n");

    scriptBuilder.append("# com.ngs.stash.externalhooks tag: " + tag + "\n\n");

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

  public interface CallbackLogDelete {
    void onSuccess(String hookKey, Long id);

    void onMissingId(String hookKey);

    void onMissingScript(String hookKey, Long id);
  }
}

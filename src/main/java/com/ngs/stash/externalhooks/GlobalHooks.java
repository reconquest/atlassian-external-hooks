package com.ngs.stash.externalhooks;

import java.util.List;

import com.atlassian.bitbucket.project.ProjectType;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsBuilder;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;

public class GlobalHooks {
  private GlobalHookSettings preReceive = null;
  private GlobalHookSettings postReceive = null;
  private GlobalHookSettings mergeCheck = null;

  public GlobalHooks(List<GlobalHookSettings> settings) {
    settings.stream().forEach((setting) -> {
      if (setting.getHook().equals(Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID)) {
        this.preReceive = setting;
      }
      if (setting.getHook().equals(Const.PLUGIN_KEY + ":" + Const.POST_RECEIVE_HOOK_ID)) {
        this.postReceive = setting;
      }
      if (setting.getHook().equals(Const.PLUGIN_KEY + ":" + Const.MERGE_CHECK_HOOK_ID)) {
        this.mergeCheck = setting;
      }
    });
  }

  public boolean isEnabled(String hookKey) {
    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID)) {
      return this.isPreReceiveEnabled();
    }

    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.POST_RECEIVE_HOOK_ID)) {
      return this.isPostReceiveEnabled();
    }

    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.MERGE_CHECK_HOOK_ID)) {
      return this.isMergeCheckEnabled();
    }

    return false;
  }

  public boolean isEligible(String hookKey, RepositoryScope scope) {
    GlobalHookSettings globalHook = this.getHook(hookKey);
    FilterPersonalRepositories filter =
        FilterPersonalRepositories.fromId(globalHook.getFilterPersonalRepositories());
    if (filter == null) {
      filter = FilterPersonalRepositories.DISABLED;
    }

    boolean isPersonal = scope.getProject().getType() == ProjectType.PERSONAL;

    if (filter == FilterPersonalRepositories.DISABLED
        || (filter == FilterPersonalRepositories.ONLY_PERSONAL && isPersonal)
        || (filter == FilterPersonalRepositories.EXCLUDE_PERSONAL && !isPersonal)) {
      return true;
    }

    return false;
  }

  public GlobalHookSettings getHook(String hookKey) {
    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID)) {
      return this.preReceive;
    }

    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.POST_RECEIVE_HOOK_ID)) {
      return this.postReceive;
    }

    if (hookKey.equals(Const.PLUGIN_KEY + ":" + Const.MERGE_CHECK_HOOK_ID)) {
      return this.mergeCheck;
    }

    return null;
  }

  public boolean isPreReceiveEnabled() {
    return this.isEnabled(this.preReceive);
  }

  public boolean isPostReceiveEnabled() {
    return this.isEnabled(this.postReceive);
  }

  public boolean isMergeCheckEnabled() {
    return this.isEnabled(this.mergeCheck);
  }

  private boolean isEnabled(GlobalHookSettings hook) {
    return hook != null && hook.getEnabled();
  }

  public Settings getSettings(GlobalHookSettings hook) {
    if (hook == null) {
      return null;
    }

    SettingsBuilder settingsBuilder = new SimpleSettingsBuilder();
    settingsBuilder.add("safe_path", hook.getSafePath());
    settingsBuilder.add("async", hook.getAsync());
    if (hook.getExe() != null) {
      settingsBuilder.add("exe", hook.getExe());
    }
    if (hook.getParams() != null) {
      settingsBuilder.add("params", hook.getParams());
    }

    return settingsBuilder.build();
  }

  public Settings getSettings(String hookKey) {
    return getSettings(getHook(hookKey));
  }
}

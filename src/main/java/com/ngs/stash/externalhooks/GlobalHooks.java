package com.ngs.stash.externalhooks;

import java.util.List;

import com.atlassian.bitbucket.setting.Settings;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;

public class GlobalHooks {
  private GlobalHookSettings preReceive = null;
  private GlobalHookSettings postReceive = null;
  private GlobalHookSettings mergeCheck = null;

  public GlobalHooks(List<GlobalHookSettings> settings) {
    settings.stream().forEach((setting) -> {
      if (setting.getHook() == Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID) {
        this.preReceive = setting;
      }
      if (setting.getHook() == Const.PLUGIN_KEY + ":" + Const.POST_RECEIVE_HOOK_ID) {
        this.postReceive = setting;
      }
      if (setting.getHook() == Const.PLUGIN_KEY + ":" + Const.MERGE_CHECK_HOOK_ID) {
        this.mergeCheck = setting;
      }
    });
  }

  public boolean isEnabled(String hookKey) {
    if (hookKey == Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID) {
      return this.isPreReceiveEnabled();
    }

    if (hookKey == Const.PLUGIN_KEY + ":" + Const.POST_RECEIVE_HOOK_ID) {
      return this.isPostReceiveEnabled();
    }

    if (hookKey == Const.PLUGIN_KEY + ":" + Const.MERGE_CHECK_HOOK_ID) {
      return this.isMergeCheckEnabled();
    }

    return false;
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
    return hook == null || hook.getEnabled();
  }

  public Settings getSettings(String hookKey) {
    // TODO
    return null;
  }
}

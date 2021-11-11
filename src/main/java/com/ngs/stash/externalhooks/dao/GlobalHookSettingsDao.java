package com.ngs.stash.externalhooks.dao;

import java.util.Arrays;
import java.util.List;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;

import net.java.ao.Query;

public class GlobalHookSettingsDao {
  private ActiveObjects ao;

  public GlobalHookSettingsDao(ActiveObjects ao) {
    this.ao = ao;
  }

  public GlobalHookSettings create() {
    return ao.create(GlobalHookSettings.class);
  }

  public List<GlobalHookSettings> find() {
    GlobalHookSettings[] states =
        ao.find(GlobalHookSettings.class, Query.select().from(GlobalHookSettings.class));
    return Arrays.asList(states);
  }

  public GlobalHookSettings get(String hookKey) {
    GlobalHookSettings[] items = ao.find(
        GlobalHookSettings.class,
        Query.select().from(GlobalHookSettings.class).where("HOOK = ?", hookKey));
    if (items.length == 0) {
      return null;
    }
    return items[0];
  }
}

package com.ngs.stash.externalhooks.dao;

import java.util.Arrays;
import java.util.List;

import jakarta.inject.Inject;
import jakarta.inject.Named;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.ao.GlobalHookSettings;

import net.java.ao.Query;

@Named
public class GlobalHookSettingsDao {
  private ActiveObjects ao;

  @Inject
  public GlobalHookSettingsDao(@ComponentImport ActiveObjects ao) {
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

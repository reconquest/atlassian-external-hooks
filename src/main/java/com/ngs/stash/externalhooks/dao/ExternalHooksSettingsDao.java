package com.ngs.stash.externalhooks.dao;

import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.DefaultSettings;
import com.ngs.stash.externalhooks.ExternalHooksSettings;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExternalHooksSettingsDao {
  private static Logger log = LoggerFactory.getLogger(ExternalHooksSettingsDao.class);
  private PluginSettings pluginSettings;

  public ExternalHooksSettingsDao(PluginSettingsFactory pluginSettingsFactory) {
    this.pluginSettings = pluginSettingsFactory.createGlobalSettings();
  }

  public ExternalHooksSettingsDao(PluginSettings pluginSettings) {
    this.pluginSettings = pluginSettings;
  }

  public ExternalHooksSettings getSettings() {
    ExternalHooksSettings settings = new ExternalHooksSettings();

    settings.triggers.preReceive = getIds(getPreReceiveHookTriggers());
    settings.triggers.postReceive = getIds(getPostReceiveHookTriggers());
    settings.triggers.mergeCheck = getIds(getMergeCheckHookTriggers());

    return settings;
  }

  public ExternalHooksSettings getDefaultSettings() {
    ExternalHooksSettings settings = new ExternalHooksSettings();

    settings.triggers.preReceive = getIds(DefaultSettings.PreReceiveHookTriggers);
    settings.triggers.postReceive = getIds(DefaultSettings.PostReceiveHookTriggers);
    settings.triggers.mergeCheck = getIds(DefaultSettings.MergeCheckHookTriggers);

    return settings;
  }

  public void save(ExternalHooksSettings settings) {
    ExternalHooksSettings.Triggers triggers = settings.triggers;

    if (triggers.preReceive != null) {
      pluginSettings.put(
          getPluginSettingsKey("pre_receive"),
          sanitize(triggers.preReceive, DefaultSettings.PreReceiveHookTriggers));
    }

    if (triggers.postReceive != null) {
      pluginSettings.put(
          getPluginSettingsKey("post_receive"),
          sanitize(triggers.postReceive, DefaultSettings.PostReceiveHookTriggers));
    }

    if (triggers.mergeCheck != null) {
      pluginSettings.put(
          getPluginSettingsKey("merge_check"),
          sanitize(triggers.mergeCheck, DefaultSettings.MergeCheckHookTriggers));
    }
  }

  public List<RepositoryHookTrigger> getPreReceiveHookTriggers() {
    return getHookTriggers("pre_receive", DefaultSettings.PreReceiveHookTriggers);
  }

  public List<RepositoryHookTrigger> getPostReceiveHookTriggers() {
    return getHookTriggers("post_receive", DefaultSettings.PostReceiveHookTriggers);
  }

  public List<RepositoryHookTrigger> getMergeCheckHookTriggers() {
    return getHookTriggers("merge_check", DefaultSettings.MergeCheckHookTriggers);
  }

  private List<String> sanitize(List<String> items, List<RepositoryHookTrigger> defaults) {
    // while converting to RepositoryHookTrigger we will get rid of
    // invalid identifiers then we convert triggers back to strings
    return getIds(getHookTriggers(items, defaults));
  }

  private String getPluginSettingsKey(String component) {
    String prefix = Const.PLUGIN_KEY + ":global:settings:";

    return prefix + component;
  }

  @SuppressWarnings("unchecked")
  private List<RepositoryHookTrigger> getHookTriggers(
      String component, List<RepositoryHookTrigger> defaults) {

    Object raw = pluginSettings.get(getPluginSettingsKey(component));
    if (raw == null) {
      return defaults;
    }

    List<String> items = (List<String>) raw;
    return this.getHookTriggers(items, defaults);
  }

  private List<RepositoryHookTrigger> getHookTriggers(
      List<String> items, List<RepositoryHookTrigger> defaults) {
    List<RepositoryHookTrigger> result = new ArrayList<RepositoryHookTrigger>();
    for (String item : items) {
      Optional<StandardRepositoryHookTrigger> trigger = StandardRepositoryHookTrigger.fromId(item);
      if (trigger.isPresent()) {
        result.add(trigger.get());
      } else {
        log.warn("unrecognized hook trigger in settings: {}", item);
      }
    }

    if (result.size() == 0) {
      return defaults;
    }

    return result;
  }

  private List<String> getIds(List<RepositoryHookTrigger> items) {
    List<String> result = new ArrayList<String>();
    for (RepositoryHookTrigger item : items) {
      result.add(item.getId());
    }

    return result;
  }
}

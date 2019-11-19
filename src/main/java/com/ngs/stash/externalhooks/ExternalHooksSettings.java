package com.ngs.stash.externalhooks;

import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

import com.atlassian.bitbucket.hook.repository.RepositoryHookTrigger;
import com.atlassian.bitbucket.hook.repository.StandardRepositoryHookTrigger;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.ngs.stash.externalhooks.hook.ExternalAsyncPostReceiveHook;
import com.ngs.stash.externalhooks.hook.ExternalMergeCheckHook;
import com.ngs.stash.externalhooks.hook.ExternalPreReceiveHook;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@XmlRootElement
public class ExternalHooksSettings {
  private static Logger log = LoggerFactory.getLogger(ExternalHooksSettings.class.getSimpleName());

  @XmlElement(name = "triggers")
  public Triggers triggers;

  public class Triggers {
    @XmlElement(name = "pre_receive")
    public List<String> preReceive;

    private List<RepositoryHookTrigger> preReceiveHookTriggers;

    @XmlElement(name = "post_receive")
    public List<String> postReceive;

    private List<RepositoryHookTrigger> postReceiveHookTriggers;

    @XmlElement(name = "merge_check")
    public List<String> mergeCheck;

    private List<RepositoryHookTrigger> mergeCheckHookTriggers;

    /**
     * Retrieves list of {@link RepositoryHookTrigger} ready to use for creating {@link HookScript}
     * of {@link ExternalPreReceiveHook}.
     */
    public List<RepositoryHookTrigger> getPreReceiveHookTriggers() {
      return this.preReceiveHookTriggers;
    }

    /**
     * Retrieves list of {@link RepositoryHookTrigger} ready to use for creating {@link HookScript}
     * of {@link ExternalAsyncPostReceiveHook}.
     */
    public List<RepositoryHookTrigger> getPostReceiveHookTriggers() {
      return this.postReceiveHookTriggers;
    }

    /**
     * Retrieves list of {@link RepositoryHookTrigger} ready to use for creating {@link HookScript}
     * of {@link ExternalMergeCheckHook}.
     */
    public List<RepositoryHookTrigger> getMergeCheckHookTriggers() {
      return this.mergeCheckHookTriggers;
    }
  }

  public ExternalHooksSettings() {}

  public ExternalHooksSettings(PluginSettingsFactory pluginSettingsFactory) {
    load(pluginSettingsFactory.createGlobalSettings());
  }

  public ExternalHooksSettings(PluginSettings pluginSettings) {
    load(pluginSettings);
  }

  private void load(PluginSettings pluginSettings) {
    triggers = new Triggers();
    triggers.preReceiveHookTriggers =
        getHookTriggers(pluginSettings, "pre_receive", DefaultSettings.PreReceiveHookTriggers);
    triggers.postReceiveHookTriggers =
        getHookTriggers(pluginSettings, "post_receive", DefaultSettings.PostReceiveHookTriggers);
    triggers.mergeCheckHookTriggers =
        getHookTriggers(pluginSettings, "merge_check", DefaultSettings.MergeCheckHookTriggers);

    triggers.preReceive = getIds(triggers.preReceiveHookTriggers);
    triggers.postReceive = getIds(triggers.postReceiveHookTriggers);
    triggers.mergeCheck = getIds(triggers.mergeCheckHookTriggers);
  }

  /**
   * Saves received settings to internal db using {@link PluginSettings}.
   *
   * @param pluginSettings
   */
  public void save(PluginSettings pluginSettings) {
    pluginSettings.put(
        getPluginSettingsKey("pre_receive"),
        sanitize(triggers.preReceive, DefaultSettings.PreReceiveHookTriggers));
    pluginSettings.put(
        getPluginSettingsKey("post_receive"),
        sanitize(triggers.postReceive, DefaultSettings.PostReceiveHookTriggers));
    pluginSettings.put(
        getPluginSettingsKey("merge_check"),
        sanitize(triggers.mergeCheck, DefaultSettings.MergeCheckHookTriggers));

    triggers.preReceiveHookTriggers =
        getHookTriggers(triggers.preReceive, DefaultSettings.PreReceiveHookTriggers);
    triggers.postReceiveHookTriggers =
        getHookTriggers(triggers.postReceive, DefaultSettings.PostReceiveHookTriggers);
    triggers.mergeCheckHookTriggers =
        getHookTriggers(triggers.mergeCheck, DefaultSettings.MergeCheckHookTriggers);
  }

  public void save(PluginSettingsFactory pluginSettingsFactory) {
    save(pluginSettingsFactory.createGlobalSettings());
  }

  private List<String> sanitize(List<String> items, List<RepositoryHookTrigger> defaults) {
    // while converting to RepositoryHookTrigger we will get rid of
    // invalid identifiers then we convert triggers back to strings
    return getIds(getHookTriggers(items, defaults));
  }

  private String getPluginSettingsKey(String component) {
    String prefix = ExternalHooks.PLUGIN_KEY + ":global:settings:";

    return prefix + component;
  }

  @SuppressWarnings("unchecked")
  private List<RepositoryHookTrigger> getHookTriggers(
      PluginSettings pluginSettings, String component, List<RepositoryHookTrigger> defaults) {

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

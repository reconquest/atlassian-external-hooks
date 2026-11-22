package com.ngs.stash.externalhooks;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ExternalHooksSettings {
  @JsonProperty("triggers")
  public ExternalHookSettingsTriggers triggers = new ExternalHookSettingsTriggers();
}

package com.ngs.stash.externalhooks.rest;

import com.fasterxml.jackson.annotation.JsonProperty;

public class GlobalHookSettingsSchema {
  @JsonProperty("safe_path")
  public boolean safePath;

  @JsonProperty("exe")
  public String exe;

  @JsonProperty("params")
  public String params;

  @JsonProperty("async")
  public boolean async;

  @JsonProperty("enabled")
  public boolean enabled;

  @JsonProperty("filter_personal_repositories")
  public int filterPersonalRepositories;
}

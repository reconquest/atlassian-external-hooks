package com.ngs.stash.externalhooks;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ExternalHookSettingsTriggers {
  @JsonProperty("pre_receive")
  public List<String> pre_receive;

  @JsonProperty("post_receive")
  public List<String> post_receive;

  @JsonProperty("merge_check")
  public List<String> merge_check;
}

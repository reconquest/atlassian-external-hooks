package com.ngs.stash.externalhooks;

import java.util.List;

import javax.xml.bind.annotation.XmlElement;

public class ExternalHookSettingsTriggers {
  @XmlElement(name = "pre_receive")
  public List<String> pre_receive;

  @XmlElement(name = "post_receive")
  public List<String> post_receive;

  @XmlElement(name = "merge_check")
  public List<String> merge_check;
}

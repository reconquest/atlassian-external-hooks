package com.ngs.stash.externalhooks;

import java.util.List;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

@XmlRootElement
public class ExternalHooksSettings {
  @XmlElement(name = "triggers")
  public Triggers triggers;

  public class Triggers {
    @XmlElement(name = "pre_receive")
    public List<String> preReceive;

    @XmlElement(name = "post_receive")
    public List<String> postReceive;

    @XmlElement(name = "merge_check")
    public List<String> mergeCheck;
  }

  public ExternalHooksSettings() {
    this.triggers = new Triggers();
  }
}

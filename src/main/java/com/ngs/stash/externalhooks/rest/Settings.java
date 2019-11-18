package com.ngs.stash.externalhooks.rest;

import java.util.List;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

@XmlRootElement
public class Settings {
  @XmlElement(name = "triggers")
  public Triggers triggers;

  public class Triggers {
    @XmlElement(name = "pre_receive")
    public List<String> PreReceive;

    @XmlElement(name = "post_receive")
    public List<String> PostReceive;

    @XmlElement(name = "merge_check")
    public List<String> MergeCheck;
  }
}

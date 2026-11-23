package com.ngs.stash.externalhooks;

import jakarta.xml.bind.annotation.XmlElement;
import jakarta.xml.bind.annotation.XmlRootElement;

@XmlRootElement
public class ExternalHooksSettings {
  @XmlElement(name = "triggers")
  public ExternalHookSettingsTriggers triggers = new ExternalHookSettingsTriggers();
}

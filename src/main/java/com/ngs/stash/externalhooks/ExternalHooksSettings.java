package com.ngs.stash.externalhooks;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

@XmlRootElement
public class ExternalHooksSettings {
  @XmlElement(name = "triggers")
  public ExternalHookSettingsTriggers triggers = new ExternalHookSettingsTriggers();
}

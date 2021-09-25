package com.ngs.stash.externalhooks.rest;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

import com.ngs.stash.externalhooks.FilterPersonalRepositories;

@XmlRootElement
public class GlobalHookSettingsSchema {
  @XmlElement(name = "safe_path")
  public boolean safePath;

  @XmlElement(name = "exe")
  public String exe;

  @XmlElement(name = "params")
  public String params;

  @XmlElement(name = "async")
  public boolean async;

  @XmlElement(name = "enabled")
  public boolean enabled;

  @XmlElement(name = "filter_personal_repositories")
  public FilterPersonalRepositories filterPersonalRepositories;
}

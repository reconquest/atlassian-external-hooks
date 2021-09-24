package com.ngs.stash.externalhooks.ao;

import com.ngs.stash.externalhooks.FilterPersonalRepositories;

import net.java.ao.Entity;
import net.java.ao.schema.Table;

@Table("global_hook")
public interface GlobalHookSettings extends Entity {
  String getHook();

  void setHook(String value);

  boolean getEnabled();

  void setEnabled(boolean enabled);

  String getExe();

  void setExe(String value);

  String getParams();

  void setParams(String value);

  boolean getAsync();

  void setAsync(boolean value);

  boolean getSafePath();

  void setSafePath(boolean value);

  FilterPersonalRepositories getFilterPersonalRepositories();

  void setFilterPersonalRepositories(FilterPersonalRepositories value);
}

package com.ngs.stash.externalhooks.ao;

import net.java.ao.Entity;
import net.java.ao.Preload;
import net.java.ao.schema.Table;

@Table("factory_state")
@Preload({"STARTED", "FINISHED", "CURRENT", "TOTAL"})
public interface FactoryState extends Entity {
  boolean getStarted();

  boolean getFinished();

  int getCurrent();

  int getTotal();

  void setStarted(boolean started);

  void setFinished(boolean finished);

  void setCurrent(int current);

  void setTotal(int total);
}

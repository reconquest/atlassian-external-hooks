package com.ngs.stash.externalhooks.dao;

import com.atlassian.activeobjects.external.ActiveObjects;
import com.ngs.stash.externalhooks.ao.FactoryState;

import net.java.ao.Query;

public class FactoryStateDao {
  private ActiveObjects ao;

  public FactoryStateDao(ActiveObjects ao) {
    this.ao = ao;
  }

  public FactoryState create() {
    return ao.create(FactoryState.class);
  }

  public FactoryState find(Integer id) {
    FactoryState[] states =
        ao.find(FactoryState.class, Query.select().from(FactoryState.class).where("ID = ?", id));
    if (states.length == 0) {
      return null;
    }

    return states[0];
  }
}

package com.ngs.stash.externalhooks.rest;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

import com.ngs.stash.externalhooks.ao.FactoryState;

@XmlRootElement
public class FactoryStateResponse {
  @XmlElement private int id;
  @XmlElement private boolean started;
  @XmlElement private boolean finished;
  @XmlElement private int current;
  @XmlElement private int total;

  public FactoryStateResponse(FactoryState state) {
    id = state.getID();
    started = state.getStarted();
    finished = state.getFinished();
    current = state.getCurrent();
    total = state.getTotal();
  }

  public FactoryStateResponse(int id) {
    this.id = id;
  }
}

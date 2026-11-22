package com.ngs.stash.externalhooks.rest;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.ngs.stash.externalhooks.ao.FactoryState;

public class FactoryStateResponse {
  @JsonProperty
  private int id;

  @JsonProperty
  private boolean started;

  @JsonProperty
  private boolean finished;

  @JsonProperty
  private int current;

  @JsonProperty
  private int total;

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

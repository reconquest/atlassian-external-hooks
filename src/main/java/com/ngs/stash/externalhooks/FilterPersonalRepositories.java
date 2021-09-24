package com.ngs.stash.externalhooks;

public enum FilterPersonalRepositories {
  DISABLED(0),
  ONLY_PERSONAL(1),
  EXCLUDE_PERSONAL(2);

  private final int id;

  private FilterPersonalRepositories(int id) {
    this.id = id;
  }

  public static FilterPersonalRepositories fromId(int id) {
    for (FilterPersonalRepositories value : values()) {
      if (value.getId() == id) {
        return value;
      }
    }

    throw new IllegalArgumentException(
        "No FilterPersonalRepositories is associated with ID [" + id + "]");
  }

  public int getId() {
    return id;
  }
}

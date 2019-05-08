package com.ngs.stash.externalhooks.hook.helpers;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.repository.MinimalRef;
import com.atlassian.bitbucket.repository.RefChange;
import com.atlassian.bitbucket.repository.RefChangeType;

public class ExternalRefChange implements RefChange {
  MinimalRef ref;
  String refId;
  String fromHash;
  String toHash;
  RefChangeType type;

  public ExternalRefChange(
      String refId, String fromHash, String toHash, RefChangeType type, MinimalRef ref) {
    this.refId = refId;
    this.fromHash = fromHash;
    this.toHash = toHash;
    this.type = type;
    this.ref = ref;
  }

  @Nonnull
  public String getRefId() {
    return refId;
  }

  @Nonnull
  @Override
  public MinimalRef getRef() {
    return ref;
  }

  @Nonnull
  @Override
  public String getFromHash() {
    return fromHash;
  }

  @Nonnull
  @Override
  public String getToHash() {
    return toHash;
  }

  @Nonnull
  @Override
  public RefChangeType getType() {
    return type;
  }
}

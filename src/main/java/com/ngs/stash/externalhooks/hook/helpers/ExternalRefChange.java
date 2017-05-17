package com.ngs.stash.externalhooks.hook.helpers;

import com.atlassian.bitbucket.repository.RefChange;
import com.atlassian.bitbucket.repository.RefChangeType;
import com.atlassian.bitbucket.repository.MinimalRef;

import javax.annotation.Nonnull;

public class ExternalRefChange implements RefChange {
    MinimalRef ref;
    String fromHash;
    String toHash;
    RefChangeType type;

    public ExternalRefChange(MinimalRef ref, String fromHash, String toHash, RefChangeType type) {
        this.ref = ref;
        this.fromHash = fromHash;
        this.toHash = toHash;
        this.type = type;
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

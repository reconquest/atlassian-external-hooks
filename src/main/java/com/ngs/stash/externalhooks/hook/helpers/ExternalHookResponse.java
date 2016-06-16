package com.ngs.stash.externalhooks.hook.helpers;

import java.io.PrintWriter;
import com.atlassian.bitbucket.hook.HookResponse;

import javax.annotation.Nonnull;

public class ExternalHookResponse implements HookResponse {
    PrintWriter outWriter;
    PrintWriter errWriter;

    public ExternalHookResponse(PrintWriter outWriter, PrintWriter errWriter) {
        this.outWriter = outWriter;
        this.errWriter = errWriter;
    }

    @Nonnull
    @Override
    public PrintWriter out() {
        return outWriter;
    }

    @Nonnull
    @Override
    public PrintWriter err() {
        return errWriter;
    }
}

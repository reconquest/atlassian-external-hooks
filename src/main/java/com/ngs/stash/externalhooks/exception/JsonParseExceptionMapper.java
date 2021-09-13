package com.ngs.stash.externalhooks.exception;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.ext.ExceptionMapper;
import javax.ws.rs.ext.Provider;

import com.google.gson.JsonParseException;

@Provider
public class JsonParseExceptionMapper implements ExceptionMapper<JsonParseException> {
  @Override
  public Response toResponse(JsonParseException exception) {
    exception.printStackTrace();
    return Response.status(Response.Status.BAD_REQUEST)
        .entity(
            "This is an invalid request. At least one field format is not readable by the system.")
        .type(MediaType.TEXT_PLAIN)
        .build();
  }
}

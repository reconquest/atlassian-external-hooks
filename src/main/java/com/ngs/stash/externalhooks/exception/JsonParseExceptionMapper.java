package com.ngs.stash.externalhooks.exception;

import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;
import jakarta.ws.rs.ext.ExceptionMapper;
import jakarta.ws.rs.ext.Provider;

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

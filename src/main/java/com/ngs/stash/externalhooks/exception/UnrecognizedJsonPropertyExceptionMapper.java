package com.ngs.stash.externalhooks.exception;

import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;
import jakarta.ws.rs.ext.ExceptionMapper;
import jakarta.ws.rs.ext.Provider;

import com.fasterxml.jackson.databind.exc.UnrecognizedPropertyException;

@Provider
public class UnrecognizedJsonPropertyExceptionMapper
    implements ExceptionMapper<UnrecognizedPropertyException> {
  @Override
  public Response toResponse(UnrecognizedPropertyException exception) {
    return Response.status(Response.Status.BAD_REQUEST)
        .entity("This is an invalid request. The field "
            + exception.getPropertyName()
            + " is not recognized by the system.")
        .type(MediaType.TEXT_PLAIN)
        .build();
  }
}

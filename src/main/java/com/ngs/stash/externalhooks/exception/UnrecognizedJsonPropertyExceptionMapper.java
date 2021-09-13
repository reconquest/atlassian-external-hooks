package com.ngs.stash.externalhooks.exception;


import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.ext.ExceptionMapper;
import javax.ws.rs.ext.Provider;

import org.codehaus.jackson.map.exc.UnrecognizedPropertyException;

@Provider
public class UnrecognizedJsonPropertyExceptionMapper
    implements ExceptionMapper<UnrecognizedPropertyException> {
  @Override
  public Response toResponse(UnrecognizedPropertyException exception) {
    return Response.status(Response.Status.BAD_REQUEST)
        .entity("This is an invalid request. The field "
            + exception.getUnrecognizedPropertyName()
            + " is not recognized by the system.")
        .type(MediaType.TEXT_PLAIN)
        .build();
  }
}

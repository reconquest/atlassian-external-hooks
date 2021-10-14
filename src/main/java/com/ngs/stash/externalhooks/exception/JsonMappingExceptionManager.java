package com.ngs.stash.externalhooks.exception;

import java.util.stream.Collectors;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.ext.ExceptionMapper;
import javax.ws.rs.ext.Provider;

import org.codehaus.jackson.map.JsonMappingException;

@Provider
public class JsonMappingExceptionManager implements ExceptionMapper<JsonMappingException> {
  @Override
  public Response toResponse(JsonMappingException exception) {
    exception.printStackTrace();
    return Response.status(Response.Status.BAD_REQUEST)
        .entity("This is an invalid request. The field '"
            + exception.getPath().stream()
                .map((x) -> x.getFieldName())
                .collect(Collectors.joining("."))
            + "' is not recognized by the system: " + exception.getMessage())
        .type(MediaType.TEXT_PLAIN)
        .build();
  }
}

package com.ngs.stash.externalhooks.rest;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import java.util.List;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ngs.stash.externalhooks.ExternalHooksSettings;
import com.ngs.stash.externalhooks.SimpleSettingsValidationErrors;

import org.junit.Test;

public class RestJsonTest {
  private final ObjectMapper mapper = new ObjectMapper();

  @Test
  public void globalHookSettingsRoundTrip() throws Exception {
    String json = "{\"safe_path\":true,\"exe\":\"check.sh\",\"params\":\"one two\","
        + "\"async\":true,\"enabled\":true,\"filter_personal_repositories\":2}";

    GlobalHookSettingsSchema settings = mapper.readValue(json, GlobalHookSettingsSchema.class);

    assertTrue(settings.safePath);
    assertEquals("check.sh", settings.exe);
    assertEquals("one two", settings.params);
    assertTrue(settings.async);
    assertTrue(settings.enabled);
    assertEquals(2, settings.filterPersonalRepositories);
    assertEquals(mapper.readTree(json), mapper.valueToTree(settings));
  }

  @Test
  public void triggerSettingsRoundTrip() throws Exception {
    String json = "{\"triggers\":{\"pre_receive\":[\"repo-push\"],"
        + "\"post_receive\":[\"repo-push\"],\"merge_check\":[\"pull-request-merge\"]}}";

    ExternalHooksSettings settings = mapper.readValue(json, ExternalHooksSettings.class);

    assertEquals(List.of("repo-push"), settings.triggers.pre_receive);
    assertEquals(List.of("repo-push"), settings.triggers.post_receive);
    assertEquals(List.of("pull-request-merge"), settings.triggers.merge_check);
    assertEquals(mapper.readTree(json), mapper.valueToTree(settings));
  }

  @Test
  public void factoryStateContainsProgress() {
    JsonNode state = mapper.valueToTree(new FactoryStateResponse(42));

    assertEquals(5, state.size());
    assertEquals(42, state.get("id").asInt());
    assertFalse(state.get("started").asBoolean());
    assertFalse(state.get("finished").asBoolean());
    assertEquals(0, state.get("current").asInt());
    assertEquals(0, state.get("total").asInt());
  }

  @Test
  public void validationErrorsContainFormAndFields() {
    SimpleSettingsValidationErrors errors = new SimpleSettingsValidationErrors();
    errors.addFormError("Invalid settings");
    errors.addFieldError("exe", "Executable is blank");

    JsonNode response = mapper.valueToTree(new FormValidationErrors(errors));

    assertEquals(2, response.size());
    assertEquals("Invalid settings", response.get("errors_form").get(0).asText());
    assertEquals("Executable is blank", response.get("errors_fields").get("exe").get(0).asText());
  }
}

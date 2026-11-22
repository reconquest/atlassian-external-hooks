package com.ngs.stash.externalhooks.rest;

import java.util.Collection;
import java.util.Map;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.ngs.stash.externalhooks.SimpleSettingsValidationErrors;

public class FormValidationErrors {
  @JsonProperty("errors_form")
  public Collection<String> form;

  @JsonProperty("errors_fields")
  public Map<String, Collection<String>> fields;

  public FormValidationErrors(SimpleSettingsValidationErrors errors) {
    this.form = errors.getFormErrors();
    this.fields = errors.getFieldErrors();
  }
}

package com.ngs.stash.externalhooks.rest;

import java.util.Collection;
import java.util.Map;

import javax.xml.bind.annotation.XmlElement;

import com.ngs.stash.externalhooks.SimpleSettingsValidationErrors;

public class FormValidationErrors {
  @XmlElement(name = "errors_form")
  public Collection<String> form;

  @XmlElement(name = "errors_fields")
  public Map<String, Collection<String>> fields;

  public FormValidationErrors(SimpleSettingsValidationErrors errors) {
    this.form = errors.getFormErrors();
    this.fields = errors.getFieldErrors();
  }
}

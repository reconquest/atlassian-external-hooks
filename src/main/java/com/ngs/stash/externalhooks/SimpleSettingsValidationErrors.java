package com.ngs.stash.externalhooks;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collection;
import java.util.HashMap;
import java.util.Map;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.validation.FormErrors;

// This is a dead simple copy of what Bitbucket's SimpleSettingsValidationErrors does
public class SimpleSettingsValidationErrors implements SettingsValidationErrors, FormErrors {

  private final HashMap<String, Collection<String>> fields = new HashMap<>();
  private final Collection<String> form = new ArrayList<String>();

  @Override
  public void addFieldError(@Nonnull String fieldName, @Nonnull String errorMessage) {
    Collection<String> list = fields.get(fieldName);
    if (list == null) {
      fields.put(fieldName, Arrays.asList(errorMessage));
      return;
    }

    list.add(errorMessage);
  }

  @Override
  public void addFormError(@Nonnull String errorMessage) {
    this.form.add(errorMessage);
  }

  @Nonnull
  @Override
  public Collection<String> getErrors(@Nonnull String field) {
    return fields.get(field);
  }

  @Nonnull
  @Override
  public Map<String, Collection<String>> getFieldErrors() {
    return this.fields;
  }

  @Nonnull
  @Override
  public Collection<String> getFormErrors() {
    return this.form;
  }

  @Override
  public boolean isEmpty() {
    return fields.isEmpty();
  }
}

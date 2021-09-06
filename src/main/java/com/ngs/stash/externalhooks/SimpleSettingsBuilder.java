package com.ngs.stash.externalhooks;

import static com.google.common.base.Preconditions.checkArgument;
import static com.google.common.base.Preconditions.checkNotNull;

import java.util.Map;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsBuilder;
import com.google.common.collect.ImmutableMap;

public class SimpleSettingsBuilder implements SettingsBuilder {

  private final ImmutableMap.Builder<String, Object> builder = ImmutableMap.builder();

  @Nonnull
  @Override
  public SettingsBuilder add(@Nonnull String key, @Nonnull String value) {
    builder.put(key, value);
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder add(@Nonnull String key, boolean value) {
    builder.put(key, value);
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder add(@Nonnull String key, int value) {
    builder.put(key, value);
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder add(@Nonnull String key, long value) {
    builder.put(key, value);
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder add(@Nonnull String key, double value) {
    builder.put(key, value);
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder addAll(@Nonnull Map<String, ?> values) {
    for (Map.Entry<String, ?> entry : values.entrySet()) {
      String key = checkNotNull(entry.getKey(), "key");
      Object value = entry.getValue();
      if (value == null) {
        continue;
      }
      // In the future we may want to support lists of values.
      // I doubt there is a need for nested structures.
      checkArgument(
          value instanceof String
              || value instanceof Boolean
              || value instanceof Integer
              || value instanceof Long
              || value instanceof Double,
          "value of type %s is not supported",
          value.getClass());
      builder.put(key, value);
    }
    return this;
  }

  @Nonnull
  @Override
  public SettingsBuilder addAll(@Nonnull Settings settings) {
    // The settings object should already have safe values in it
    // so don't need to worry checking them
    builder.putAll(settings.asMap());
    return this;
  }

  @Nonnull
  @Override
  public Settings build() {
    return new SimpleSettings(builder.build());
  }
}

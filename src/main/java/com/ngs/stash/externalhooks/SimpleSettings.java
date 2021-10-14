package com.ngs.stash.externalhooks;

import static com.google.common.base.Preconditions.checkNotNull;

import java.util.Map;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.setting.Settings;
import com.google.common.base.MoreObjects;
import com.google.common.collect.ImmutableMap;

public class SimpleSettings implements Settings {
  private final Map<String, Object> values;

  SimpleSettings(Map<String, Object> values) {
    this.values = ImmutableMap.copyOf(values);
  }

  @SuppressWarnings("unchecked")
  private <T> T get(String key) {
    checkNotNull(key, "key");
    return (T) values.get(key);
  }

  @Override
  public String getString(@Nonnull String key) {
    return get(key);
  }

  @Nonnull
  @Override
  public String getString(@Nonnull String key, @Nonnull String defaultValue) {
    return MoreObjects.firstNonNull(getString(key), defaultValue);
  }

  @Override
  public Boolean getBoolean(@Nonnull String key) {
    try {
      return get(key);
    } catch (ClassCastException e) {
      return Boolean.parseBoolean(getString(key));
    }
  }

  @Override
  public boolean getBoolean(@Nonnull String key, boolean defaultValue) {
    return MoreObjects.firstNonNull(getBoolean(key), defaultValue);
  }

  private Number getNumber(String key) {
    return get(key);
  }

  @Override
  public Integer getInt(@Nonnull String key) {
    try {
      Number value = getNumber(key);
      return value != null ? value.intValue() : null;
    } catch (ClassCastException e) {
      return Integer.parseInt(getString(key));
    }
  }

  @Override
  public int getInt(@Nonnull String key, int defaultValue) {
    return MoreObjects.firstNonNull(getInt(key), defaultValue);
  }

  @Override
  public Long getLong(@Nonnull String key) {
    try {
      Number value = getNumber(key);
      return value != null ? value.longValue() : null;
    } catch (ClassCastException e) {
      return Long.parseLong(getString(key));
    }
  }

  @Override
  public long getLong(@Nonnull String key, long defaultValue) {
    return MoreObjects.firstNonNull(getLong(key), defaultValue);
  }

  @Override
  public Double getDouble(@Nonnull String key) {
    try {
      Number value = getNumber(key);
      return value != null ? value.doubleValue() : null;
    } catch (ClassCastException e) {
      return Double.parseDouble(getString(key));
    }
  }

  @Override
  public double getDouble(@Nonnull String key, double defaultValue) {
    return MoreObjects.firstNonNull(getDouble(key), defaultValue);
  }

  @Override
  public Map<String, Object> asMap() {
    return values;
  }
}

package org.timo.gitconfig;

import java.util.AbstractMap;
import java.util.Iterator;
import java.util.Map.Entry;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * @author Timoteo Ponce
 *
 */
public class ConfigurationIterator implements Iterator<Entry<String, String>> {

    private final Configuration configuration;

    private final Iterator<String> keySetIterator;

    private String currentKey;

    public ConfigurationIterator(final Configuration configuration) {
        this.configuration = configuration;
        this.keySetIterator = this.configuration.getKeySet().iterator();
    }

    @Override
    public boolean hasNext() {
        return keySetIterator.hasNext();
    }

    @Override
    public Entry<String, String> next() {
        currentKey = keySetIterator.next();
        final String value = configuration.getValue(currentKey);
        return new AbstractMap.SimpleEntry(currentKey, value);
    }

    @Override
    public void remove() {
        if (currentKey != null && !currentKey.isEmpty()) {
            configuration.remove(currentKey);
            currentKey = null;
        }
    }

}

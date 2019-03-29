package org.timo.gitconfig;

import java.util.Collection;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * @author Timoteo Ponce
 *
 */
public class Section {

    private String name;

    private final Map<String, String> variables = new HashMap<String, String>();

    public Section(final String name) {
        this.name = name;
    }

    public void setVariable(final String key, final String value) {
        variables.put(key, value);
    }

    public String getVariable(final String key) {
        String value = variables.get(key);
        if (value == null) {
            value = "";
        }
        return value;
    }

    public String getName() {
        return name;
    }

    public void setName(final String name) {
        this.name = name;
    }

    public Map<String, String> getVariables() {
        final Map<String, String> outputVars = new HashMap<String, String>(
                variables);
        return outputVars;
    }

    public void removeVariable(final String key) {
        variables.remove(key);
    }

    public Set<String> getKeySet() {
        return variables.keySet();
    }

    public boolean isEmpty() {
        return variables.isEmpty();
    }

    public Collection<String> getValues() {
        return variables.values();
    }

    @Override
    public int hashCode() {
        final int prime = 31;
        int result = 1;
        result = prime * result + ((name == null) ? 0 : name.hashCode());
        return result;
    }

    @Override
    public boolean equals(final Object obj) {
        if (this == obj) {
            return true;
        }
        if (obj == null) {
            return false;
        }
        if (getClass() != obj.getClass()) {
            return false;
        }
        final Section other = (Section) obj;
        if (name == null) {
            if (other.name != null) {
                return false;
            }
        } else if (!name.equals(other.name)) {
            return false;
        }
        return true;
    }

}

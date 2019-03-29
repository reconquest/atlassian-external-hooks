package org.timo.gitconfig;

import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.Map.Entry;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * @author Timoteo Ponce
 *
 */
class RootSection extends Section {

    private final Map<String, Section> sectionMap = new HashMap<String, Section>();

    public RootSection(final String name) {
        super(name);
    }

    public void setSection(final Section section) {
        if (section instanceof RootSection) {
            throw new IllegalArgumentException(
                    "Nested RootSections are not supported.");
        }
        sectionMap.put(section.getName(), section);
    }

    public Section getSection(final String name) {
        return sectionMap.get(name);
    }

    public Section getOrCreateSection(final String name) {
        Section section = sectionMap.get(name);
        if (section == null) {
            section = new Section(name);
            sectionMap.put(name, section);
        }
        return section;
    }

    public Section removeSection(final String subSectionName) {
        return sectionMap.remove(subSectionName);
    }

    /**
     * Returns all variables keys as variable paths. e.g.
     *
     * <pre>
     * core.editor
     * core.editor.command
     * </pre>
     *
     * @return variables keys as variable paths. e.g. core.editor.command
     */
    public Set<String> getAllKeySet() {
        final Set<String> keySet = getLocalizedKeySet();
        for (final Section subSection : sectionMap.values()) {
            final Set<String> sectionKeySet = subSection.getKeySet();
            for (final String sectionKey : sectionKeySet) {
                keySet.add(getName() + "." + subSection.getName() + "."
                        + sectionKey);
            }
        }
        return keySet;
    }

    private Set<String> getLocalizedKeySet() {
        final Set<String> keySet = new HashSet<String>();
        for (final String key : super.getKeySet()) {
            keySet.add(getName() + "." + key);
        }
        return keySet;
    }

    /**
     * Returns all variables of section as a Map<Key,Value>. It will return all
     * variables keys as variable paths. e.g.
     *
     * <pre>
     * core.editor = emacs
     * core.editor.command = /usr/bin/emacs
     * </pre>
     *
     * @return Map containing all variables using their paths as keys e.g.
     *         core.editor = emacs
     */
    public Map<String, String> getAllVariables() {
        final Map<String, String> variables = getLocalizedVariables();
        for (final Section subSection : sectionMap.values()) {
            for (final Entry<String, String> subVar : subSection.getVariables()
                    .entrySet()) {
                variables.put(getName() + "." + subSection.getName() + "."
                        + subVar.getKey(), subVar.getValue());
            }
        }
        return variables;
    }

    private Map<String, String> getLocalizedVariables() {
        final Map<String, String> variables = new HashMap<String, String>();
        for (final Entry<String, String> var : super.getVariables().entrySet()) {
            variables.put(getName() + "." + var.getKey(), var.getValue());
        }
        return variables;
    }

    /**
     * @return
     */
    public Collection<String> getAllValues() {
        final Collection<String> values = new ArrayList<String>(super
                .getValues());
        for (final Section subSection : sectionMap.values()) {
            values.addAll(subSection.getValues());
        }
        return values;
    }

    public boolean isAllEmpty() {
        boolean isEmpty = super.isEmpty() && sectionMap.isEmpty();
        if (!isEmpty) {
            for (final Section section : sectionMap.values()) {
                if (section.isEmpty()) {
                    isEmpty = true;
                    break;
                }
            }
        }
        return isEmpty;
    }

    public Collection<Section> getSections() {
        return sectionMap.values();
    }

}

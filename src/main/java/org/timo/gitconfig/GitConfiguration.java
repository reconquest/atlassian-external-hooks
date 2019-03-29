package org.timo.gitconfig;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;
import java.util.logging.Logger;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * @author Timoteo Ponce
 *
 */
public class GitConfiguration implements Configuration {

    private static final Logger LOG = Logger.getLogger(GitConfiguration.class
            .getName());

    private final Map<String, RootSection> rootSectionsMap = new HashMap<String, RootSection>();

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getKeySet()
     */
    @Override
    public Set<String> getKeySet() {
        final Set<String> keySet = new HashSet<String>();
        for (final Entry<String, RootSection> entry : rootSectionsMap
                .entrySet()) {
            keySet.addAll(entry.getValue().getAllKeySet());
        }
        return keySet;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getTextContent()
     */
    @Override
    public String getTextContent() {
        final StringBuilder builder = new StringBuilder();
        for (final Entry<String, RootSection> entry : rootSectionsMap
                .entrySet()) {
            appendSection(builder, entry.getValue());
        }
        return builder.toString();
    }

    /**
     * @param builder
     * @param rootSection
     */
    private void appendSection(final StringBuilder builder,
                               final RootSection rootSection) {
        if (!rootSection.isEmpty()) {
            builder.append("[" + rootSection.getName() + "] \n");
        }
        appendVariables(builder, rootSection);
        appendSubSections(builder, rootSection);
    }

    /**
     * @param builder
     * @param section
     */
    private void appendVariables(final StringBuilder builder,
                                 final Section section) {
        for (final Entry<String, String> entry : section.getVariables()
                .entrySet()) {
            builder.append("\t\t " + entry.getKey() + " = " + entry.getValue()
                    + "\n");
        }
        if (section.getVariables().size() > 0) {
            builder.append("\n");
        }
    }

    /**
     * @param builder
     * @param rootSection
     */
    private void appendSubSections(final StringBuilder builder,
                                   final RootSection rootSection) {
        for (final Section section : rootSection.getSections()) {
            builder.append("[" + rootSection.getName() + " '"
                    + section.getName() + "'] \n");
            appendVariables(builder, section);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getValue(java.lang.String)
     */
    @Override
    public String getValue(final String composedKey) {
        final String[] keys = splitKeys(composedKey);
        if (keys.length < 2) {
            throw new IllegalArgumentException("Invalid variable key : "
                    + composedKey);
        }
        final String rootSection = keys[0];
        if (keys.length > 2) {
            return getValue(rootSection, keys[1], keys[2]);
        } else {
            return getValue(rootSection, keys[1]);
        }
    }

    /* (non-Javadoc)
     * @see org.timo.gitconfig.Configuration#getBoolean(java.lang.String)
     */
    @Override
    public Boolean getBoolean(final String composedKey) {
        return Boolean.valueOf(getValue(composedKey));
    }

    /* (non-Javadoc)
     * @see org.timo.gitconfig.Configuration#getDouble(java.lang.String)
     */
    @Override
    public Double getDouble(final String composedKey) {
        return Double.valueOf(getValue(composedKey));
    }

    /* (non-Javadoc)
     * @see org.timo.gitconfig.Configuration#getInt(java.lang.String)
     */
    @Override
    public Integer getInt(final String composedKey) {
        return Integer.parseInt(getValue(composedKey));
    }

    /* (non-Javadoc)
     * @see org.timo.gitconfig.Configuration#getLong(java.lang.String)
     */
    @Override
    public Long getLong(final String composedKey) {
        return Long.valueOf(getValue(composedKey));
    }

    /**
     * @param composedKey
     * @return
     */
    private String[] splitKeys(final String composedKey) {
        return composedKey.split("\\.");
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getValue(java.lang.String,
     * java.lang.String, java.lang.String)
     */
    @Override
    public String getValue(final String sectionName,
                           final String subSectionName, final String key) {
        final RootSection rootSection = rootSectionsMap.get(sectionName);
        String value = "";
        if (rootSection != null
                && rootSection.getSection(subSectionName) != null) {
            final Section subSection = rootSection.getSection(subSectionName);
            value = subSection.getVariable(key);
        }
        return value;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getValue(java.lang.String,
     * java.lang.String)
     */
    @Override
    public String getValue(final String sectionName, final String key) {
        final RootSection rootSection = rootSectionsMap.get(sectionName);
        String value = "";
        if (rootSection != null) {
            value = rootSection.getVariable(key);
        }
        return value;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getValues()
     */
    @Override
    public Collection<String> getValues() {
        final Collection<String> values = new ArrayList<String>();
        for (final RootSection rootSection : rootSectionsMap.values()) {
            values.addAll(rootSection.getAllValues());
        }
        return values;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getValues(java.lang.String)
     */
    @Override
    public Collection<String> getValues(final String composedKey) {
        final String[] keys = splitKeys(composedKey);
        final Collection<String> values = new ArrayList<String>();
        final RootSection rootSection = rootSectionsMap.get(keys[0]);

        if (rootSection != null) {
            if (keys.length == 1) {
                values.addAll(rootSection.getAllValues());
            } else {
                final Section subSection = rootSection.getSection(keys[1]);
                if (subSection != null) {
                    values.addAll(subSection.getValues());
                }
            }
        }
        return values;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#remove(java.lang.String)
     */
    @Override
    public void remove(final String composedKey) {
        final String[] keys = splitKeys(composedKey);
        if (keys.length == 2) {
            LOG.info("Removing rootSection variable : " + composedKey);
            remove(keys[0], keys[1]);
        } else {
            LOG.info("Removing subSection variable : " + composedKey);
            remove(keys[0], keys[1], keys[2]);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#remove(java.lang.String,
     * java.lang.String)
     */
    @Override
    public void remove(final String sectionName, final String key) {
        final String[] keys = splitKeys(sectionName);
        if (keys.length > 1) {
            remove(keys[0], keys[1], key);
        } else {
            final RootSection rootSection = rootSectionsMap.get(sectionName);
            if (rootSection != null) {
                rootSection.removeVariable(key);
            }
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#remove(java.lang.String,
     * java.lang.String, java.lang.String)
     */
    @Override
    public void remove(final String sectionName, final String subSectionName,
                       final String key) {
        final RootSection rootSection = rootSectionsMap.get(sectionName);
        if (rootSection != null
                && rootSection.getSection(subSectionName) != null) {
            final Section section = rootSection.getSection(subSectionName);
            section.removeVariable(key);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#removeSection(java.lang.String)
     */
    @Override
    public void removeSection(final String sectionName) {
        final String[] keys = splitKeys(sectionName);
        if (keys.length == 2) {
            removeSection(keys[0], keys[1]);
        } else {
            rootSectionsMap.remove(sectionName);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#removeSection(java.lang.String,
     * java.lang.String)
     */
    @Override
    public void removeSection(final String sectionName, final String subSection) {
        final RootSection rootSection = rootSectionsMap.get(sectionName);
        rootSection.removeSection(subSection);
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#renameSection(java.lang.String,
     * java.lang.String)
     */
    @Override
    public void renameSection(final String oldName, final String newName) {
        final String[] keys = splitKeys(oldName);
        if (keys.length > 1) {
            renameSection(keys[0], keys[1], newName);
        } else {
            final RootSection rootSection = rootSectionsMap.remove(oldName);
            if (rootSection != null) {
                rootSection.setName(newName);
                LOG.info("Renaming section '" + oldName + "' to '" + newName
                        + "'");
                rootSectionsMap.put(newName, rootSection);
            }
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#renameSection(java.lang.String,
     * java.lang.String, java.lang.String)
     */
    @Override
    public void renameSection(final String sectionName, final String oldName,
                              final String newName) {
        final String[] names = splitKeys(newName);
        final RootSection rootSection = rootSectionsMap.get(sectionName);
        if (rootSection != null && rootSection.getSection(oldName) != null) {
            LOG.info("Renaming sub-section '" + sectionName + "." + oldName
                    + "' to '" + newName + "'");
            final Section section = rootSection.removeSection(oldName);
            if (names.length == 1) {
                section.setName(names[0]);
            } else {
                rootSectionsMap.remove(rootSection.getName());
                rootSection.setName(names[0]);
                section.setName(names[1]);
                rootSectionsMap.put(rootSection.getName(), rootSection);
            }
            rootSection.setSection(section);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#setValue(java.lang.String,
     * java.lang.String)
     */
    @Override
    public void setValue(final String composedKey, final String value) {
        final String[] keys = splitKeys(composedKey);
        if (keys.length < 2) {
            throw new IllegalArgumentException("Invalid variable key : "
                    + keys.length);
        }
        if (keys.length > 2) {
            setValue(keys[0], keys[1], keys[2], value);
        } else {
            setValue(keys[0], keys[1], value);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#setValue(java.lang.String,
     * java.lang.String, java.lang.String)
     */
    @Override
    public void setValue(final String sectionName, final String key,
                         final String value) {
        if (value == null) {
            throw new NullPointerException("Null values are not allowed");
        }
        final RootSection rootSection = getOrCreateSection(sectionName);
        if (key.length() > 0) {
            rootSection.setVariable(key, value);
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#setValue(java.lang.String,
     * java.lang.String, java.lang.String, java.lang.String)
     */
    @Override
    public void setValue(final String sectionName, final String subSectionName,
                         final String key, final String value) {
        if (value == null) {
            throw new NullPointerException("Null values are not allowed");
        }
        final RootSection rootSection = getOrCreateSection(sectionName);
        final Section subSection = rootSection
                .getOrCreateSection(subSectionName);
        subSection.setVariable(key, value);
    }

    private RootSection getOrCreateSection(final String sectionName) {
        RootSection rootSection = rootSectionsMap.get(sectionName);
        if (rootSection == null) {
            rootSection = new RootSection(sectionName);
            rootSectionsMap.put(sectionName, rootSection);
        }
        return rootSection;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#load(java.lang.String)
     */
    @Override
    public void load(final String fileName) throws IOException {
        final Configuration config = FileHandler.loadConfiguration(fileName);
        this.addVariables(config.getVariables());
    }

    /* (non-Javadoc)
     * @see org.timo.gitconfig.Configuration#addVariables(java.util.Map)
     */
    @Override
    public void addVariables(final Map<String, String> variables) {
        for (final Entry<String, String> entry : variables.entrySet()) {
            this.setValue(entry.getKey(), entry.getValue());
        }
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#load(java.io.InputStream)
     */
    @Override
    public void load(final InputStream inputStream) throws IOException {
        final Configuration config = FileHandler.loadConfiguration(inputStream);
        this.addVariables(config.getVariables());
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#save(java.lang.String)
     */
    @Override
    public void save(final String fileName) throws IOException {
        FileHandler.save(fileName, this);
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#save(java.io.OutputStream)
     */
    @Override
    public void save(final OutputStream outputStream) throws IOException {
        FileHandler.save(outputStream, this);
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getVariables()
     */
    @Override
    public Map<String, String> getVariables() {
        final Map<String, String> variables = new HashMap<String, String>();
        for (final RootSection rootSection : rootSectionsMap.values()) {
            variables.putAll(rootSection.getAllVariables());
        }
        return variables;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#getVariables(java.lang.String)
     */
    public Map<String, String> getVariables(final String composedKey) {
        final String[] keys = splitKeys(composedKey);
        final Map<String, String> variables = new HashMap<String, String>();
        final RootSection rootSection = rootSectionsMap.get(keys[0]);

        if (rootSection != null) {
            if (keys.length == 1) {
                variables.putAll(rootSection.getAllVariables());
            } else {
                final Section subSection = rootSection.getSection(keys[1]);
                if (subSection != null) {
                    variables.putAll(subSection.getVariables());
                }
            }
        }
        return variables;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#clear()
     */
    @Override
    public void clear() {
        this.rootSectionsMap.clear();
    }

    /*
     * (non-Javadoc)
     *
     * @see java.lang.Iterable#iterator()
     */
    @Override
    public Iterator<Entry<String, String>> iterator() {
        return new ConfigurationIterator(this);
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#containsVariable(java.lang.String)
     */
    @Override
    public boolean containsVariable(final String composedKey) {
        final String[] keys = splitKeys(composedKey);
        boolean exists = false;
        if (keys.length > 0 && keys.length < 4) {
            if (keys.length == 2) {
                exists = !getValue(keys[0], keys[1]).isEmpty();
            } else {
                exists = !getValue(keys[0], keys[1], keys[2]).isEmpty();
            }
        }
        return exists;
    }

    /*
     * (non-Javadoc)
     *
     * @see org.timo.gitconfig.Configuration#isEmpty()
     */
    @Override
    public boolean isEmpty() {
        boolean isEmpty = rootSectionsMap.isEmpty();
        if (!isEmpty) {
            for (final RootSection rootSection : rootSectionsMap.values()) {
                if (rootSection.isAllEmpty()) {
                    isEmpty = true;
                    break;
                }
            }
        }
        return isEmpty;
    }


}

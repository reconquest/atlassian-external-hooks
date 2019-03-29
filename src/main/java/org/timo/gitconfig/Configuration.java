package org.timo.gitconfig;

import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.Collection;
import java.util.Map;
import java.util.Properties;
import java.util.Set;
import java.util.Map.Entry;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * {@link Properties}-like component allowing a configuration structure:
 *
 * <pre>
 * - Section [0..*]
 * 	- Variable [0..*]
 * 	- Sub-Section [0..*]
 *
 *  e.g.
 * [project 'config']
 * 	 	owner = Hugo Ponce
 * 		path = /opt/projects
 *
 * [merge 'externalTool']
 * 		path = /usr/bin
 * 		command = merge
 *
 * [user]
 * 		email = timo.slack@gmail.com
 * 		name = Timoteo Ponce
 * </pre>
 *
 * As a feature, this {@link Configuration} does not expose implementation
 * details or internal components, giving a simple and generic public API.
 *
 * @author Timoteo Ponce
 *
 */
public interface Configuration extends Iterable<Entry<String, String>> {

    /**
     * Retrieves a given variable value from configuration related to given
     * variable path.
     *
     * @param composedKey
     *            variable path. e.g. core.editor
     * @return value of variable or an empty string if not present
     */
    String getValue(String composedKey);

    Integer getInt(String composedKey);

    Long getLong(String composedKey);

    Double getDouble(String composedKey);

    Boolean getBoolean(String composedKey);


    /**
     * Retrieves a given variable value from configuration related to given
     * variable section and key.
     *
     * @param sectionName
     *            section path. e.g. core.general
     * @param key
     *            variable key
     * @return value of variable or an empty string if not present
     */
    String getValue(String sectionName, String key);

    /**
     * Retrieves a given variable value from configuration related to given
     * section, subSection and key.
     *
     * @param sectionName
     *            variable section
     * @param subSectionName
     *            variable subSection
     * @param key
     *            variable key
     * @return value of variable or an empty string if not present
     */
    String getValue(String sectionName, String subSectionName, String key);

    /**
     * Sets the value of a variable to given variable path. Null values are not
     * allowed and if they are passed a {@link NullPointerException} will be
     * raised.
     *
     * @param composedKey
     *            variable path. e.g. core.user.name
     * @param value
     *            variable value
     *
     */
    void setValue(String composedKey, String value);

    /**
     * @param sectionName
     * @param key
     * @param value
     */
    void setValue(String sectionName, String key, String value);

    /**
     * @param sectionName
     * @param subSectionName
     * @param key
     * @param value
     */
    void setValue(String sectionName, String subSectionName, String key,
                  String value);

    Set<String> getKeySet();

    /**
     * Returns all variable values held in this configuration. Duplicated values
     * are allowed.
     *
     * @return All configuration variable values.
     */
    Collection<String> getValues();

    /**
     * Returns all variable values held in given path. e.g.
     *
     * <pre>
     * - "core" -> all values held in 'core' section.
     * - "core.editor " -> all values held in 'core.editor' section.
     * </pre>
     *
     * @param composedKey
     * @return
     */
    Collection<String> getValues(String composedKey);

    /**
     * Returns a {@link Map} containing all variables in a format:
     *
     * <pre>
     * - [variableKey=variableValue] => e.g. "core.path.format = non-iso "
     * </pre>
     *
     * @return
     */
    Map<String, String> getVariables();

    /**
     * Returns a {@link Map} containing all variables related to a given path in
     * a format:
     *
     * <pre>
     * - [variableKey=variableValue] => e.g. "core.path.format = non-iso "
     * </pre>
     *
     * @return
     */
    Map<String, String> getVariables(String composedKey);

    /**
     * Adds all given variables to configuration, overriding any existing
     * variable.
     *
     * @param variables
     */
    void addVariables(Map<String, String> variables);

    /**
     * Removes a complete configuration section that matches with given
     * sectionName or subSection path. If section or subSection do not exist,
     * nothing happens.
     *
     * @param sectionName
     *            name of the section or subSection path to remove
     */
    void removeSection(String sectionName);

    /**
     * Removes a complete configuration sub-section contained in given section.
     * If the section or the sub-section don't exist, nothing happens.
     *
     * @param sectionName
     * @param subSectionName
     */
    void removeSection(String sectionName, String subSectionName);

    /**
     * Removes given variable from configuration, if the variable or the path to
     * the variable are missing, nothing happens. e.g.
     *
     * <pre>
     * - remove("core.path.format") : removes "core.path.format" variable
     * </pre>
     *
     * @param composedKey
     *            path of the variable to remove
     */
    void remove(String composedKey);

    /**
     * Removes given variable from configuration, if the variable or the path to
     * the variable are missing, nothing happens. e.g.
     *
     * <pre>
     * - remove("core","user") : removes "core.user" variable
     * - remove("core.path","format") : removes "core.path.format" variable
     * </pre>
     *
     * @param sectionName
     *            parent section
     * @param key
     *            variable to remove
     */
    void remove(String sectionName, String key);

    /**
     * Removes given variable from configuration, if the variable or the path to
     * the variable are missing, nothing happens. e.g.
     *
     * <pre>
     * - remove("core","path","format") : removes "core.path.format" variable
     * </pre>
     *
     * @param sectionName
     *            parent section
     * @param subSectionName
     *            variable sub-section
     * @param key
     *            variable to remove
     */
    void remove(String sectionName, String subSectionName, String key);

    /**
     * Renames a given section or sub-section matching with oldName parameter.
     * If section or sub-section does not exists, nothing happens. e.g.
     *
     * <pre>
     * - renameSection("core","main") : renames 'core' sub-section to 'main'
     * - renameSection("general","principal") : renames 'general' sub-section to 'principal'
     * - renameSection("core.path","url") : renames 'core.path' sub-section to 'core.url'
     * - renameSection("application.config","resources"): renames 'application.config' sub-section to 'application.resources'
     * </pre>
     *
     * @param oldName
     *            name of section or path of sub-section
     * @param newName
     *            new name of section or sub-section
     */
    void renameSection(String oldName, String newName);

    /**
     * Renames a sub-section matching with oldName parameter. If section does
     * not exists, nothing happens. e.g.
     *
     * <pre>
     * - renameSection("core","path","url") : renames 'core.path' sub-section to 'core.url'
     * - renameSection("application","config","resources"): renames 'application.config' sub-section to 'application.resources'
     * </pre>
     *
     * @param sectionName
     *            parent section
     * @param oldName
     *            current name of sub-section
     * @param newName
     *            new name for sub-section
     */
    void renameSection(String sectionName, String oldName, String newName);

    /**
     * Returns configuration string content formatted as it will be stored in
     * configuration file.
     *
     * @return text-formatted configuration content
     */
    String getTextContent();

    /**
     * Writes configuration values in a given file, this option will override
     * any previous configuration.
     *
     * @param fileName
     *            target file for configuration variables
     */
    void save(final String fileName) throws IOException;

    void save(final OutputStream outputStream) throws IOException;

    /**
     * Loads all variables from given file, appending them to current
     * configuration variables and overriding entries if are present in file.
     *
     * @param fileName
     *            configuration source file
     * @throws FileNotFoundException
     *             if file can't be read
     */
    void load(final String fileName) throws IOException;

    void load(final InputStream inputStream) throws IOException;

    /**
     * Clear all sections and sub-sections in configuration.
     */
    void clear();

    boolean isEmpty();

    boolean containsVariable(String composedKey);

}

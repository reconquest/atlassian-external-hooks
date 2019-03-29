package org.timo.gitconfig;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.Reader;
import java.util.StringTokenizer;
import java.util.logging.Logger;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Copyright (C) 2010 Timoteo Ponce
 *
 * @author Timoteo Ponce
 *
 */
public class FileHandler {

    private static final Logger LOG = Logger.getLogger(FileHandler.class
            .getName());

    private static final String COMMENT_CHARS = "#;";

    private static final Pattern SECTION_PATTERN = Pattern
            .compile("(\\w)*[^\\s'\"\\[\\]]");

    public static Configuration loadConfiguration(final String fileName)
            throws IOException {
        final Configuration config = new GitConfiguration();
        Reader reader = null;
        BufferedReader bufferedReader = null;
        try {
            reader = new FileReader(fileName);
            bufferedReader = new BufferedReader(reader);
            load(bufferedReader, config);
        } finally {
            if (bufferedReader != null) {
                bufferedReader.close();
                reader.close();
            }
        }
        return config;
    }

    private static void load(final BufferedReader bufferedReader,
                             final Configuration config) throws IOException {
        String line;
        while ((line = bufferedReader.readLine()) != null) {
            line = line.trim();
            if (isNotEmpty(line)) {
                final String sectionPath = readSection(line, config);
                readVariables(bufferedReader, sectionPath, config);
            }
        }
    }

    private static boolean isNotEmpty(final String line) {
        return line.length() > 0 && COMMENT_CHARS.indexOf(line.charAt(0)) == -1;
    }

    private static String readSection(final String line,
                                      final Configuration config) {
        LOG.info("Reading section from line : " + line);
        if (line.startsWith("[") && line.endsWith("]")) {
            final Matcher matcher = SECTION_PATTERN.matcher(line);
            matcher.find();// find the first match
            final String sectionName = matcher.group().trim();

            final boolean isSubSection = matcher.find();
            // [ sectionName 'subSection' ]
            String sectionPath = sectionName;
            if (isSubSection) {
                final String subSection = matcher.group().trim();
                sectionPath += "." + subSection;
                LOG.info("Reading subSection: " + sectionName + "->"
                        + subSection);
            } else {
                LOG.info("Reading section: " + sectionName);
            }
            config.setValue(sectionPath, "", "");
            return sectionPath;
        } else {
            throw new IllegalArgumentException(
                    "Unreadable section declaration [ sectionName *'subSectionName'] :"
                            + line);
        }
    }

    /**
     * @param bufferedReader
     * @param section
     * @throws IOException
     */
    private static void readVariables(final BufferedReader bufferedReader,
                                      final String sectionPath, final Configuration config)
            throws IOException {
        final StringBuilder variablesBuffer = new StringBuilder();
        String line;
        while ((line = bufferedReader.readLine()) != null) {
            line = line.trim();
            if (isNotEmpty(line)) {
                if (line.startsWith("[")) {
                    final String localSection = readSection(line, config);
                    readVariables(bufferedReader, localSection, config);
                } else {
                    variablesBuffer.append(line + "\n");
                }
            }
        }
        // variable = value
        final StringTokenizer tokenizer = new StringTokenizer(
                variablesBuffer.toString(), "\n=");

        while (tokenizer.hasMoreTokens()) {
            final String key = sectionPath + "." + tokenizer.nextToken().trim();
            config.setValue(key, tokenizer.nextToken().trim());
        }
    }

    public static Configuration loadConfiguration(final InputStream inputStream)
            throws IOException {
        final Configuration config = new GitConfiguration();
        Reader reader = null;
        BufferedReader bufferedReader = null;
        try {
            reader = new InputStreamReader(inputStream);
            bufferedReader = new BufferedReader(reader);
            load(bufferedReader, config);
        } finally {
            if (bufferedReader != null) {
                bufferedReader.close();
                reader.close();
            }
        }
        return config;
    }

    public static void save(final String fileName, final Configuration config)
            throws IOException {
        FileWriter writer = null;
        try {
            writer = new FileWriter(fileName);
            writer.append(config.getTextContent());
        } finally {
            if (writer != null) {
                writer.close();
            }
        }
    }

    public static void save(final OutputStream outputStream,
                            final Configuration config) throws IOException {
        OutputStreamWriter writer = null;
        try {
            writer = new OutputStreamWriter(outputStream);
            writer.append(config.getTextContent());
        } finally {
            if (writer != null) {
                writer.close();
            }
        }
    }

}

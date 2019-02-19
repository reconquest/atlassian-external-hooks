#!/bin/bash

grep -h -R -Po '^import \K.*' ./src \
    | sed -r -e 's/static //' \
    -e 's/^java.(io|lang|nio|math|net|text|util).*/Java SE SDK/' \
    -e 's/^javax.inject.*/Javax Inject/' \
    -e 's/^javax.annotation.*/Javax Annotation/' \
    -e 's/^com.atlassian.*/Atlassian Plugin SDK/' \
    -e 's/^org.apache.commons.*/Apache Commons Java/' \
    -e 's/^net.java.ao.*/Java ActiveObjects/' \
    | grep -vP '^com.ngs.stash.externalhooks' \
    | sort -n | uniq

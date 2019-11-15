package com.ngs.stash.externalhooks.rest;

import javax.xml.bind.annotation.XmlElement;
import javax.xml.bind.annotation.XmlRootElement;

@XmlRootElement
public class Settings {
  @XmlElement public String todo1;
  @XmlElement public String todo2;
}

package com.ngs.stash.externalhooks.license;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.InvalidKeyException;
import java.security.KeyFactory;
import java.security.NoSuchAlgorithmException;
import java.security.Signature;
import java.security.SignatureException;
import java.security.interfaces.DSAPublicKey;
import java.security.spec.X509EncodedKeySpec;

import com.atlassian.bitbucket.cluster.ClusterService;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.upm.api.license.PluginLicenseManager;
import com.atlassian.upm.api.license.entity.PluginLicense;
import com.atlassian.upm.api.util.Option;

import org.apache.commons.codec.binary.Base64;
import org.apache.commons.io.IOUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class LicenseValidator {
  private static Logger log = LoggerFactory.getLogger(LicenseValidator.class.getSimpleName());
  private PluginLicenseManager pluginLicenseManager;
  private StorageService storageService;
  private ClusterService clusterService;
  private String pluginKey;
  private boolean licenseSignatureVerified = false;

  private static final String PUBLIC_KEY = ""
      + "MIIDRjCCAjkGByqGSM44BAEwggIsAoIBAQCXNVVR/55M+fXGU6GmpW6RmSIIxi+V\n"
      + "65651FSMztGZYUAcLKpVBopXLB+SZamNDsXbMVklog/umUa5mKRUQjZD2dXrgLrt\n"
      + "jbs9EIpWF9jvDkcjlTSgdFSsrN+w1zJt+ImG6zLVeWRF36ozlBB/9w1CCszQe7Vt\n"
      + "4x/JgUuGHu0xJGoJHNhFsj1KncG06fqz/vQml08KCQfIbBbgt7ahfmGPqL9we6Ka\n"
      + "bpe2vflH/j9UOH+6GWXBbBtLWay0JoJ58fc6+mdm0cv0EL/Vki7qjK3d7V282TqD\n"
      + "fR+PCXfIJN3EL6u/0ZJkQIkEcKw1Zq5mqlU7DZYZdtCb6TQcBJy9kct7AiEArjwA\n"
      + "vCPwp4IaI5XiOSb40v3tu+lyMLvHzQZYyub0sbUCggEAIqbSiYZP1sIazTbl5AYb\n"
      + "znseWYp1zCc9rJRVhbT8Kj8ap6yXlANRPHAKcIrl+NzkyIit0P9+f+MoKeFPV7us\n"
      + "LyN3tgHOudOG1Ha3KBrYP9rL0EoSL8lpl4W03f1csKXYa2t9W1UwVZyuYZA/x7vU\n"
      + "dZC2CrWehLfAFZXJjRSrF6H5vGQ8IxcRBjVWloUs2w/0SB4Zd+/EwC7BRwQACfyY\n"
      + "LEDEDJkE6d5DA2ww6htrccXcUOGCCUwl0DEc1Xn8ek9qLVwWGnY7D1teCqdtn5Ao\n"
      + "rrBdpptxf2D9fD3j8nc3/XKxDd9OdO+XQqm5RZPXnL7KRu5xmyzixXUdL2Om/5ey\n"
      + "owOCAQUAAoIBAECfmKkgneq7Lm4o/YkTAHBtx8DDcDHjnJNMwnsahVg2+b2r2CCF\n"
      + "DM6r8L6+xXkXhDCpA4Y+V8my0/G7nxthBU8nJd9Z2cdq8qbITnXYnaSGx6OSk5T0\n"
      + "V6eP8ck2CtePSoADhenvIoeFgC+4biFQsCLF/NWckudPr5/Nx4773c3b7oe2mC3A\n"
      + "QE35aoGYg6d4kIOtIvlxopIVyXPhEqGcvoP5RNWsn/PwGaq8rgiFa92RjX4Xd9Xc\n"
      + "ZJXFZ3lnW1fqyDe7KIaTZw3sGrBZ/4IhpvnvGHVxBYz1rBahL3KpYt1b6E6N65t4\n"
      + "2nOMzOgp6Glhr++St20VeNwfwV2PTN5Je80=";

  public LicenseValidator(
      String pluginKey,
      PluginLicenseManager pluginLicenseManager,
      StorageService storageService,
      ClusterService clusterService) {
    this.pluginKey = pluginKey;
    this.pluginLicenseManager = pluginLicenseManager;
    this.storageService = storageService;
    this.clusterService = clusterService;

    this.initLicense();
  }

  private void initLicense() {
    licenseSignatureVerified = verifyLicense();
    if (licenseSignatureVerified) {
      log.warn("license signature verified");
    }
  }

  public boolean verifyLicense() {
    Signature dsa;
    try {
      dsa = Signature.getInstance("SHA256withDSA");
    } catch (NoSuchAlgorithmException e) {
      log.error("Unable to init Signature verifier", e);
      return false;
    }

    String encodedLicense = readPluginLicense();
    if (encodedLicense == null || encodedLicense.isEmpty()) {
      return false;
    }

    DSAPublicKey key;
    try {
      key = getPublicKey();
    } catch (IOException | GeneralSecurityException e) {
      log.error("Unable to decode public dsa key", e);
      return false;
    }

    try {
      dsa.initVerify(key);
    } catch (InvalidKeyException e) {
      log.error("unable to init dsa verifier", e);
      return false;
    }

    License license = new License(encodedLicense);

    boolean verified;
    try {
      dsa.update(license.getData());
      verified = dsa.verify(license.getSignature());
    } catch (SignatureException e) {
      log.error("invalid license", e);
      return false;
    }

    return verified;
  }

  public String readPluginLicense() {
    File homeDir = getHomeDir();
    File license = new File(homeDir.getAbsolutePath(), pluginKey + ".license");
    if (!license.exists()) {
      return null;
    }

    try {
      String result = IOUtils.toString(new FileInputStream(license), StandardCharsets.UTF_8);
      return result;
    } catch (IOException e) {
      log.warn("unable to read license file", e);
      return null;
    }
  }

  private DSAPublicKey getPublicKey() throws IOException, GeneralSecurityException {
    byte[] encoded = Base64.decodeBase64(PUBLIC_KEY);
    KeyFactory kf = KeyFactory.getInstance("DSA");
    DSAPublicKey pubKey = (DSAPublicKey) kf.generatePublic(new X509EncodedKeySpec(encoded));
    return pubKey;
  }

  private File getHomeDir() {
    if (this.clusterService.isAvailable()) {
      return this.storageService.getSharedHomeDir().toFile();
    } else {
      return this.storageService.getHomeDir().toFile();
    }
  }

  public boolean isDefined() {
    if (licenseSignatureVerified) {
      return true;
    }
    Option<PluginLicense> licenseOption = pluginLicenseManager.getLicense();
    return licenseOption.isDefined();
  }

  public boolean isValid() {
    if (licenseSignatureVerified) {
      return true;
    }

    Option<PluginLicense> licenseOption = pluginLicenseManager.getLicense();
    if (!licenseOption.isDefined()) {
      return false;
    }

    PluginLicense pluginLicense = licenseOption.get();
    return pluginLicense.isValid();
  }

  private class License {
    private byte[] signature;
    private byte[] data;

    public License(String encoded) {
      String[] chunks = encoded.split(" ", 2);
      if (chunks.length != 2) {
        log.warn("invalid license format");
        return;
      }

      this.signature = Base64.decodeBase64(chunks[0]);
      this.data = Base64.decodeBase64(chunks[1]);
    }

    public byte[] getSignature() {
      return this.signature;
    }

    public byte[] getData() {
      return this.data;
    }
  }
}

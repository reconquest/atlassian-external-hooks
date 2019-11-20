package com.ngs.stash.externalhooks.servlet;

import java.io.IOException;
import java.net.URI;
import java.util.HashMap;
import java.util.Map;

import javax.inject.Inject;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import com.atlassian.plugin.spring.scanner.annotation.component.Scanned;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.auth.LoginUriProvider;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.sal.api.user.UserKey;
import com.atlassian.sal.api.user.UserManager;
import com.atlassian.templaterenderer.TemplateRenderer;
import com.ngs.stash.externalhooks.ExternalHooksSettings;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@Scanned
public class Settings extends HttpServlet {
  private static final Logger log = LoggerFactory.getLogger(Settings.class);

  private TemplateRenderer templateRenderer;
  private PluginSettingsFactory pluginSettingsFactory;
  private UserManager userManager;
  private LoginUriProvider loginUriProvider;

  @Inject
  public Settings(
      @ComponentImport UserManager userManager,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport LoginUriProvider loginUriProvider,
      @ComponentImport TemplateRenderer templateRenderer) {
    this.userManager = userManager;
    this.templateRenderer = templateRenderer;
    this.loginUriProvider = loginUriProvider;
    this.pluginSettingsFactory = pluginSettingsFactory;
  }

  @Override
  protected void doGet(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    UserKey user = userManager.getRemoteUserKey(request);
    if (user == null || !userManager.isSystemAdmin(user)) {
      redirectToLogin(request, response);
      return;
    }

    renderPanel(response, false);
  }

  @Override
  protected void doPost(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    UserKey user = userManager.getRemoteUserKey(request);
    if (user == null || !userManager.isSystemAdmin(user)) {
      redirectToLogin(request, response);
      return;
    }

    renderPanel(response, true);
  }

  protected void renderPanel(HttpServletResponse response, Boolean success) throws IOException {
    response.setContentType("text/html;charset=utf-8");

    Map<String, Object> context = new HashMap<String, Object>();

    if (success) {
      context.put("success", "true");
    }

    templateRenderer.render("ui/settings.vm", context, response.getWriter());
  }

  private void redirectToLogin(HttpServletRequest request, HttpServletResponse response)
      throws IOException {
    response.sendRedirect(loginUriProvider.getLoginUri(getUri(request)).toASCIIString());
  }

  private URI getUri(HttpServletRequest request) {
    StringBuffer builder = request.getRequestURL();
    if (request.getQueryString() != null) {
      builder.append("?");
      builder.append(request.getQueryString());
    }
    return URI.create(builder.toString());
  }
}

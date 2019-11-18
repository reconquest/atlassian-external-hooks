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
import com.atlassian.sal.api.pluginsettings.PluginSettings;
import com.atlassian.sal.api.pluginsettings.PluginSettingsFactory;
import com.atlassian.sal.api.user.UserKey;
import com.atlassian.sal.api.user.UserManager;
import com.atlassian.templaterenderer.TemplateRenderer;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@Scanned
public class Settings extends HttpServlet {
  private static final Logger log = LoggerFactory.getLogger(Settings.class);

  private TemplateRenderer templateRenderer;
  private PluginSettingsFactory pluginSettingsFactory;
  private UserManager userManager;
  private LoginUriProvider loginUriProvider;

  private PluginSettings pluginSettings;

  @Inject
  public Settings(
      @ComponentImport UserManager userManager,
      @ComponentImport PluginSettingsFactory pluginSettingsFactory,
      @ComponentImport LoginUriProvider loginUriProvider,
      @ComponentImport TemplateRenderer templateRenderer) {
    this.userManager = userManager;
    this.templateRenderer = templateRenderer;
    this.loginUriProvider = loginUriProvider;
    this.pluginSettings = pluginSettingsFactory.createSettingsForKey(ExternalHookScript.PLUGIN_KEY);
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

    com.ngs.stash.externalhooks.rest.Settings settings =
        new com.ngs.stash.externalhooks.rest.Settings();

    // Map<String, Boolean> triggers = new HashMap<String, Boolean>();

    // triggers.put("pre_receive_branch_create", true);
    // triggers.put("pre_receive_branch_delete", true);
    // triggers.put("pre_receive_tag_create", true);
    // triggers.put("pre_receive_tag_delete", true);
    // triggers.put("pre_receive_file_edit", true);
    // triggers.put("pre_receive_pr_merge_check", false);
    // triggers.put("pre_receive_internal_merge", false);
    // triggers.put("pre_receive_repo_push", true);

    // triggers.put("post_receive_branch_create", true);
    // triggers.put("post_receive_branch_delete", true);
    // triggers.put("post_receive_tag_create", true);
    // triggers.put("post_receive_tag_delete", true);
    // triggers.put("post_receive_file_edit", true);
    // triggers.put("post_receive_pr_merge_check", false);
    // triggers.put("post_receive_internal_merge", false);
    // triggers.put("post_receive_repo_push", true);

    context.put("settings", settings);

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

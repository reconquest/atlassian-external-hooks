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
import com.atlassian.sal.api.user.UserKey;
import com.atlassian.sal.api.user.UserManager;
import com.atlassian.templaterenderer.TemplateRenderer;

@Scanned
@SuppressWarnings("serial") // suppress because the http servlet is not going to be serialized
public class Global extends HttpServlet {
  private TemplateRenderer templateRenderer;
  private UserManager userManager;
  private LoginUriProvider loginUriProvider;

  @Inject
  public Global(
      @ComponentImport UserManager userManager,
      @ComponentImport LoginUriProvider loginUriProvider,
      @ComponentImport TemplateRenderer templateRenderer) {
    this.userManager = userManager;
    this.templateRenderer = templateRenderer;
    this.loginUriProvider = loginUriProvider;
  }

  @Override
  protected void doGet(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    UserKey user = userManager.getRemoteUserKey(request);
    if (user == null || !userManager.isSystemAdmin(user)) {
      redirectToLogin(request, response);
      return;
    }

    response.setContentType("text/html;charset=utf-8");

    Map<String, Object> context = new HashMap<String, Object>();

    templateRenderer.render("ui/global.vm", context, response.getWriter());
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

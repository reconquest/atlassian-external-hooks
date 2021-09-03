package com.ngs.stash.externalhooks.servlet;

import java.io.IOException;
import java.io.OutputStream;
import java.net.URI;
import java.util.HashMap;
import java.util.Map;

import javax.inject.Inject;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.bitbucket.util.PageRequest;
import com.atlassian.bitbucket.util.PageRequestImpl;
import com.atlassian.plugin.spring.scanner.annotation.component.Scanned;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.auth.LoginUriProvider;
import com.atlassian.sal.api.user.UserKey;
import com.atlassian.sal.api.user.UserManager;
import com.atlassian.templaterenderer.TemplateRenderer;
import com.ngs.stash.externalhooks.Const;

import org.json.simple.JSONObject;
import org.slf4j.LoggerFactory;

import ch.qos.logback.classic.Level;

@Scanned
@SuppressWarnings("serial") // suppress because the http servlet is not going to be serialized
public class Diagnostics extends HttpServlet {
  private UserManager userManager;
  private TemplateRenderer templateRenderer;
  private LoginUriProvider loginUriProvider;
  private HookScriptService hookScriptService;

  @Inject
  public Diagnostics(
      @ComponentImport HookScriptService hookScriptService,
      @ComponentImport LoginUriProvider loginUriProvider,
      @ComponentImport UserManager userManager,
      @ComponentImport TemplateRenderer templateRenderer) {
    this.hookScriptService = hookScriptService;
    this.loginUriProvider = loginUriProvider;
    this.templateRenderer = templateRenderer;
    this.userManager = userManager;
  }

  @Override
  protected void doGet(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    UserKey user = userManager.getRemoteUserKey(request);
    if (user == null || !userManager.isSystemAdmin(user)) {
      redirectToLogin(request, response);
      return;
    }

    if (request.getParameter("dump") != null) {
      this.dumpHookScripts(request, response);
      return;
    }

    render(response, null);
  }

  private void render(HttpServletResponse response, Map<String, Object> context)
      throws IOException {
    response.setContentType("text/html;charset=utf-8");

    if (context == null) {
      context = new HashMap<String, Object>();
    }

    context.put("hook_scripts_total", this.getTotalHookScripts());

    ch.qos.logback.classic.Logger logger =
        (ch.qos.logback.classic.Logger) LoggerFactory.getLogger(Const.PACKAGE);

    context.put(
        "log_level",
        logger.getLevel() == null ? Level.INFO.toString() : logger.getLevel().toString());

    templateRenderer.render("ui/diagnostics.vm", context, response.getWriter());
  }

  private void dumpHookScripts(HttpServletRequest request, HttpServletResponse response)
      throws IOException {
    response.setContentType("application/octet-stream");
    response.setHeader(
        "Content-Disposition",
        "attachment; filename=\"external_hooks_scripts_"
            + String.valueOf(System.currentTimeMillis())
            + ".json\"");

    OutputStream output = response.getOutputStream();

    output.write("[".getBytes());

    PageRequest page = new PageRequestImpl(0, 100);
    int total = 0;
    while (true) {
      Page<HookScript> scripts = hookScriptService.findByPluginKey(Const.PLUGIN_KEY, page);
      if (scripts.getSize() == 0) {
        break;
      }

      for (HookScript script : scripts.getValues()) {
        if (total > 0) {
          output.write(",".getBytes());
        }

        total++;

        HashMap<String, Object> object = new HashMap<>();

        object.put("id", script.getId());
        object.put("name", script.getName());
        object.put("version", script.getVersion());
        object.put("size", script.getSize());
        object.put("created_date", script.getCreatedDate().getTime() / 1000);
        object.put("updated_date", script.getUpdatedDate().getTime() / 1000);
        object.put("plugin_key", script.getPluginKey());

        output.write(new JSONObject(object).toJSONString().getBytes());
      }

      page = scripts.getNextPageRequest();
      if (page == null) {
        break;
      }
    }

    output.write("]".getBytes());
  }

  @Override
  protected void doPost(HttpServletRequest request, HttpServletResponse response)
      throws ServletException, IOException {
    UserKey user = userManager.getRemoteUserKey(request);
    if (user == null || !userManager.isSystemAdmin(user)) {
      redirectToLogin(request, response);
      return;
    }

    Map<String, Object> context = new HashMap<String, Object>();
    if (request.getParameter("action") != null) {
      if (request.getParameter("action").equals("remove_by_plugin_key")) {
        hookScriptService.deleteByPluginKey(Const.PLUGIN_KEY);
        context.put("success", Boolean.TRUE);
      }

      if (request.getParameter("action").equals("change_log_level")) {
        ch.qos.logback.classic.Logger levelSet =
            (ch.qos.logback.classic.Logger) LoggerFactory.getLogger(Const.PACKAGE);

        String targetLevel = request.getParameter("log_level");
        if (targetLevel != null) {
          levelSet.setLevel(Level.toLevel(targetLevel, Level.INFO));
        }
      }
    }

    render(response, context);
  }

  private int getTotalHookScripts() {
    PageRequest page = new PageRequestImpl(0, 100);
    int total = 0;
    while (true) {
      Page<HookScript> scripts = hookScriptService.findByPluginKey(Const.PLUGIN_KEY, page);
      if (scripts.getSize() == 0) {
        break;
      }

      total += scripts.getSize();

      page = scripts.getNextPageRequest();
      if (page == null) {
        break;
      }
    }

    return total;
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

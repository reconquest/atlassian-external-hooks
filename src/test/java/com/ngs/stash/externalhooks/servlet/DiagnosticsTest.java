package com.ngs.stash.externalhooks.servlet;

import static org.junit.Assert.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.URI;
import java.util.Map;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.hook.script.HookScriptService;
import com.atlassian.bitbucket.log.LoggingService;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.sal.api.auth.LoginUriProvider;
import com.atlassian.sal.api.user.UserKey;
import com.atlassian.sal.api.user.UserManager;
import com.atlassian.templaterenderer.TemplateRenderer;
import com.ngs.stash.externalhooks.Const;

import org.junit.Before;
import org.junit.Test;
import org.mockito.ArgumentCaptor;

public class DiagnosticsTest {
  private final HookScriptService scripts = mock(HookScriptService.class);
  private final LoggingService logging = mock(LoggingService.class);
  private final LoginUriProvider login = mock(LoginUriProvider.class);
  private final UserManager users = mock(UserManager.class);
  private final TemplateRenderer renderer = mock(TemplateRenderer.class);
  private final HttpServletRequest request = mock(HttpServletRequest.class);
  private final HttpServletResponse response = mock(HttpServletResponse.class);
  private final Diagnostics diagnostics = new Diagnostics(scripts, logging, login, users, renderer);

  @Before
  @SuppressWarnings("unchecked")
  public void setUp() throws Exception {
    UserKey admin = new UserKey("admin");
    when(users.getRemoteUserKey(request)).thenReturn(admin);
    when(users.isSystemAdmin(admin)).thenReturn(true);
    when(response.getWriter()).thenReturn(new PrintWriter(new StringWriter()));
    Page<HookScript> page = mock(Page.class);
    when(scripts.findByPluginKey(eq(Const.PLUGIN_KEY), any())).thenReturn(page);
  }

  @Test
  @SuppressWarnings({"unchecked", "rawtypes"})
  public void rendersHostLogLevel() throws Exception {
    when(logging.getLevel(Const.PACKAGE)).thenReturn("DEBUG");

    diagnostics.doGet(request, response);

    ArgumentCaptor<Map<String, Object>> context = ArgumentCaptor.forClass((Class) Map.class);
    verify(renderer).render(eq("ui/diagnostics.vm"), context.capture(), any());
    assertEquals("DEBUG", context.getValue().get("log_level"));
  }

  @Test
  public void updatesHostLogLevel() throws Exception {
    when(request.getParameter("action")).thenReturn("change_log_level");
    when(request.getParameter("log_level")).thenReturn("debug");

    diagnostics.doPost(request, response);

    verify(logging).setLevel(Const.PACKAGE, "DEBUG");
  }

  @Test
  public void invalidLogLevelUsesInfo() throws Exception {
    when(request.getParameter("action")).thenReturn("change_log_level");
    when(request.getParameter("log_level")).thenReturn("invalid");

    diagnostics.doPost(request, response);

    verify(logging).setLevel(Const.PACKAGE, "INFO");
  }

  @Test
  public void missingLogLevelDoesNotChangeLogging() throws Exception {
    when(request.getParameter("action")).thenReturn("change_log_level");

    diagnostics.doPost(request, response);

    verify(logging, never()).setLevel(any(), any());
  }

  @Test
  public void anonymousPostCannotChangeDiagnostics() throws Exception {
    when(users.getRemoteUserKey(request)).thenReturn(null);
    when(request.getRequestURL()).thenReturn(new StringBuffer("http://localhost/diagnostics"));
    when(login.getLoginUri(any())).thenReturn(URI.create("http://localhost/login"));
    when(request.getParameter("action")).thenReturn("remove_by_plugin_key");

    diagnostics.doPost(request, response);

    verify(response).sendRedirect("http://localhost/login");
    verifyNoInteractions(logging, scripts, renderer);
  }
}

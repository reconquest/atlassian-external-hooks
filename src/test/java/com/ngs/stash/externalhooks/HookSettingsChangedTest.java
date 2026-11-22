package com.ngs.stash.externalhooks;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import java.util.List;

import com.atlassian.bitbucket.event.hook.RepositoryHookSettingsChangedEvent;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.project.Project;
import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.event.api.EventPublisher;
import com.ngs.stash.externalhooks.dao.GlobalHookSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;

import org.junit.Before;
import org.junit.Test;

public class HookSettingsChangedTest {
  private static final String HOOK_KEY = Const.PLUGIN_KEY + ":" + Const.PRE_RECEIVE_HOOK_ID;

  private final RepositoryHookService hooks = mock(RepositoryHookService.class);
  private final HookInstaller installer = mock(HookInstaller.class);
  private final GlobalHookSettingsDao settings = mock(GlobalHookSettingsDao.class);
  private final RepositoryHook hook = mock(RepositoryHook.class);
  private final ExternalHookScript script = mock(ExternalHookScript.class);
  private final Project project = mock(Project.class);
  private final Scope projectScope = new ProjectScope(project);
  private final BitbucketEventListener listener = new BitbucketEventListener(
      mock(EventPublisher.class), settings, hooks, installer);

  @Before
  public void setUp() {
    when(installer.getScript(HOOK_KEY)).thenReturn(script);
    when(script.getHookKey()).thenReturn(HOOK_KEY);
    when(settings.find()).thenReturn(List.of());
    when(hooks.getByKey(any(), any())).thenReturn(hook);
    when(hook.isEnabled()).thenReturn(true);
    when(hook.isConfigured()).thenReturn(true);
    when(hook.getScope()).thenReturn(projectScope);
  }

  @Test
  public void refreshesEnabledProjectHook() {
    listener.onHookSettingsChanged(event(projectScope, HOOK_KEY));

    verify(installer).enable(projectScope, script);
  }

  @Test
  public void refreshesEnabledRepositoryHook() {
    Scope scope = repositoryScope();
    when(hook.getScope()).thenReturn(scope);

    listener.onHookSettingsChanged(event(scope, HOOK_KEY));

    verify(installer).enable(scope, script);
  }

  @Test
  public void ignoresDisabledHook() {
    when(hook.isEnabled()).thenReturn(false);

    listener.onHookSettingsChanged(event(projectScope, HOOK_KEY));

    verify(installer, never()).enable(any(Scope.class), any(ExternalHookScript.class));
  }

  @Test
  public void ignoresUnconfiguredHook() {
    when(hook.isConfigured()).thenReturn(false);

    listener.onHookSettingsChanged(event(projectScope, HOOK_KEY));

    verify(installer, never()).enable(any(Scope.class), any(ExternalHookScript.class));
  }

  @Test
  public void preservesInheritedHook() {
    listener.onHookSettingsChanged(event(repositoryScope(), HOOK_KEY));

    verify(installer, never()).enable(any(Scope.class), any(ExternalHookScript.class));
  }

  @Test
  public void ignoresOtherPlugins() {
    listener.onHookSettingsChanged(event(projectScope, "other:hook"));

    verifyNoInteractions(hooks, settings);
    verify(installer, never()).enable(any(Scope.class), any(ExternalHookScript.class));
  }

  private Scope repositoryScope() {
    Repository repository = mock(Repository.class);
    when(repository.getProject()).thenReturn(project);
    return new RepositoryScope(repository);
  }

  private RepositoryHookSettingsChangedEvent event(Scope scope, String key) {
    Settings settings = new SimpleSettingsBuilder().build();
    return new RepositoryHookSettingsChangedEvent(this, scope, key, settings, settings);
  }
}

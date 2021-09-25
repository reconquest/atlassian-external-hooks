package com.ngs.stash.externalhooks;

import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookEnabledEvent;
import com.atlassian.bitbucket.event.repository.RepositoryCreatedEvent;
import com.atlassian.bitbucket.event.repository.RepositoryDeletedEvent;
import com.atlassian.bitbucket.scope.GlobalScope;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.event.api.EventListener;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.dao.GlobalHookSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.ScopeUtil;

public class BitbucketEventListener {
  private GlobalHookSettingsDao globalHookSettingsDao;
  private HooksFactory hooksFactory;
  private HookInstaller hookInstaller;

  public BitbucketEventListener(
      @ComponentImport GlobalHookSettingsDao globalHookSettingsDao,
      @ComponentImport HooksFactory hooksFactory,
      @ComponentImport HookInstaller hookInstaller) {
    this.hookInstaller = hookInstaller;
    this.hooksFactory = hooksFactory;
    this.globalHookSettingsDao = globalHookSettingsDao;
  }

  @EventListener
  public void onHookEnabled(RepositoryHookEnabledEvent event) {
    ExternalHookScript script = hookInstaller.getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    hookInstaller.enable(event.getScope(), script);

    GlobalHooks globalHooks = new GlobalHooks(this.globalHookSettingsDao.find());
    if (globalHooks.isEnabled(script.getHookKey())) {
      hookInstaller.enable(event.getScope(), script, globalHooks);
    }
  }

  @EventListener
  public void onHookDisabled(RepositoryHookDisabledEvent event) {
    ExternalHookScript script = hookInstaller.getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      hookInstaller.disable((RepositoryScope) scope, script);
    } else if (ScopeUtil.isProject(scope)) {
      hookInstaller.disable((ProjectScope) scope, script);
    }
  }

  // This event is triggered when repository hook transfered from 'Enabled' to
  // 'Inherited' state. It means that the repository doesn't have any its own
  // hook, but might have project's hook.
  //
  // Also, triggered when the state changed from 'Disabled' to 'Inherited'
  @EventListener
  public void onHookInherited(RepositoryHookDeletedEvent event) {
    ExternalHookScript script = hookInstaller.getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      hookInstaller.inherit((RepositoryScope) scope, script);
    }
  }

  @EventListener
  public void onRepositoryCreated(RepositoryCreatedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    GlobalHooks globalHooks = new GlobalHooks(this.globalHookSettingsDao.find());
    hookInstaller.getScripts().forEach((hookId, script) -> {
      hookInstaller.inherit(scope, script);

      if (globalHooks.isEnabled(script.getHookKey())) {
        hookInstaller.enable(scope, script, globalHooks);
      }
    });
  }

  @EventListener
  public void onRepositoryDeleted(RepositoryDeletedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    hookInstaller.getScripts().forEach((hookId, script) -> {
      script.uninstall(scope);
      script.uninstall(new GlobalScope(), scope);
    });
  }
}

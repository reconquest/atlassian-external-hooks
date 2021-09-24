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
  private HooksCoordinator hooksCoordinator;

  public BitbucketEventListener(
      @ComponentImport GlobalHookSettingsDao globalHookSettingsDao,
      @ComponentImport HooksFactory hooksFactory,
      @ComponentImport HooksCoordinator hooksCoordinator) {
    this.hooksCoordinator = hooksCoordinator;
    this.hooksFactory = hooksFactory;
    this.globalHookSettingsDao = globalHookSettingsDao;
  }

  @EventListener
  public void onHookEnabled(RepositoryHookEnabledEvent event) {
    ExternalHookScript script = hooksCoordinator.getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    hooksFactory.apply(event.getScope(), new GlobalHooks(this.globalHookSettingsDao.find()));
  }

  @EventListener
  public void onHookDisabled(RepositoryHookDisabledEvent event) {
    ExternalHookScript script = hooksCoordinator.getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      hooksCoordinator.disable((RepositoryScope) scope, script);
    } else if (ScopeUtil.isProject(scope)) {
      hooksCoordinator.disable((ProjectScope) scope, script);
    }
  }

  // This event is triggered when repository hook transfered from 'Enabled' to
  // 'Inherited' state. It means that the repository doesn't have any its own
  // hook, but might have project's hook.
  //
  // Also, triggered when the state changed from 'Disabled' to 'Inherited'
  @EventListener
  public void onHookInherited(RepositoryHookDeletedEvent event) {
    ExternalHookScript script = hooksCoordinator.getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    Scope scope = event.getScope();
    if (ScopeUtil.isRepository(scope)) {
      hooksCoordinator.inherit((RepositoryScope) scope, script);
    }
  }

  @EventListener
  public void onRepositoryCreated(RepositoryCreatedEvent event) {
    hooksFactory.apply(
        new RepositoryScope(event.getRepository()),
        new GlobalHooks(this.globalHookSettingsDao.find()));
  }

  @EventListener
  public void onRepositoryDeleted(RepositoryDeletedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    hooksCoordinator.getScripts().forEach((hookId, script) -> {
      script.uninstall(scope);
      script.uninstall(new GlobalScope(), scope);
    });
  }
}

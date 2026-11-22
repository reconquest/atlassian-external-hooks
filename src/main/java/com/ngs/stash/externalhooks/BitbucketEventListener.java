package com.ngs.stash.externalhooks;

import jakarta.inject.Inject;
import jakarta.inject.Named;

import com.atlassian.bitbucket.event.hook.RepositoryHookDeletedEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookDisabledEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookEnabledEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookEvent;
import com.atlassian.bitbucket.event.hook.RepositoryHookSettingsChangedEvent;
import com.atlassian.bitbucket.event.repository.RepositoryCreatedEvent;
import com.atlassian.bitbucket.event.repository.RepositoryDeletedEvent;
import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.scope.GlobalScope;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.event.api.EventListener;
import com.atlassian.event.api.EventPublisher;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.dao.GlobalHookSettingsDao;
import com.ngs.stash.externalhooks.hook.ExternalHookScript;
import com.ngs.stash.externalhooks.util.ScopeUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.DisposableBean;
import org.springframework.beans.factory.InitializingBean;

@Named
public class BitbucketEventListener implements InitializingBean, DisposableBean {
  private static Logger log = LoggerFactory.getLogger(BitbucketEventListener.class);

  private final EventPublisher eventPublisher;
  private GlobalHookSettingsDao globalHookSettingsDao;
  private HookInstaller hookInstaller;
  private final RepositoryHookService repositoryHookService;

  @Inject
  public BitbucketEventListener(
      @ComponentImport EventPublisher eventPublisher,
      GlobalHookSettingsDao globalHookSettingsDao,
      @ComponentImport RepositoryHookService repositoryHookService,
      HookInstaller hookInstaller) {
    this.eventPublisher = eventPublisher;
    this.repositoryHookService = repositoryHookService;
    this.hookInstaller = hookInstaller;
    this.globalHookSettingsDao = globalHookSettingsDao;
  }

  @Override
  public void afterPropertiesSet() {
    eventPublisher.register(this);
  }

  @Override
  public void destroy() {
    eventPublisher.unregister(this);
  }

  @EventListener
  public void onHookEnabled(RepositoryHookEnabledEvent event) {
    ExternalHookScript script = hookInstaller.getScript(event.getRepositoryHookKey());
    // this will be null if there is no such hook (it's not ours hook)
    if (script == null) {
      return;
    }

    enableHook(event, script);
  }

  @EventListener
  public void onHookSettingsChanged(RepositoryHookSettingsChangedEvent event) {
    ExternalHookScript script = hookInstaller.getScript(event.getRepositoryHookKey());
    if (script == null) {
      return;
    }

    RepositoryHook hook = repositoryHookService.getByKey(
        event.getScope(), event.getRepositoryHookKey());
    if (!hook.isEnabled() || !hook.isConfigured()
        || ScopeUtil.isInheritedEnabled(hook, event.getScope())) {
      return;
    }

    enableHook(event, script);
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

    log.debug(
        "external hook disabled and hook scripts uninstalled: {} {}",
        event.getRepositoryHookKey(),
        ScopeUtil.toString(event.getScope()));
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

      log.debug(
          "external hook inherited and hook scripts configured: {} {}",
          event.getRepositoryHookKey(),
          ScopeUtil.toString(event.getScope()));
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

    log.debug(
        "external hook configured for created repository: {} {}",
        ScopeUtil.toString(new RepositoryScope(event.getRepository())));
  }

  @EventListener
  public void onRepositoryDeleted(RepositoryDeletedEvent event) {
    RepositoryScope scope = new RepositoryScope(event.getRepository());
    hookInstaller.getScripts().forEach((hookId, script) -> {
      script.uninstall(scope);
      script.uninstall(new GlobalScope(), scope);
    });

    log.debug(
        "external hook unconfigured for deleted repository: {} {}",
        ScopeUtil.toString(new RepositoryScope(event.getRepository())));
  }

  private void enableHook(RepositoryHookEvent event, ExternalHookScript script) {
    hookInstaller.enable(event.getScope(), script);

    GlobalHooks globalHooks = new GlobalHooks(this.globalHookSettingsDao.find());
    if (globalHooks.isEnabled(script.getHookKey())) {
      hookInstaller.enable(event.getScope(), script, globalHooks);
    }

    log.debug(
        "external hook enabled and hook scripts installed: {} {}",
        event.getRepositoryHookKey(),
        ScopeUtil.toString(event.getScope()));
  }
}

package com.ngs.stash.externalhooks;

import com.atlassian.bitbucket.hook.repository.RepositoryHook;
import com.atlassian.bitbucket.hook.repository.RepositoryHookSearchRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookService;
import com.atlassian.bitbucket.hook.script.HookScript;
import com.atlassian.bitbucket.scope.GlobalScope;
import com.atlassian.bitbucket.scope.ProjectScope;
import com.atlassian.bitbucket.scope.RepositoryScope;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.util.Page;
import com.atlassian.bitbucket.util.PageRequest;
import com.atlassian.bitbucket.util.PageRequestImpl;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.util.ScopeUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class HooksFactory {
  private static Logger log = LoggerFactory.getLogger(HooksFactory.class);
  private RepositoryHookService repositoryHookService;
  private HooksCoordinator hooksCoordinator;

  public HooksFactory(
      @ComponentImport RepositoryHookService repositoryHookService,
      @ComponentImport HooksCoordinator hooksCoordinator) {
    this.repositoryHookService = repositoryHookService;
    this.hooksCoordinator = hooksCoordinator;
  }

  /**
   * Re-creates Atlassian {@link HookScript} for every {@link RepositoryHook}. Works with both
   * {@link ProjectScope} and {@link RepositoryScope}
   *
   * @param scope
   */
  public void apply(Scope scope, GlobalHooks globalHooks) {
    log.debug("creating hook scripts on {}", ScopeUtil.toString(scope));

    RepositoryHookSearchRequest.Builder searchBuilder =
        new RepositoryHookSearchRequest.Builder(scope);

    Page<RepositoryHook> page = repositoryHookService.search(
        searchBuilder.build(), new PageRequestImpl(0, PageRequest.MAX_PAGE_LIMIT));

    Integer created = 0;
    for (RepositoryHook hook : page.getValues()) {
      String hookKey = hook.getDetails().getKey();
      if (!hookKey.startsWith(Const.PLUGIN_KEY)) {
        continue;
      }

      boolean skipScopeWide = false;
      if (!hook.isEnabled() || !hook.isConfigured()) {
        skipScopeWide = true;
      } else if (ScopeUtil.isInheritedEnabled(hook, scope)) {
        // if this is a repository and we have a project's hook but don't have
        // if we don't have a global hook
        log.info(
            "hook {} is enabled & configured (inherited of {})",
            hookKey,
            ScopeUtil.toString(hook.getScope()));
        skipScopeWide = true;
      }

      if (!skipScopeWide) {
        try {
          hooksCoordinator.enable(scope, hookKey);

          created++;
        } catch (Exception e) {
          e.printStackTrace();

          log.error("Unable to install hook script {}: {}", hookKey, e.toString());
        }
      }

      if (globalHooks.isEnabled(hookKey)) {
        hooksCoordinator.enable(scope, hookKey, globalHooks.getSettings(hookKey));
      } else {
        hooksCoordinator.disable(scope, hookKey, new GlobalScope());
      }
    }

    log.info("created {} hook scripts on scope {}", created, ScopeUtil.toString(scope));
  }
}

package com.ngs.stash.externalhooks.hook;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.hook.repository.PostRepositoryHook;
import com.atlassian.bitbucket.hook.repository.PostRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.RepositoryHookRequest;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.setting.SettingsValidator;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.HookInstaller;

public class ExternalAsyncPostReceiveHook
    implements PostRepositoryHook<RepositoryHookRequest>, SettingsValidator {
  private HookInstaller hookInstaller;

  public ExternalAsyncPostReceiveHook(@ComponentImport HookInstaller hookInstaller) {
    this.hookInstaller = hookInstaller;
  }

  @Override
  public void validate(
      @Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
    hookInstaller.validate(Const.POST_RECEIVE_HOOK_ID, settings, errors, scope);
  }

  @Override
  public void postUpdate(
      @Nonnull PostRepositoryHookContext context, @Nonnull RepositoryHookRequest request) {}
}

package com.ngs.stash.externalhooks.hook;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.hook.repository.PreRepositoryHook;
import com.atlassian.bitbucket.hook.repository.PreRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.RepositoryHookRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookResult;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.setting.SettingsValidator;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.HooksCoordinator;
import com.ngs.stash.externalhooks.LicenseValidator;

public class ExternalPreReceiveHook
    implements PreRepositoryHook<RepositoryHookRequest>, SettingsValidator {
  private HooksCoordinator hooksCoordinator;
  private LicenseValidator licenseValidator;

  public ExternalPreReceiveHook(
      @ComponentImport HooksCoordinator hooksCoordinator,
      @ComponentImport LicenseValidator licenseValidator) {
    this.licenseValidator = licenseValidator;
    this.hooksCoordinator = hooksCoordinator;
  }

  @Override
  public void validate(
      @Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
    hooksCoordinator.validate(Const.PRE_RECEIVE_HOOK_ID, settings, errors, scope);
  }

  @Nonnull
  @Override
  public RepositoryHookResult preUpdate(
      @Nonnull PreRepositoryHookContext context, @Nonnull RepositoryHookRequest request) {
    if (!licenseValidator.isDefined()) {
      return RepositoryHookResult.rejected(
          "Unlicensed Add-on.",
          "License for External Hooks Add-on is missing.\n"
              + "Visit \"Manage Apps\" page in your Bitbucket instance for more info.");
    }

    if (!licenseValidator.isValid()) {
      return RepositoryHookResult.rejected(
          "License is not valid.",
          "License for External Hooks Add-on is expired.\n"
              + "Visit \"Manage Apps\" page in your Bitbucket instance for more info.");
    }

    return RepositoryHookResult.accepted();
  }
}

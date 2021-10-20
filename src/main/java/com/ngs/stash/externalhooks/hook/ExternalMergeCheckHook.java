package com.ngs.stash.externalhooks.hook;

import javax.annotation.Nonnull;

import com.atlassian.bitbucket.hook.repository.PreRepositoryHookContext;
import com.atlassian.bitbucket.hook.repository.PullRequestMergeHookRequest;
import com.atlassian.bitbucket.hook.repository.RepositoryHookResult;
import com.atlassian.bitbucket.hook.repository.RepositoryMergeCheck;
import com.atlassian.bitbucket.scope.Scope;
import com.atlassian.bitbucket.setting.Settings;
import com.atlassian.bitbucket.setting.SettingsValidationErrors;
import com.atlassian.bitbucket.setting.SettingsValidator;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.ngs.stash.externalhooks.Const;
import com.ngs.stash.externalhooks.HookInstaller;
import com.ngs.stash.externalhooks.LicenseValidator;

public class ExternalMergeCheckHook implements RepositoryMergeCheck, SettingsValidator {
  private HookInstaller hookInstaller;
  private LicenseValidator licenseValidator;

  public ExternalMergeCheckHook(
      @ComponentImport HookInstaller hookInstaller,
      @ComponentImport LicenseValidator licenseValidator) {
    this.licenseValidator = licenseValidator;
    this.hookInstaller = hookInstaller;
  }

  @Override
  public void validate(
      @Nonnull Settings settings, @Nonnull SettingsValidationErrors errors, @Nonnull Scope scope) {
    hookInstaller.validate(Const.MERGE_CHECK_HOOK_ID, settings, errors, scope);
  }

  @Override
  public RepositoryHookResult preUpdate(
      PreRepositoryHookContext context, PullRequestMergeHookRequest request) {
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

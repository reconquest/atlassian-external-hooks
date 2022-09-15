package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/cog"
	"github.com/reconquest/pkg/log"
)

func (suite *Suite) TestGlobalHooks_PersonalRepositoriesFilter(
	params TestParams,
) {
	suite.UseBitbucket(params.Bitbucket, params.Cluster)
	suite.InstallAddon(params.Addon)
	suite.RecordHookScripts()

	context := suite.ExternalHooks().OnGlobal()

	log := log.NewChildWithPrefix("{test: global hooks/personal repositories}")

	suite.testGlobalHooks_PersonalRepositoriesFilter_SwitchFilter(log, context)

	suite.testGlobalHooks_PersonalRepositoriesFilter_Settings(
		log,
		context,
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_EXCLUDE_PERSONAL),
		func(userRepositoryBefore, userRepositoryAfter, repositoryBefore, repositoryAfter *stash.Repository) {
			Assert_PushRejected(suite, repositoryBefore, `XXXGLOBAL`)
			Assert_PushRejected(suite, repositoryAfter, `XXXGLOBAL`)
			Assert_PushDoesNotOutputMessages(
				suite,
				userRepositoryBefore,
				`XXXGLOBAL`,
			)
			Assert_PushDoesNotOutputMessages(
				suite,
				userRepositoryAfter,
				`XXXGLOBAL`,
			)
		},
	)

	suite.testGlobalHooks_PersonalRepositoriesFilter_Settings(
		log,
		context,
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_DISABLED),
		func(userRepositoryBefore, userRepositoryAfter, repositoryBefore, repositoryAfter *stash.Repository) {
			Assert_PushRejected(suite, repositoryBefore, `XXXGLOBAL`)
			Assert_PushRejected(suite, repositoryAfter, `XXXGLOBAL`)
			Assert_PushRejected(suite, userRepositoryBefore, `XXXGLOBAL`)
			Assert_PushRejected(suite, userRepositoryAfter, `XXXGLOBAL`)
		},
	)

	suite.testGlobalHooks_PersonalRepositoriesFilter_Settings(
		log,
		context,
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_ONLY_PERSONAL),
		func(userRepositoryBefore, userRepositoryAfter, repositoryBefore, repositoryAfter *stash.Repository) {
			Assert_PushDoesNotOutputMessages(
				suite,
				repositoryBefore,
				`XXXGLOBAL`,
			)
			Assert_PushDoesNotOutputMessages(
				suite,
				repositoryAfter,
				`XXXGLOBAL`,
			)
			Assert_PushRejected(suite, userRepositoryBefore, `XXXGLOBAL`)
			Assert_PushRejected(suite, userRepositoryAfter, `XXXGLOBAL`)
		},
	)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testGlobalHooks_PersonalRepositoriesFilter_Settings(
	log *cog.Logger,
	context *external_hooks.Context,
	settings *external_hooks.GlobalSettings,
	asserter func(userRepositoryBefore, userRepositoryAfter, repositoryBefore, repositoryAfter *stash.Repository),
) {
	userProject := &stash.Project{
		Key: "~admin",
	}
	userRepositoryBefore := suite.CreateRandomRepository(userProject)

	project := suite.CreateRandomProject()
	repositoryBefore := suite.CreateRandomRepository(project)

	hook := suite.ConfigureSettingsHook(
		context.PreReceive(),
		settings.
			UseSafePath(true).
			WithExe(`hook.`+lojban.GetRandomID(5)),
		`echo XXXGLOBAL; exit 1`,
	)

	// suite.WaitExternalHookEnabled()

	userRepositoryAfter := suite.CreateRandomRepository(userProject)
	repositoryAfter := suite.CreateRandomRepository(project)

	asserter(
		userRepositoryBefore,
		userRepositoryAfter,
		repositoryBefore,
		repositoryAfter,
	)

	suite.DisableHook(hook)
	// suite.WaitExternalHookDisabled()
}

func (suite *Suite) testGlobalHooks_PersonalRepositoriesFilter_SwitchFilter(
	log *cog.Logger,
	context *external_hooks.Context,
) {
	userProject := &stash.Project{
		Key: "~admin",
	}
	userRepository := suite.CreateRandomRepository(userProject)

	hook := suite.ConfigureSettingsHook(
		context.PreReceive(),
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_DISABLED).
			UseSafePath(true).
			WithExe(`hook.`+lojban.GetRandomID(5)),
		`echo XXX_DISABLED; exit 1`,
	)

	// suite.WaitExternalHookEnabled()

	Assert_PushRejected(suite, userRepository, "XXX_DISABLED")

	hook = suite.ConfigureSettingsHook(
		context.PreReceive(),
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_ONLY_PERSONAL).
			UseSafePath(true).
			WithExe(`hook.`+lojban.GetRandomID(5)),
		`echo XXX_ONLY_PERSONAL; exit 1`,
	)

	// suite.WaitExternalHookEnabled()

	Assert_PushRejected(suite, userRepository, "XXX_ONLY_PERSONAL")

	hook = suite.ConfigureSettingsHook(
		context.PreReceive(),
		external_hooks.NewGlobalSettings().
			WithFilterPersonalRepositories(external_hooks.FILTER_PERSONAL_REPOSITORIES_EXCLUDE_PERSONAL).
			UseSafePath(true).
			WithExe(`hook.`+lojban.GetRandomID(5)),
		`echo XXX_EXCLUDE_PERSONAL; exit 1`,
	)

	// suite.WaitExternalHookEnabled()

	Assert_PushDoesNotOutputMessages(
		suite,
		userRepository,
		"XXX_EXCLUDE_PERSONAL",
	)

	suite.DisableHook(hook)
	// suite.WaitExternalHookDisabled()
}

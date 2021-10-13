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
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	context := suite.ExternalHooks().OnGlobal()

	log := log.NewChildWithPrefix("{test: global hooks/personal repositories}")

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
		HookOptions{
			WaitHookScripts: true,
		},
		`echo XXXGLOBAL; exit 1`,
	)

	userRepositoryAfter := suite.CreateRandomRepository(userProject)
	repositoryAfter := suite.CreateRandomRepository(project)

	asserter(
		userRepositoryBefore,
		userRepositoryAfter,
		repositoryBefore,
		repositoryAfter,
	)

	suite.DisableHook(hook, HookOptions{WaitHookScripts: true})
}

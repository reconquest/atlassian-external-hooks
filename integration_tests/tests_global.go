package main

import (
	"context"
	"strings"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/cog"
	"github.com/reconquest/pkg/log"
)

func (suite *Suite) TestGlobalHooks(params TestParams) {
	suite.UseBitbucket(params.Bitbucket, params.Cluster)
	suite.InstallAddon(params.Addon)
	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnGlobal()

	log := log.NewChildWithPrefix("{test: global hooks}")

	suite.testGlobalHooks(log, context, project, repository)

	suite.testPreReceive(log, context, repository)
	suite.testPostReceive(log, context, repository)
	suite.testMergeCheck(log, context, repository)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testGlobalHooks(
	log *cog.Logger,
	context *external_hooks.Context,
	project *stash.Project,
	repo *stash.Repository,
) {
	suite.testGlobalHooks_ProjectOrRepository_EnabledOrDisabled(
		log, context, project, repo,
		context.
			OnProject(project.Key).
			PreReceive(),
	)
	suite.testGlobalHooks_ProjectOrRepository_EnabledOrDisabled(
		log, context, project, repo,
		context.
			OnProject(project.Key).
			OnRepository(repo.Slug).
			PreReceive(),
	)

	suite.testGlobalHooks_RepositoryDeleted(log, context)
	suite.testGlobalHooks_DoubleEnable(log, context)
	suite.testGlobalHooks_RepositoryCreatedAfterEnabling(log, context)
}

func (suite *Suite) testGlobalHooks_RepositoryCreatedAfterEnabling(
	log *cog.Logger,
	context *external_hooks.Context,
) {
	hook := context.PreReceive()

	const message = `XXX`
	suite.ConfigureSampleHook_FailWithMessage(
		hook,
		message,
	)

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	Assert_PushRejected(suite, repository, message)

	suite.DisableHook(hook)

	Assert_PushDoesNotOutputMessages(suite, repository, message)
}

func (suite *Suite) testGlobalHooks_ProjectOrRepository_EnabledOrDisabled(
	log *cog.Logger,
	context *external_hooks.Context,
	project *stash.Project,
	repo *stash.Repository,
	resourceHook *external_hooks.Hook,
) {
	globalHook := context.PreReceive()

	enableGlobal := func() {
		suite.ConfigureSampleHook_Message(
			globalHook,
			`XXX_GLOBAL`,
		)

		// suite.WaitExternalHookEnabled()
	}

	enableGlobal()

	suite.ConfigureSampleHook_Message(
		resourceHook,
		`XXX_RESOURCE`,
	)

	suite.WaitExternalHookEnabled(resourceHook)

	Assert_PushOutputsMessages(suite, repo, `XXX_GLOBAL`, `XXX_RESOURCE`)

	suite.DisableHook(globalHook)

	Assert_PushOutputsMessages(suite, repo, `XXX_RESOURCE`)
	Assert_PushDoesNotOutputMessages(suite, repo, `XXX_GLOBAL`)

	enableGlobal()
	suite.DisableHook(resourceHook)

	Assert_PushOutputsMessages(suite, repo, `XXX_GLOBAL`)
	Assert_PushDoesNotOutputMessages(suite, repo, `XXX_RESOURCE`)

	suite.DisableHook(globalHook)
}

func (suite *Suite) testGlobalHooks_DoubleEnable(
	log *cog.Logger,
	context *external_hooks.Context,
) {
	suite.RecordHookScripts()

	hook := context.PreReceive()

	suite.ConfigureSampleHook_FailWithMessage(
		hook,
		`XXX`,
	)

	suite.CreateRandomRepository(suite.CreateRandomProject())

	suite.EnableHook(hook)

	suite.CreateRandomRepository(suite.CreateRandomProject())

	suite.EnableHook(hook)

	suite.CreateRandomRepository(suite.CreateRandomProject())

	suite.DisableHook(hook)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testGlobalHooks_RepositoryDeleted(
	log *cog.Logger,
	hooks *external_hooks.Context,
) {
	hook := hooks.PreReceive()

	suite.ConfigureSampleHook_FailWithMessage(
		hook,
		`XXX`,
	)

	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	waiter := suite.Bitbucket().WaitLog(
		context.Background(),
		bitbucket.LOGS_TESTCASES,
		func(line string) bool {
			return strings.Contains(
				line,
				"deleted global/repository hook script",
			)
		},
		bitbucket.DEFAULT_LOG_WAIT_TIMEOUT,
	)

	err := suite.Bitbucket().Repositories(project.Key).Remove(repository.Slug)
	suite.NoError(err, "remove repository")

	waiter.Wait(suite.FailNow, "hook scripts", "deleted")

	suite.DetectHookScriptsLeak()
}

package main

import (
	"fmt"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/cog"
	"github.com/reconquest/pkg/log"
)

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
	log *cog.Logger,
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))

	suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		`XXX`,
	)

	Assert_PushRejected(suite, repository, `XXX`)

	suite.DisableHook(context.OnRepository(repository.Slug).PreReceive())

	Assert_PushRejected(suite, repository, `XXX`)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Fixed(
	log *cog.Logger,
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))
	suite.RecordHookScripts()

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)

	suite.DisableHook(context.PreReceive())

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryDisabledHooks(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	var (
		context = suite.ExternalHooks().OnProject(project.Key)
		log     = log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))
	)

	suite.testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
		log,
		params,
		context,
		project,
		repository,
	)

	suite.testBug_ProjectEnabledRepositoryDisabledHooks_Fixed(
		log,
		params,
		context,
		project,
		repository,
	)
}

func (suite *Suite) TestBug_ProjectHookCreatedBeforeRepository(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	var (
		context = suite.ExternalHooks().OnProject(project.Key)
		log     = log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))
	)

	suite.testBug_ProjectHookCreatedBeforeRepository_Reproduced(
		log,
		params,
		context,
		project,
	)

	suite.testBug_ProjectHookCreatedBeforeRepository_Fixed(
		log,
		params,
		context,
		project,
	)
}

func (suite *Suite) testBug_ProjectHookCreatedBeforeRepository_Reproduced(
	log *cog.Logger,
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
) {
	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))
	suite.RecordHookScripts()

	preReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		`XXX`,
	)

	repository := suite.CreateRandomRepository(project)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)

	suite.DisableHook(preReceive)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testBug_ProjectHookCreatedBeforeRepository_Fixed(
	log *cog.Logger,
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
) {
	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))
	suite.RecordHookScripts()

	preReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		`XXX`,
	)

	repository := suite.CreateRandomRepository(project)

	Assert_PushRejected(suite, repository, `XXX`)

	suite.DisableHook(preReceive)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testBug_ProjectEnabledRepositoryOverriddenHooks_Reproduced(
	log *cog.Logger,
	params TestParams,
	settings *external_hooks.Settings,
	project *stash.Project,
) {
	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))

	var repository = suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithArgs(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
	)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithArgs(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
	)

	Assert_PushOutputsMessages(suite, repository, `XXX PROJECT`)
	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.DisableHook(preReceiveProject)
	suite.DisableHook(preReceiveRepository)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryOverriddenHooks_Fixed(
	log *cog.Logger,
	params TestParams,
	settings *external_hooks.Settings,
	project *stash.Project,
) {
	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))

	repository := suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithArgs(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
	)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithArgs(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
	)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX PROJECT`)
	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.DisableHook(preReceiveProject)
	suite.DisableHook(preReceiveRepository)
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryOverriddenHooks(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	log := log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))

	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`hook.` + lojban.GetRandomID(5))

	suite.testBug_ProjectEnabledRepositoryOverriddenHooks_Reproduced(
		log,
		params,
		settings,
		project,
	)

	suite.testBug_ProjectEnabledRepositoryOverriddenHooks_Fixed(
		log,
		params,
		settings,
		project,
	)
}

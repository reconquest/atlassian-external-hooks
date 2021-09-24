package main

import (
	"fmt"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/pkg/log"
)

func (suite *Suite) TestBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
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

	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))

	suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX`,
	)

	Assert_PushRejected(suite, repository, `XXX`)

	suite.DisableHook(context.OnRepository(repository.Slug).PreReceive())

	Assert_PushRejected(suite, repository, `XXX`)
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryDisabledHooks_Fixed(
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

	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))
	suite.RecordHookScripts()

	suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX`,
	)

	suite.DisableHook(context.OnRepository(repository.Slug).PreReceive())

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)

	suite.DisableHook(context.PreReceive(), HookOptions{WaitHookScripts: false})

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_ProjectHookCreatedBeforeRepository_Reproduced(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	var (
		context = suite.ExternalHooks().OnProject(project.Key)
		log     = log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))
	)

	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))
	suite.RecordHookScripts()

	preReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: false},
		`XXX`,
	)

	repository := suite.CreateRandomRepository(project)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)

	suite.DisableHook(preReceive, HookOptions{WaitHookScripts: false})

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_ProjectHookCreatedBeforeRepository_Fixed(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	var (
		projectContext = suite.ExternalHooks().OnProject(project.Key)
		log            = log.NewChildWithPrefix(
			fmt.Sprintf("{test} %s", project.Key),
		)
	)

	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))
	suite.RecordHookScripts()

	preReceive := suite.ConfigureSampleHook_FailWithMessage(
		projectContext.PreReceive(),
		HookOptions{WaitHookScripts: false},
		`XXX`,
	)

	repository := suite.CreateRandomRepository(project)

	suite.WaitHookScriptsCreated()

	Assert_PushRejected(suite, repository, `XXX`)

	suite.DisableHook(preReceive, HookOptions{WaitHookScripts: true})

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_RepositoryHookCreatedBeforeProject_Reproduced(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	var (
		project        = suite.CreateRandomProject()
		projectContext = suite.ExternalHooks().OnProject(project.Key)

		repository        = suite.CreateRandomRepository(project)
		repositoryContext = suite.ExternalHooks().OnProject(project.Key).
					OnRepository(repository.Slug)

		log = log.NewChildWithPrefix(
			fmt.Sprintf("{test} %s", project.Key),
		)
	)

	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))
	suite.RecordHookScripts()

	// repository first and project second
	repositoryPreReceive := suite.ConfigureSampleHook_Message(
		repositoryContext.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX_REPOSITORY_XXX`,
	)

	projectPreReceive := suite.ConfigureSampleHook_Message(
		projectContext.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX_PROJECT_XXX`,
	)

	Assert_PushOutputsMessages(
		suite,
		repository,
		`XXX_PROJECT_XXX`,
		`XXX_REPOSITORY_XXX`,
	)

	suite.DisableHook(projectPreReceive)
	suite.DisableHook(repositoryPreReceive)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_RepositoryHookCreatedBeforeProject_Fixed(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	var (
		project        = suite.CreateRandomProject()
		projectContext = suite.ExternalHooks().OnProject(project.Key)

		repository        = suite.CreateRandomRepository(project)
		repositoryContext = suite.ExternalHooks().OnProject(project.Key).
					OnRepository(repository.Slug)

		log = log.NewChildWithPrefix(
			fmt.Sprintf("{test} %s", project.Key),
		)
	)

	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))
	suite.RecordHookScripts()

	// repository first and project second
	repositoryPreReceive := suite.ConfigureSampleHook_Message(
		repositoryContext.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX_REPOSITORY_XXX`,
	)

	projectPreReceive := suite.ConfigureSampleHook_Message(
		projectContext.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX_PROJECT_XXX`,
	)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX_PROJECT_XXX`)
	Assert_PushOutputsMessages(suite, repository, `XXX_REPOSITORY_XXX`)

	suite.DisableHook(repositoryPreReceive)
	suite.DisableHook(projectPreReceive, HookOptions{WaitHookScripts: false})

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryOverriddenHooks_Reproduced(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	log := log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))

	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExe(`hook.` + lojban.GetRandomID(5))

	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))

	repository := suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithParams(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithParams(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	Assert_PushOutputsMessages(suite, repository, `XXX PROJECT`)
	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.DisableHook(preReceiveProject)
	suite.DisableHook(preReceiveRepository)
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryOverriddenHooks_Fixed(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	log := log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))

	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExe(`hook.` + lojban.GetRandomID(5))

	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))

	repository := suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithParams(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithParams(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX PROJECT`)
	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.DisableHook(preReceiveProject, HookOptions{WaitHookScripts: false})
	suite.DisableHook(preReceiveRepository)
}

func (suite *Suite) TestBug_UserWithoutProjectAccessModifiesInheritedHook_Reproduced(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	log := log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))

	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExe(`hook.` + lojban.GetRandomID(5))

	log.Infof(
		nil,
		"> reproducing bug on add-on version %s",
		params["addon_reproduced"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_reproduced"].(Addon))

	repository := suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithParams(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	alice := suite.CreateUserAlice()

	err := suite.Bitbucket().
		Repositories(project.Key).
		Permissions(repository.Slug).
		GrantUserPermission(alice.Name, "REPO_ADMIN")
	suite.NoError(err)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks(alice).
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithParams(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.InheritHook(
		preReceiveRepository,
		InheritHookExpectedStateEnabledProject,
	)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX PROJECT`)

	suite.DisableHook(preReceiveProject)
}

func (suite *Suite) TestBug_UserWithoutProjectAccessModifiesInheritedHook_Fixed(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

	project := suite.CreateRandomProject()

	log := log.NewChildWithPrefix(fmt.Sprintf("{test} %s", project.Key))

	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExe(`hook.` + lojban.GetRandomID(5))

	log.Infof(
		nil,
		"> validating fix on add-on version %s",
		params["addon_fixed"].(Addon).Version,
	)

	suite.InstallAddon(params["addon_fixed"].(Addon))

	repository := suite.CreateRandomRepository(project)

	preReceiveProject := suite.ConfigureHook(
		suite.ExternalHooks().
			OnProject(project.Key).
			PreReceive(),
		settings.WithParams(`XXX PROJECT`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	alice := suite.CreateUserAlice()

	err := suite.Bitbucket().
		Repositories(project.Key).
		Permissions(repository.Slug).
		GrantUserPermission(alice.Name, "REPO_ADMIN")
	suite.NoError(err)

	preReceiveRepository := suite.ConfigureHook(
		suite.ExternalHooks(alice).
			OnProject(project.Key).
			OnRepository(repository.Slug).
			PreReceive(),
		settings.WithParams(`YYY REPOSITORY`),
		text(
			`#!/bin/bash`,
			`echo $1`,
		),
		HookOptions{WaitHookScripts: true},
	)

	Assert_PushOutputsMessages(suite, repository, `YYY REPOSITORY`)

	suite.InheritHook(
		preReceiveRepository,
		InheritHookExpectedStateEnabledProject,
	)

	Assert_PushOutputsMessages(suite, repository, `XXX PROJECT`)

	suite.DisableHook(preReceiveProject)
}

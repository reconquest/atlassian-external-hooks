package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
)

func (suite *Suite) TestProjectHooks() {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := suite.Bitbucket().Projects().Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	context := suite.ExternalHooks().OnProject(project.Key)

	suite.testBasicPreReceiveScenario(nil, context, repository)
	suite.testBasicPostReceiveScenario(nil, context, repository)
}

func (suite *Suite) TestRepositoryHooks() {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := suite.Bitbucket().Projects().
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(nil, context, repository)
	suite.testBasicPostReceiveScenario(nil, context, repository)
}

func (suite *Suite) TestProjectEnabledRepositoryDisabledHooks() {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := suite.Bitbucket().Projects().
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	context := suite.ExternalHooks().OnProject(project.Key)

	{
		hook := suite.ConfigurePreReceiveHook(context, `pre.fail.sh`, text(
			`#!/bin/bash`,
			`echo XXX`,
			`exit 1`,
		))

		Testcase_PushRejected(suite, repository, `XXX`)

		err = context.OnRepository(repository.Slug).PreReceive(hook.Settings).
			Disable()
		suite.NoError(err, "unable to disable repository hook")

		Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)
	}

	{
		hook := suite.ConfigurePostReceiveHook(context, `post.fail.sh`, text(
			`#!/bin/bash`,
			`echo YYY`,
			`exit 1`,
		))

		Testcase_PushOutputsMessage(suite, repository, `YYY`)

		err = context.OnRepository(repository.Slug).PostReceive(hook.Settings).
			Disable()
		suite.NoError(err, "unable to disable repository hook")

		Testcase_PushDoesNotOutputMessage(suite, repository, `YYY`)
	}
}

func (suite *Suite) TestPersonalRepositoriesHooks() {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project := &stash.Project{
		Key: "~admin",
	}

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(nil, context, repository)
	suite.testBasicPostReceiveScenario(nil, context, repository)
}

func (suite *Suite) TestBitbucketUpgrade() {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := suite.Bitbucket().Projects().Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	context := suite.ExternalHooks().OnProject(project.Key)

	pre := suite.testBasicPreReceiveScenario(nil, context, repository)
	post := suite.testBasicPostReceiveScenario(nil, context, repository)

	err = pre.Enable()
	suite.NoError(err, "unable to enable pre-receive hook")

	err = post.Enable()
	suite.NoError(err, "unable to enable post-receive hook")

	suite.UseBitbucket("6.9.0")

	suite.testBasicPreReceiveScenario(pre, context, repository)
	suite.testBasicPostReceiveScenario(post, context, repository)
}

func (suite *Suite) testBasicPreReceiveScenario(
	hook *external_hooks.Hook,
	context *external_hooks.Context,
	repository *stash.Repository,
) *external_hooks.Hook {
	if hook == nil {
		hook = suite.ConfigurePreReceiveHook(context, `pre.fail.sh`, text(
			`#!/bin/bash`,
			`echo XXX`,
			`exit 1`,
		))
	}

	Testcase_PushRejected(suite, repository, `XXX`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)

	return hook
}

func (suite *Suite) testBasicPostReceiveScenario(
	hook *external_hooks.Hook,
	context *external_hooks.Context,
	repository *stash.Repository,
) *external_hooks.Hook {
	if hook == nil {
		hook = suite.ConfigurePostReceiveHook(context, `post.fail.sh`, text(
			`#!/bin/bash`,
			`echo YYY`,
			`exit 1`,
		))
	}

	Testcase_PushOutputsMessage(suite, repository, `YYY`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `YYY`)

	return hook
}

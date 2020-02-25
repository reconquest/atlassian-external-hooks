package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
)

func (suite *Suite) TestProjectHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"])
	suite.InstallAddon(params["addon"])

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
}

func (suite *Suite) TestRepositoryHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"])
	suite.InstallAddon(params["addon"])

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
}

func (suite *Suite) TestPersonalRepositoriesHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"])
	suite.InstallAddon(params["addon"])

	project := &stash.Project{
		Key: "~admin",
	}

	var (
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	addon := suite.InstallAddon(params["addon_reproduced"])

	hook := suite.CreateSamplePreReceiveHook_FailWithMessage(
		context,
		`XXX`,
	)

	Testcase_PushRejected(suite, repository, `XXX`)

	err := context.OnRepository(repository.Slug).
		PreReceive(hook.Settings).
		Disable()
	suite.NoError(err, "unable to disable repository hook")

	Testcase_PushRejected(suite, repository, `XXX`)

	suite.UninstallAddon(addon)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Fixed(
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	addon := suite.InstallAddon(params["addon_fixed"])

	Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)

	suite.UninstallAddon(addon)
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryDisabledHooks(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"])

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	suite.testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
		params,
		context,
		project,
		repository,
	)

	suite.testBug_ProjectEnabledRepositoryDisabledHooks_Fixed(
		params,
		context,
		project,
		repository,
	)
}

func (suite *Suite) TestBitbucketUpgrade(params TestParams) {
	suite.UseBitbucket(params["bitbucket_from"])
	suite.InstallAddon(params["addon"])

	var cases struct {
		public, personal struct {
			repo *stash.Repository
			pre  *external_hooks.Hook
			post *external_hooks.Hook
		}
	}

	{
		var (
			project = suite.CreateRandomProject()
		)

		cases.public.repo = suite.CreateRandomRepository(project)

		context := suite.ExternalHooks().OnProject(project.Key)

		cases.public.pre, cases.public.post = suite.testBitbucketUpgrade_Before(
			project, cases.public.repo, context,
		)
	}

	{
		project := &stash.Project{
			Key: "~admin",
		}

		cases.personal.repo = suite.CreateRandomRepository(project)

		context := suite.ExternalHooks().OnProject(project.Key).
			OnRepository(cases.personal.repo.Slug)

		cases.personal.pre, cases.personal.post = suite.testBitbucketUpgrade_Before(
			project, cases.personal.repo, context,
		)
	}

	suite.UseBitbucket(params["bitbucket_to"])

	{
		suite.testBitbucketUpgrade_After(
			cases.public.repo,
			cases.public.pre,
			cases.public.post,
		)
	}

	{
		suite.testBitbucketUpgrade_After(
			cases.personal.repo,
			cases.personal.pre,
			cases.personal.post,
		)
	}
}

func (suite *Suite) testBitbucketUpgrade_Before(
	project *stash.Project,
	repo *stash.Repository,
	context *external_hooks.Context,
) (*external_hooks.Hook, *external_hooks.Hook) {
	pre := suite.ConfigurePreReceiveHook(
		context,
		`pre.fail.sh`,
		text(
			`#!/bin/bash`,
			`echo XXX`,
			`exit 1`,
		),
	)

	Testcase_PushRejected(suite, repo, `XXX`)

	err := pre.Disable()
	suite.NoError(err, "unable to disable pre-receive hook")

	Testcase_PushDoesNotOutputMessage(suite, repo, `XXX`)

	post := suite.ConfigurePostReceiveHook(
		context,
		`post.fail.sh`,
		text(
			`#!/bin/bash`,
			`echo YYY`,
			`exit 1`,
		),
	)

	Testcase_PushOutputsMessage(suite, repo, `YYY`)

	err = post.Disable()
	suite.NoError(err, "unable to disable post-receive hook")

	Testcase_PushDoesNotOutputMessage(suite, repo, `YYY`)

	err = pre.Enable()
	suite.NoError(err, "unable to enable pre-receive hook")

	err = post.Enable()
	suite.NoError(err, "unable to enable post-receive hook")

	return pre, post
}

func (suite *Suite) testBitbucketUpgrade_After(
	repo *stash.Repository,
	pre, post *external_hooks.Hook,
) {
	pre.BitbucketURI = suite.Bitbucket().GetConnectorURI()
	post.BitbucketURI = suite.Bitbucket().GetConnectorURI()

	Testcase_PushRejected(suite, repo, `XXX`)

	err := pre.Disable()
	suite.NoError(err, "unable to disable pre-receive hook")

	Testcase_PushOutputsMessage(suite, repo, `YYY`)

	err = post.Disable()
	suite.NoError(err, "unable to disable post-receive hook")

	Testcase_PushDoesNotOutputMessage(suite, repo, `YYY`)
}

func (suite *Suite) testBasicPreReceiveScenario(
	context *external_hooks.Context,
	repository *stash.Repository,
) *external_hooks.Hook {
	hook := suite.CreateSamplePreReceiveHook_FailWithMessage(context, `XXX`)

	Testcase_PushRejected(suite, repository, `XXX`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)

	return hook
}

func (suite *Suite) testBasicPostReceiveScenario(
	context *external_hooks.Context,
	repository *stash.Repository,
) *external_hooks.Hook {
	hook := suite.CreateSamplePostReceiveHook_FailWithMessage(context, `YYY`)

	Testcase_PushOutputsMessage(suite, repository, `YYY`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `YYY`)

	return hook
}

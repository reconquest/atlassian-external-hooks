package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
)

func (suite *Suite) TestProjectHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(
		params["addon"].(Addon).Version,
		params["addon"].(Addon).Path,
	)

	var (
		project     = suite.CreateRandomProject()
		repository  = suite.CreateRandomRepository(project)
		pullRequest = suite.CreateRandomPullRequest(project, repository)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) TestRepositoryHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(
		params["addon"].(Addon).Version,
		params["addon"].(Addon).Path,
	)

	var (
		project     = suite.CreateRandomProject()
		repository  = suite.CreateRandomRepository(project)
		pullRequest = suite.CreateRandomPullRequest(project, repository)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) TestPersonalRepositoriesHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(
		params["addon"].(Addon).Version,
		params["addon"].(Addon).Path,
	)

	project := &stash.Project{
		Key: "~admin",
	}

	var (
		repository  = suite.CreateRandomRepository(project)
		pullRequest = suite.CreateRandomPullRequest(project, repository)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testBasicPreReceiveScenario(context, repository)
	suite.testBasicPostReceiveScenario(context, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	addon := suite.InstallAddon(
		params["addon_reproduced"].(Addon).Version,
		params["addon_reproduced"].(Addon).Path,
	)

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
	addon := suite.InstallAddon(
		params["addon_fixed"].(Addon).Version,
		params["addon_fixed"].(Addon).Path,
	)

	Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)

	suite.UninstallAddon(addon)
}

func (suite *Suite) TestBug_ProjectEnabledRepositoryDisabledHooks(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))

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
	suite.UseBitbucket(params["bitbucket_from"].(string))
	suite.InstallAddon(
		params["addon"].(Addon).Version,
		params["addon"].(Addon).Path,
	)

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

	suite.UseBitbucket(params["bitbucket_to"].(string))

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

func (suite *Suite) testBasicMergeCheckScenario(
	context *external_hooks.Context,
	repository *stash.Repository,
	pullRequest *stash.PullRequest,
) *external_hooks.Hook {
	hook := suite.CreateSampleMergeCheckHook_FailWithMessage(context, `ZZZ`)

	service := suite.Bitbucket().Repositories(repository.Project.Key).
		PullRequests(repository.Slug)

	pullRequest, err := service.Get(pullRequest.ID)
	suite.NoError(err, "unable to get pull request object")

	result, err := service.Merge(
		pullRequest.ID,
		pullRequest.Version,
	)
	suite.NoError(err, "unable to get merge pull request result")

	suite.Equal(
		len(result.Errors),
		1,
		"no errors found in merge response",
	)
	suite.Equal(
		len(result.Errors[0].Vetoes),
		1,
		"no vetoes found in merge response",
	)
	suite.Equal(
		result.Errors[0].Vetoes[0].SummaryMessage,
		"external-merge-check-hook declined",
	)

	suite.Contains(
		result.Errors[0].Vetoes[0].DetailedMessage,
		"ZZZ",
	)

	err = hook.Disable()
	suite.NoError(err, "unable to disable hook")

	pullRequest, err = service.Get(pullRequest.ID)
	suite.NoError(err, "unable to get pull request object")

	result, err = service.Merge(
		pullRequest.ID,
		pullRequest.Version,
	)
	suite.NoError(err, "unable to get merge pull request result")
	suite.Equal(
		len(result.Errors),
		0,
		"should be able to merge pull request",
	)
	suite.Equal(
		"MERGED",
		result.State,
		"pull request should be in merged state",
	)

	return hook
}

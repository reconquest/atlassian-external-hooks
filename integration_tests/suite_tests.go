package main

import (
	"fmt"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
)

func (suite *Suite) TestProjectHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	suite.testPreReceive(context, repository)
	suite.testPostReceive(context, repository)

	pullRequest := suite.CreateRandomPullRequest(project, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) TestRepositoryHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))

	var (
		project     = suite.CreateRandomProject()
		repository  = suite.CreateRandomRepository(project)
		pullRequest = suite.CreateRandomPullRequest(project, repository)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testPreReceive(context, repository)
	suite.testPostReceive(context, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) TestPersonalRepositoriesHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))

	project := &stash.Project{
		Key: "~admin",
	}

	var (
		repository  = suite.CreateRandomRepository(project)
		pullRequest = suite.CreateRandomPullRequest(project, repository)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.testPreReceive(context, repository)
	suite.testPostReceive(context, repository)
	suite.testBasicMergeCheckScenario(context, repository, pullRequest)
}

func (suite *Suite) testBug_ProjectEnabledRepositoryDisabledHooks_Reproduced(
	params TestParams,
	context *external_hooks.Context,
	project *stash.Project,
	repository *stash.Repository,
) {
	addon := suite.InstallAddon(params["addon_reproduced"].(Addon))

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
	addon := suite.InstallAddon(params["addon_fixed"].(Addon))

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
	suite.InstallAddon(params["addon"].(Addon))

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
		external_hooks.NewSettings().
			UseSafePath(true).
			WithExecutable(`pre.fail.sh`),
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
		external_hooks.NewSettings().
			UseSafePath(true).
			WithExecutable(`post.fail.sh`),
		text(
			`#!/bin/bash`,
			`echo YYY`,
			`exit 1`,
		),
	)

	Testcase_PushOutputsMessages(suite, repo, `YYY`)

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

	Testcase_PushOutputsMessages(suite, repo, `YYY`)

	err = post.Disable()
	suite.NoError(err, "unable to disable post-receive hook")

	Testcase_PushDoesNotOutputMessage(suite, repo, `YYY`)
}

func (suite *Suite) testPreReceive(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testPreReceiveHookInput(context, repository)
	suite.testPreReceiveHookReject(context, repository)
}

func (suite *Suite) testPreReceiveHookReject(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	hook := suite.CreateSamplePreReceiveHook_FailWithMessage(context, `XXX`)

	Testcase_PushRejected(suite, repository, `XXX`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `XXX`)
}

func (suite *Suite) testPostReceive(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testPostReceiveHookInput(context, repository)
	suite.testPostReceiveOutputsMessage(context, repository)
}

func (suite *Suite) testPostReceiveOutputsMessage(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	hook := suite.CreateSamplePostReceiveHook_FailWithMessage(context, `YYY`)

	Testcase_PushOutputsMessages(suite, repository, `YYY`)

	err := hook.Disable()
	suite.NoError(err, "unable to disable hook")

	Testcase_PushDoesNotOutputMessage(suite, repository, `YYY`)
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

func (suite *Suite) testPreReceiveHookInput(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testReceiveHookInput_Common(context, repository)
	suite.testHookInput_Env_BB_HOOK_TYPE(context, repository, `PRE`)
}

func (suite *Suite) testPostReceiveHookInput(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testReceiveHookInput_Common(context, repository)
	suite.testHookInput_Env_BB_HOOK_TYPE(context, repository, `POST`)
}

func (suite *Suite) testReceiveHookInput_Common(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testReceiveHookInput_Stdin(context, repository)
	suite.testReceiveHookInput_Args(context, repository)

	suite.testHookInput_Env_BB_HOOK_TRIGGER_ID(context, repository, `push`)
	suite.testHookInput_Env_BB_IS_DRY_RUN(context, repository, `false`)
	suite.testHookInput_Env_BB_REPO_IS_FORK(context, repository, `false`)
	suite.testHookInput_Env_BB_REPO_IS_PUBLIC(context, repository, `false`)

	suite.testHookInput_Env_BB_PROJECT_KEY(context, repository)
	suite.testHookInput_Env_BB_REPO_SLUG(context, repository)
	suite.testHookInput_Env_BB_BASE_URL(context, repository)
	suite.testHookInput_Env_BB_REPO_CLONE_SSH(context, repository)
	suite.testHookInput_Env_BB_REPO_CLONE_HTTP(context, repository)
	suite.testHookInput_Env_BB_USER_NAME(context, repository)
	suite.testHookInput_Env_BB_USER_DISPLAY_NAME(context, repository)
	suite.testHookInput_Env_BB_USER_EMAIL(context, repository)
	suite.testHookInput_Env_BB_USER_PERMISSION(context, repository)
}

func (suite *Suite) testReceiveHookInput_Stdin(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.CreateSamplePreReceiveHook_Debug(
		context,
		`cat`,
	)

	Testcase_PushOutputsRefInfo(suite, repository)
}

func (suite *Suite) testReceiveHookInput_Args(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.CreateSamplePreReceiveHook_Debug(
		context,
		`printf "[%s]\n" "$@"`,
		"arg-1",
		"arg-2",
		"multi\nline",
	)

	Testcase_PushOutputsMessages(
		suite,
		repository,
		"[arg-1]",
		"[arg-2]",
		"[multi",
		"line]",
	)
}

func (suite *Suite) testHookInput_Env(
	context *external_hooks.Context,
	repository *stash.Repository,
	name string,
	value string,
) {
	suite.CreateSamplePreReceiveHook_Debug(
		context,
		fmt.Sprintf(`echo [$%s]`, name),
	)

	Testcase_PushOutputsMessages(
		suite,
		repository,
		fmt.Sprintf("[%s]", value),
	)
}

func (suite *Suite) testHookInput_Env_BB_HOOK_TRIGGER_ID(
	context *external_hooks.Context,
	repository *stash.Repository,
	value string,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_HOOK_TRIGGER_ID",
		value,
	)
}

func (suite *Suite) testHookInput_Env_BB_HOOK_TYPE(
	context *external_hooks.Context,
	repository *stash.Repository,
	value string,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_HOOK_TYPE",
		value,
	)
}

func (suite *Suite) testHookInput_Env_BB_IS_DRY_RUN(
	context *external_hooks.Context,
	repository *stash.Repository,
	value string,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_IS_DRY_RUN",
		value,
	)
}

func (suite *Suite) testHookInput_Env_BB_PROJECT_KEY(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_PROJECT_KEY",
		repository.Project.Key,
	)
}

func (suite *Suite) testHookInput_Env_BB_REPO_SLUG(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_REPO_SLUG",
		repository.Slug,
	)
}

func (suite *Suite) testHookInput_Env_BB_REPO_IS_FORK(
	context *external_hooks.Context,
	repository *stash.Repository,
	value string,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_REPO_IS_FORK",
		value,
	)
}

func (suite *Suite) testHookInput_Env_BB_REPO_IS_PUBLIC(
	context *external_hooks.Context,
	repository *stash.Repository,
	value string,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_REPO_IS_PUBLIC",
		value,
	)
}

func (suite *Suite) testHookInput_Env_BB_BASE_URL(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_BASE_URL",
		suite.Bitbucket().GetURI(""),
	)
}

func (suite *Suite) testHookInput_Env_BB_REPO_CLONE_SSH(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_REPO_CLONE_SSH",
		suite.Bitbucket().GetClonePathSSH(
			repository.Project.Key,
			repository.Slug,
		),
	)
}

func (suite *Suite) testHookInput_Env_BB_REPO_CLONE_HTTP(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_REPO_CLONE_HTTP",
		suite.Bitbucket().GetClonePathHTTP(
			repository.Project.Key,
			repository.Slug,
		),
	)
}

func (suite *Suite) testHookInput_Env_BB_USER_NAME(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_USER_NAME",
		suite.Bitbucket().GetOpts().AdminUser,
	)
}

func (suite *Suite) testHookInput_Env_BB_USER_EMAIL(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_USER_EMAIL",
		suite.Bitbucket().GetOpts().AdminEmail,
	)
}

func (suite *Suite) testHookInput_Env_BB_USER_DISPLAY_NAME(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_USER_DISPLAY_NAME",
		suite.Bitbucket().GetOpts().AdminUser,
	)
}

func (suite *Suite) testHookInput_Env_BB_USER_PERMISSION(
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	suite.testHookInput_Env(
		context,
		repository,
		"BB_USER_PERMISSION",
		"SYS_ADMIN",
	)
}

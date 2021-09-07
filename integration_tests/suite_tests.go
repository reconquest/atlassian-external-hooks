package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/users"
	"github.com/reconquest/cog"
	"github.com/reconquest/pkg/log"
)

func (suite *Suite) TestProjectHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	log := log.NewChildWithPrefix(
		fmt.Sprintf("{test: project hooks} %s", project.Key),
	)

	suite.testPreReceive(log, context, repository)
	suite.testPostReceive(log, context, repository)
	suite.testMergeCheck(log, context, repository)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestRepositoryHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	log := log.NewChildWithPrefix(
		fmt.Sprintf(
			"{test: repository hooks} %s / %s",
			project.Key,
			repository.Slug,
		),
	)

	suite.testPreReceive(log, context, repository)
	suite.testPostReceive(log, context, repository)
	suite.testMergeCheck(log, context, repository)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestPersonalRepositoriesHooks(params TestParams) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	project := &stash.Project{
		Key: "~admin",
	}

	repository := suite.CreateRandomRepository(project)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	log := log.NewChildWithPrefix(
		fmt.Sprintf(
			"{test: personal repositories hooks} %s / %s",
			project.Key,
			repository.Slug,
		),
	)

	suite.testPreReceive(log, context, repository)
	suite.testPostReceive(log, context, repository)
	suite.testMergeCheck(log, context, repository)

	suite.DetectHookScriptsLeak()
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

		project := suite.CreateRandomProject()

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
			project,
			cases.personal.repo,
			context,
		)
	}

	suite.UseBitbucket(params["bitbucket_to"].(string))
	suite.RecordHookScripts()

	suite.testBitbucketUpgrade_After(
		cases.public.repo,
		cases.public.pre,
		cases.public.post,
	)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testBitbucketUpgrade_Before(
	project *stash.Project,
	repo *stash.Repository,
	context *external_hooks.Context,
) (*external_hooks.Hook, *external_hooks.Hook) {
	pre := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX`,
	)

	Assert_PushRejected(suite, repo, `XXX`)

	suite.DisableHook(pre)

	Assert_PushDoesNotOutputMessages(suite, repo, `XXX`)

	post := suite.ConfigureSampleHook_FailWithMessage(
		context.PostReceive(),
		HookOptions{WaitHookScripts: true},
		`YYY`,
	)

	Assert_PushOutputsMessages(suite, repo, `YYY`)

	suite.DisableHook(post)

	Assert_PushDoesNotOutputMessages(suite, repo, `YYY`)

	err := pre.Enable()
	suite.NoError(err, "unable to enable pre-receive hook")

	err = post.Enable()
	suite.NoError(err, "unable to enable post-receive hook")

	return pre, post
}

func (suite *Suite) testBitbucketUpgrade_After(
	repo *stash.Repository,
	pre, post *external_hooks.Hook,
) {
	pre.BitbucketURI = suite.Bitbucket().GetConnectorURI(users.USER_ADMIN)
	post.BitbucketURI = suite.Bitbucket().GetConnectorURI(users.USER_ADMIN)

	Assert_PushRejected(suite, repo, `XXX`)

	suite.DisableHook(pre)

	Assert_PushOutputsMessages(suite, repo, `YYY`)

	suite.DisableHook(post)

	Assert_PushDoesNotOutputMessages(suite, repo, `YYY`)
}

func (suite *Suite) TestProjectHooks_DoNotCreateDisabledHooks(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key)

	preReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX`,
	)

	err := preReceive.Disable()
	suite.NoError(err, "pre-receive hook should be disabled")

	postReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PostReceive(),
		HookOptions{WaitHookScripts: true},
		`YYY`,
	)

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)
	Assert_PushOutputsMessages(suite, repository, `YYY`)

	err = postReceive.Disable()
	suite.NoError(err)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) TestHookScriptsLeak_NoLeakAfterRepositoryDelete(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	var (
		project    = suite.CreateRandomProject()
		repository = suite.CreateRandomRepository(project)
	)

	context := suite.ExternalHooks().OnProject(project.Key).
		OnRepository(repository.Slug)

	suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		HookOptions{WaitHookScripts: true},
		`XXX`,
	)

	waiter := suite.Bitbucket().WaitLogEntry(func(line string) bool {
		return strings.Contains(
			line,
			"Successfully deleted repository directory",
		)
	})

	err := suite.Bitbucket().Repositories(project.Key).Remove(repository.Slug)
	suite.NoError(err, "unable to remove repository")

	waiter.Wait()

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testPreReceive(
	log *cog.Logger,
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	log.Infof(nil, "> testing pre-receive hooks")

	hook := context.PreReceive()

	tester := NewHookTester(log, hook, suite, repository)

	suite.testPreReceiveHook_Input(tester)
	suite.testPreReceiveHook_Veto(tester)

	suite.DisableHook(hook)
}

func (suite *Suite) testPostReceive(
	log *cog.Logger,
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	log.Infof(nil, "> testing post-receive hooks")

	hook := context.PostReceive()

	tester := NewHookTester(log, hook, suite, repository)

	suite.testPostReceiveHook_Input(tester)
	suite.testPostReceiveHook_Output(tester)
	suite.testPostReceiveHook_AfterMerge(tester)

	suite.DisableHook(hook)
}

func (suite *Suite) testPreReceiveHook_Veto(tester *HookTester) {
	tester.TestEnableDisable(
		Assert_PushRejected,
		Assert_PushDoesNotOutputMessages,
	)
}

func (suite *Suite) testPostReceiveHook_Output(
	tester *HookTester,
) {
	tester.TestEnableDisable(
		Assert_PushOutputsMessages,
		Assert_PushDoesNotOutputMessages,
	)
}

func (suite *Suite) testPostReceiveHook_AfterMerge(
	tester *HookTester,
) {
	name := "/tmp/" + fmt.Sprint(time.Now().UnixNano())
	tester.suite.ConfigureSampleHook(
		tester.hook,
		HookOptions{WaitHookScripts: true},
		string(text(
			fmt.Sprintf(`echo 1 > `+name),
		)),
	)

	pullRequest := suite.CreateRandomPullRequest(
		&tester.repository.Project,
		tester.repository,
	)

	Assert_MergeCheckPassed(pullRequest, suite, tester.repository)

	_, err := suite.Bitbucket().Instance.ReadFile(name)
	suite.NoError(err, "should have file")
}

func (suite *Suite) testMergeCheck(
	log *cog.Logger,
	context *external_hooks.Context,
	repository *stash.Repository,
) {
	log.Infof(nil, "> testing merge check")

	var (
		hook   = context.MergeCheck()
		tester = NewHookTester(log, hook, suite, repository).
			WithExitCode(1)
		pullRequest = suite.CreateRandomPullRequest(
			&repository.Project,
			repository,
		)
	)

	suite.testMergeCheck_Input(tester, pullRequest)
	suite.testMergeCheck_Veto(tester, pullRequest)

	suite.DisableHook(hook)
}

func (suite *Suite) testMergeCheck_Veto(
	tester *HookTester,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnableDisable(
		func(suite *Suite, repository *stash.Repository, messages ...string) {
			Assert_MergeCheckOutputsMessages(
				pullRequest, suite, repository, messages...,
			)
		},
		func(suite *Suite, repository *stash.Repository, messages ...string) {
			Assert_MergeCheckPassed(pullRequest, suite, repository)
		},
	)
}

func (suite *Suite) testPreReceiveHook_Input(tester *HookTester) {
	suite.testReceiveHook_Input_Common(tester)
	tester.TestEnv_BB_HOOK_TYPE(Assert_PushOutputsMessages, `PRE`)
}

func (suite *Suite) testPostReceiveHook_Input(
	tester *HookTester,
) {
	suite.testReceiveHook_Input_Common(tester)
	tester.TestEnv_BB_HOOK_TYPE(Assert_PushOutputsMessages, `POST`)
}

func (suite *Suite) testMergeCheck_Input(
	tester *HookTester,
	pullRequest *stash.PullRequest,
) {
	suite.testMergeCheck_Input_Common(tester, pullRequest)

	assert := AssertWithPullRequest(
		pullRequest,
		Assert_MergeCheckOutputsMessages,
	)

	tester.TestEnv_BB_HOOK_TYPE(assert, `PRE`)
	tester.TestEnv_BB_FROM_PROJECT_KEY(assert, pullRequest)
	tester.TestEnv_BB_FROM_REPO_SLUG(assert, pullRequest)
	tester.TestEnv_BB_FROM_REPO_IS_FORK(assert, `false`)
	tester.TestEnv_BB_FROM_REPO_IS_PUBLIC(assert, `false`)
	tester.TestEnv_BB_FROM_REF(assert, pullRequest)
	tester.TestEnv_BB_FROM_HASH(assert, pullRequest)
	tester.TestEnv_BB_FROM_REPO_CLONE_HTTP(assert, pullRequest)
	tester.TestEnv_BB_FROM_REPO_CLONE_SSH(assert, pullRequest)
	tester.TestEnv_BB_MERGE_IS_CROSS_REPO(assert, `false`)
	tester.TestEnv_BB_TO_REF(assert, pullRequest)
	tester.TestEnv_BB_TO_HASH(assert, pullRequest)
	tester.TestEnv_BB_MERGE_STRATEGY_ID(assert, `no-ff`)
	tester.TestEnv_BB_PULL_REQUEST_ID(assert, pullRequest)
	tester.TestEnv_BB_PULL_REQUEST_AUTHOR_USER_NAME(assert, pullRequest)
	tester.TestEnv_BB_PULL_REQUEST_AUTHOR_USER_DISPLAY_NAME(assert, pullRequest)
	tester.TestEnv_BB_PULL_REQUEST_AUTHOR_USER_EMAIL(assert, pullRequest)
	tester.TestEnv_BB_PULL_REQUEST_AUTHOR_USER_PERMISION(assert, `SYS_ADMIN`)
}

func (suite *Suite) testReceiveHook_Input_Common(
	tester *HookTester,
) {
	tester.TestStdin(Assert_PushOutputsRefInfo)
	tester.TestArgs(Assert_PushOutputsMessages)
	suite.testHookInput_EnvCommon(tester, Assert_PushOutputsMessages)
	tester.TestEnv_BB_HOOK_TRIGGER_ID(Assert_PushOutputsMessages, `push`)
}

func (suite *Suite) testHookInput_EnvCommon(
	tester *HookTester,
	assert HookTesterAssert,
) {
	tester.TestEnv_BB_IS_DRY_RUN(assert, `false`)
	tester.TestEnv_BB_REPO_IS_FORK(assert, `false`)
	tester.TestEnv_BB_REPO_IS_PUBLIC(assert, `false`)
	tester.TestEnv_BB_PROJECT_KEY(assert)
	tester.TestEnv_BB_REPO_SLUG(assert)
	tester.TestEnv_BB_BASE_URL(assert)
	tester.TestEnv_BB_REPO_CLONE_SSH(assert)
	tester.TestEnv_BB_REPO_CLONE_HTTP(assert)
	tester.TestEnv_BB_USER_NAME(assert)
	tester.TestEnv_BB_USER_DISPLAY_NAME(assert)
	tester.TestEnv_BB_USER_EMAIL(assert)
	tester.TestEnv_BB_USER_PERMISSION(assert)
}

func (suite *Suite) testMergeCheck_Input_Common(
	tester *HookTester,
	pullRequest *stash.PullRequest,
) {
	tester.TestStdin(
		func(suite *Suite, repository *stash.Repository, _ ...string) {
			// Since BB uses non-fast-forward merge strategy by default,
			// merge check script will receive merge commit SHA to stdin
			// which is unknown at the time of test, so we need to retrieve
			// it directly in test script.
			Assert_MergeCheck_Callback(
				pullRequest,
				suite, repository,
				func(reply string) {
					lines := strings.Split(strings.TrimSpace(reply), "\n")

					suite.GreaterOrEqual(
						len(lines), 2,
						"output from merge hook must contain at least 2 lines",
					)

					suite.Equal(
						fmt.Sprintf(
							"%s %s %s",
							pullRequest.ToRef.LatestCommit,
							lines[0],
							pullRequest.ToRef.ID,
						),
						lines[1],
					)
				},
			)
		},

		`echo $BB_MERGE_HASH`,
	)

	tester.TestArgs(
		AssertWithPullRequest(pullRequest, Assert_MergeCheckOutputsMessages),
	)

	tester.TestEnv_BB_HOOK_TRIGGER_ID(
		AssertWithPullRequest(pullRequest, Assert_MergeCheckOutputsMessages),
		`pull-request-merge`,
	)

	suite.testHookInput_EnvCommon(
		tester,
		AssertWithPullRequest(pullRequest, Assert_MergeCheckOutputsMessages),
	)
}

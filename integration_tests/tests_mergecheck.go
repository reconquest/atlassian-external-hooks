package main

import (
	"fmt"
	"strings"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/cog"
)

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
	suite.WaitExternalHookDisabled(hook)
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

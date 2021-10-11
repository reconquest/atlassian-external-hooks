package main

import (
	"fmt"
	"time"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/cog"
)

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

	suite.DisableHook(hook, HookOptions{
		// should be already disabled at this moment
		WaitHookScripts: false,
	})
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

func (suite *Suite) testPostReceiveHook_Input(
	tester *HookTester,
) {
	suite.testReceiveHook_Input_Common(tester)
	tester.TestEnv_BB_HOOK_TYPE(Assert_PushOutputsMessages, `POST`)
}

package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/cog"
)

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

func (suite *Suite) testPreReceiveHook_Veto(tester *HookTester) {
	tester.TestEnableDisable(
		Assert_PushRejected,
		Assert_PushDoesNotOutputMessages,
	)
}

func (suite *Suite) testPreReceiveHook_Input(tester *HookTester) {
	suite.testReceiveHook_Input_Common(tester)
	tester.TestEnv_BB_HOOK_TYPE(Assert_PushOutputsMessages, `PRE`)
}

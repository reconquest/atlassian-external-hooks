package main

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

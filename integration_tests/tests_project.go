package main

import (
	"fmt"

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
		`XXX`,
	)

	suite.WaitExternalHookEnabled(context.PreReceive())

	err := preReceive.Disable()
	suite.NoError(err, "pre-receive hook should be disabled")

	suite.WaitExternalHookDisabled(preReceive)

	postReceive := suite.ConfigureSampleHook_FailWithMessage(
		context.PostReceive(),
		`YYY`,
	)

	suite.WaitExternalHookEnabled(context.PostReceive())

	Assert_PushDoesNotOutputMessages(suite, repository, `XXX`)
	Assert_PushOutputsMessages(suite, repository, `YYY`)

	err = postReceive.Disable()
	suite.NoError(err)

	suite.DetectHookScriptsLeak()
}

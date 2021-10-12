package main

import "github.com/reconquest/pkg/log"

func (suite *Suite) TestGlobalHooks_PersonalRepositoriesFilter(
	params TestParams,
) {
	suite.UseBitbucket(params["bitbucket"].(string))
	suite.InstallAddon(params["addon"].(Addon))
	suite.RecordHookScripts()

	context := suite.ExternalHooks().OnGlobal()

	log := log.NewChildWithPrefix("{test: global hooks/personal repositories}")

	_ = context
	_ = log
	// suite.testGlobalHooks(log, context, project, repository)

	suite.DetectHookScriptsLeak()
}

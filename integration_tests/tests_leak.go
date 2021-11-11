package main

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
		`XXX`,
	)
	suite.WaitExternalHookEnabled(context.PreReceive())

	err := suite.Bitbucket().Repositories(project.Key).Remove(repository.Slug)
	suite.NoError(err, "remove repository")

	suite.WaitExternalHookUnconfigured()

	suite.DetectHookScriptsLeak()
}

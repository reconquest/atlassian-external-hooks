package main

import "strings"

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
	suite.NoError(err, "remove repository")

	waiter.Wait(suite.FailNow, "repository", "deleted")

	suite.DetectHookScriptsLeak()
}

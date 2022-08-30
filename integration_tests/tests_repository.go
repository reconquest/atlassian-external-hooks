package main

import (
	"fmt"

	"github.com/reconquest/pkg/log"
)

func (suite *Suite) TestRepositoryHooks(params TestParams) {
	suite.UseBitbucket(params.Bitbucket, params.Cluster)
	suite.InstallAddon(params.Addon)
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

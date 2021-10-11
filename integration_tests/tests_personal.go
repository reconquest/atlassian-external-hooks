package main

import (
	"fmt"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/pkg/log"
)

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

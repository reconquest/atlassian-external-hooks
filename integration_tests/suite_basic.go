package main

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/stretchr/testify/assert"
)

func SuiteBasic(run *runner.Runner, assert *assert.Assertions) {
	run.UseBitbucket("6.2.0")
	run.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := run.Bitbucket().Projects().Create(lojban.GetRandomID(4))
	assert.NoError(err, "unable to create project")

	repository, err := run.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	assert.NoError(err, "unable to create repository")

	Testcase_PreReceive_RejectPush(run, assert, project, repository)
	Testcase_PostReceive_OutputMessage(run, assert, project, repository)
}

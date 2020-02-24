package main

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/stretchr/testify/assert"
)

func SuiteProjectLevel(run *runner.Runner, assert *assert.Assertions) {
	run.UseBitbucket("6.2.0")
	run.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := run.Bitbucket().Projects().Create(lojban.GetRandomID(4))
	assert.NoError(err, "unable to create project")

	repository, err := run.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	assert.NoError(err, "unable to create repository")

	context := run.ExternalHooks().OnProject(project.Key)

	Testcase_PreReceive_RejectPush(run, assert, context, repository)
	Testcase_PostReceive_OutputMessage(run, assert, context, repository)
}

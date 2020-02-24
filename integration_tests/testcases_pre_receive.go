package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/stretchr/testify/assert"
)

func Testcase_PreReceive_RejectPush(
	run *runner.Runner,
	assert *assert.Assertions,
	project *stash.Project,
	repository *stash.Repository,
) {
	err := run.Bitbucket().WriteFile(
		"shared/external-hooks/fail.sh",
		text(
			`#!/bin/bash`,
			`echo XXX`,
			`exit 1`,
		),
		0777,
	)
	assert.NoError(err, "should be able to write hook script to container")

	addon := run.ExternalHooks()

	preReceive := addon.OnProject(project.Key).PreReceive(
		external_hooks.NewSettings().
			UseSafePath(true).
			WithExecutable("fail.sh"),
	)

	err = preReceive.Configure()
	assert.NoError(err, "should be able to configure pre-receive hook")

	err = preReceive.Enable()
	assert.NoError(err, "should be able to enable pre-receive hook")

	git := run.GitClone(repository)

	err = ioutil.WriteFile(
		filepath.Join(git.GetWorkDir(), "pre."+lojban.GetRandomID(6)),
		[]byte("file."+lojban.GetRandomID(10)),
		0666,
	)
	assert.NoError(err, "should be able to write file in git repo")

	err = git.Add(".")
	assert.NoError(err, "should be able to add file to git repo")

	err = git.Commit("commit." + lojban.GetRandomID(8))
	assert.NoError(err, "should be able to commit file to git repo")

	stdout, err := git.Push()
	assert.Error(err, "git push should fail")
	assert.Contains(
		string(stdout),
		"remote: external-pre-receive-hook declined",
		"pre-receive-hook should decline push",
	)
	assert.Contains(
		string(stdout),
		"remote rejected",
		"bitbicket should reject push",
	)
	assert.Contains(
		string(stdout),
		"remote: XXX",
		"git push message should contain output from pre-receive hook",
	)

	err = preReceive.Disable()
	assert.NoError(err, "should be able to disable pre-receive hook")

	stdout, err = git.Push()
	assert.NoError(err, "git push should succeed")
	assert.NotContains(
		string(stdout),
		"remote: external-pre-receive-hook declined",
		"pre-receive-hook should not decline push",
	)
}

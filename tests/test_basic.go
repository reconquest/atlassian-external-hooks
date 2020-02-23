package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/reconquest/atlassian-external-hooks/tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/tests/internal/lojban"
)

func TestBasic(suite *Suite) {
	suite.UseBitbucket("6.2.0")
	suite.InstallAddon("target/external-hooks-9.1.0.jar")

	project, err := suite.Bitbucket().Projects().Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	err = suite.Bitbucket().WriteFile(
		"shared/external-hooks/fail.sh",
		text(
			`#!/bin/bash`,
			`exit 1`,
		),
		0777,
	)
	suite.NoError(err, "unable to write hook executable file")

	addon := suite.ExternalHooks()

	preReceive := addon.OnProject(project.Key).PreReceive(
		external_hooks.NewSettings().
			UseSafePath(true).
			WithExecutable("fail.sh"),
	)

	err = preReceive.Configure()
	suite.NoError(err, "unable to configure addon")

	err = preReceive.Enable()
	suite.NoError(err, "unable to enable addon")

	git := suite.GitClone(repository)

	err = ioutil.WriteFile(
		filepath.Join(git.GetWorkDir(), "lyrics"),
		[]byte(`hello darkness my old friend`),
		0666,
	)
	suite.NoError(err, "unable to write test file")

	err = git.Add(".")
	suite.NoError(err, "unable to git add")

	err = git.Commit("lyrics added")
	suite.NoError(err, "unable to git commit")

	err = git.Push()
	suite.NoError(err, "unable to git push")
}

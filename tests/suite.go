package main

import (
	"github.com/reconquest/atlassian-external-hooks/tests/internal/bitbucket"
	"github.com/reconquest/karma-go"
	"github.com/stretchr/testify/assert"
)

type Suite struct {
	*assert.Assertions

	testcases []Testcase

	run struct {
		dir       string
		container string
		bitbucket *bitbucket.Bitbucket
	}
}

func NewSuite() *Suite {
	return &Suite{
		Assertions: assert.New(Testing{}),
	}
}

func (suite *Suite) UseBitbucket(version string) {
	var err error

	suite.run.bitbucket, err = bitbucket.Start(
		version,
		bitbucket.StartOpts{
			ContainerID: string(suite.run.container),
		},
	)
	suite.NoError(err, "unable to start bitbucket container")

	err = suite.run.bitbucket.Configure(bitbucket.ConfigureOpts{
		License: BITBUCKET_DC_LICENSE_3H,
	})

	suite.NoError(err, "unable configure bitbucket")
}

func (suite *Suite) InstallAddon(path string) {
	addon, err := suite.run.bitbucket.Addons().Install(path)
	suite.NoError(err, "unable to install addon")

	err = suite.run.bitbucket.Addons().SetLicense(addon, ADDON_LICENSE_3H)
	suite.NoError(err, "unable to set addon license")
}

func (suite *Suite) Testcase(testcase Testcase) {
	suite.testcases = append(suite.testcases, testcase)
}

func (suite *Suite) Bitbucket() *bitbucket.Bitbucket {
	return suite.run.bitbucket
}

func (suite *Suite) Cleanup() error {
	err := suite.run.bitbucket.Stop()
	if err != nil {
		return karma.Format(
			err,
			"unable to stop bitbucket",
		)
	}

	err = suite.run.bitbucket.RemoveContainer()
	if err != nil {
		return karma.Format(
			err,
			"unable to remove bitbucket container",
		)
	}

	err = suite.run.bitbucket.RemoveVolume()
	if err != nil {
		return karma.Format(
			err,
			"unable to remove bitbucket volume",
		)
	}

	return nil
}

package runner

import (
	"os"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Suite func(*Runner, *assert.Assertions)

type Runner struct {
	assert *assert.Assertions

	suites []Suite

	run struct {
		dir       string
		container string
		bitbucket *bitbucket.Bitbucket
	}
}

func New() *Runner {
	var runner Runner

	runner.assert = assert.New(&runner)

	return &runner
}

func (runner *Runner) UseBitbucket(version string) {
	var err error

	runner.run.bitbucket, err = bitbucket.Start(
		version,
		bitbucket.StartOpts{
			ContainerID: string(runner.run.container),
		},
	)
	runner.assert.NoError(err, "unable to start bitbucket container")

	err = runner.run.bitbucket.Configure(bitbucket.ConfigureOpts{
		License: BITBUCKET_DC_LICENSE_3H,
	})

	runner.assert.NoError(err, "unable configure bitbucket")
}

func (runner *Runner) InstallAddon(path string) {
	addon, err := runner.run.bitbucket.Addons().Install(path)
	runner.assert.NoError(err, "unable to install addon")

	err = runner.run.bitbucket.Addons().SetLicense(addon, ADDON_LICENSE_3H)
	runner.assert.NoError(err, "unable to set addon license")
}

func (runner *Runner) Suite(suite Suite) {
	runner.suites = append(runner.suites, suite)
}

func (runner *Runner) Bitbucket() *bitbucket.Bitbucket {
	return runner.run.bitbucket
}

func (runner *Runner) Cleanup() error {
	err := runner.run.bitbucket.Stop()
	if err != nil {
		return karma.Format(
			err,
			"unable to stop bitbucket",
		)
	}

	err = runner.run.bitbucket.RemoveContainer()
	if err != nil {
		return karma.Format(
			err,
			"unable to remove bitbucket container",
		)
	}

	err = runner.run.bitbucket.RemoveVolume()
	if err != nil {
		return karma.Format(
			err,
			"unable to remove bitbucket volume",
		)
	}

	return nil
}

func (runner *Runner) Errorf(format string, args ...interface{}) {
	log.Errorf(nil, "<testify> assertion failed:"+format, args...)
	log.Infof(
		karma.
			Describe("work_dir", runner.run.dir).
			Describe("container", runner.run.container).
			Describe("volume", runner.run.bitbucket.GetVolume()),
		"following run resources were kept",
	)

	os.Exit(1)
}

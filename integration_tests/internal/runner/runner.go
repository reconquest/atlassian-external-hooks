package runner

import (
	"os"

	"github.com/coreos/go-semver/semver"
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

	if runner.run.bitbucket != nil {
		var (
			running   = semver.New(runner.run.bitbucket.GetVersion())
			requested = semver.New(version)
		)

		if !running.Equal(*requested) {
			if running.Compare(*requested) == -1 {
				log.Infof(
					nil,
					"upgrading bitbucket: %s -> %s",
					running,
					requested,
				)

				volume := runner.run.bitbucket.GetVolume()

				err := runner.run.bitbucket.Stop()
				runner.assert.NoError(err, "unable to stop bitbucket")

				runner.run.bitbucket, err = bitbucket.Volume(volume).Start(
					version,
					bitbucket.StartOpts{},
				)
				runner.assert.NoError(
					err,
					"unable to upgrade bitbucket container",
				)
			} else {
				runner.assert.FailNowf(
					"unable to change bitbucket version",
					"bitbucket instance cannot be downgraded: %s -> %s",
					running,
					requested,
				)
			}
		}
	} else {
		runner.run.bitbucket, err = bitbucket.Start(
			version,
			bitbucket.StartOpts{
				ContainerID: string(runner.run.container),
			},
		)
		runner.assert.NoError(err, "unable to start bitbucket container")
	}

	runner.run.container = runner.run.bitbucket.GetContainerID()

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

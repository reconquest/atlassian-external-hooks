package runner

import (
	"errors"
	"os"
	"runtime"
	"strings"

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
		dir            string
		container      string
		bitbucket      *bitbucket.Bitbucket
		cleanupPrepare func()
		gotBitbucket   chan struct{}
	}
}

func New(cleanupPrepare func()) *Runner {
	var runner Runner

	runner.assert = assert.New(&runner)
	runner.run.gotBitbucket = make(chan struct{})
	runner.run.cleanupPrepare = cleanupPrepare

	return &runner
}

func (runner *Runner) WaitBitbucket() *bitbucket.Bitbucket {
	<-runner.run.gotBitbucket
	return runner.run.bitbucket
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
					"{bitbucket} upgrading: %s -> %s",
					running,
					requested,
				)

				volume := runner.run.bitbucket.GetVolume()

				err := runner.run.bitbucket.Stop()
				runner.assert.NoError(err, "unable to stop bitbucket")

				err = runner.run.bitbucket.RemoveContainer()
				runner.assert.NoError(
					err,
					"unable to remove previous container",
				)

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
	if err == nil {
		select {
		case runner.run.gotBitbucket <- struct{}{}:
		default:
		}
	}

	runner.run.cleanupPrepare()

	runner.assert.NoError(err, "unable configure bitbucket")
}

func (runner *Runner) InstallAddon(version string, path string) string {
	key, err := runner.run.bitbucket.Addons().Install(path)
	runner.assert.NoError(err, "unable to install addon")

	addon, err := runner.run.bitbucket.Addons().Get(key)
	runner.assert.NoError(err, "unable to get addon information")

	if addon.Version != version {
		log.Debugf(
			nil,
			"{add-on} version downgrade requested: %s -> %s",
			addon.Version,
			version,
		)

		err := runner.run.bitbucket.Addons().Uninstall(key)
		runner.assert.NoError(err, "unable to uninstall add-on for downgrade")

		_, err = runner.run.bitbucket.Addons().Install(path)
		runner.assert.NoError(err, "unable to install addon")
	}

	err = runner.run.bitbucket.Addons().SetLicense(key, ADDON_LICENSE_3H)
	runner.assert.NoError(err, "unable to set addon license")

	return key
}

func (runner *Runner) UninstallAddon(key string) {
	err := runner.run.bitbucket.Addons().Uninstall(key)
	runner.assert.NoError(err, "unable to install addon")
}

func (runner *Runner) Suite(suite Suite) {
	runner.suites = append(runner.suites, suite)
}

func (runner *Runner) Bitbucket() *bitbucket.Bitbucket {
	return runner.run.bitbucket
}

func (runner *Runner) Cleanup() error {
	log.Infof(
		karma.
			Describe("container", runner.run.bitbucket.GetContainerID()).
			Describe("volume", runner.run.bitbucket.GetVolume()),
		"{bitbucket} cleaning up resources",
	)

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
	err := errors.New("{testify} assertion failed")
	for i := 1; i < 10; i++ {
		fn := getFrame(i).Function
		if strings.Contains(fn, ".Test") || strings.Contains(fn, ".test") {
			err = karma.Describe("testcase", fn).Reason(err)
			break
		}
	}

	for i := 1; i < 10; i++ {
		fn := getFrame(i).Function
		if strings.Contains(fn, "(*Suite).Test") {
			err = karma.Describe("suite", fn).Reason(err)
			break
		}
	}

	log.Errorf(err, format, args...)

	volume := ""
	if runner.run.bitbucket != nil {
		volume = runner.run.bitbucket.GetVolume()
	}

	log.Infof(
		karma.
			Describe("work_dir", runner.run.dir).
			Describe("container", runner.run.container).
			Describe("volume", volume),
		"{run} following run resources were kept",
	)

	os.Exit(1)
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

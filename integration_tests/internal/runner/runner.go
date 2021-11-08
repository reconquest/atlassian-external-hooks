package runner

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/database"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Suite struct {
	Size int
	Run  func(*Runner, *assert.Assertions)
}

type Runner struct {
	assert *assert.Assertions

	suites []Suite

	database string

	run struct {
		workdir           string
		container         string
		database          string
		instanceBitbucket *bitbucket.Bitbucket
		instanceDatabase  database.Database
		cleanupPrepare    func()
		gotBitbucket      chan struct{}
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
	return runner.run.instanceBitbucket
}

func (runner *Runner) useDatabase(id string) {
	if runner.run.instanceDatabase == nil {
		database, err := database.Start(runner.run.database, id)
		runner.assert.NoError(err, "start database: %q", runner.run.database)

		runner.run.instanceDatabase = database
	}

	execution := exec.New("docker", "network", "inspect", id)
	err := execution.Run()
	if err != nil && !lexec.IsExitStatus(err) {
		runner.assert.NoError(err, "docker network inspect")
	}

	if lexec.IsExitStatus(err) {
		execution := exec.New("docker", "network", "create", id)
		err := execution.Run()
		runner.assert.NoError(err, "docker network create")
	}

	runner.connect(id, runner.run.instanceDatabase.Container())
}

func (runner *Runner) connect(network string, container string) {
	err := exec.New(
		"docker", "network", "connect",
		network, runner.run.instanceDatabase.Container(),
	).Run()
	if err != nil {
		if strings.Contains(err.Error(), "already exists in network") {
			return
		}
	}

	runner.assert.NoError(err, "connect database to the docker network")
}

func (runner *Runner) UseBitbucket(version string) {
	var err error

	// the id is used as network domain, containers' prefix and volumes' prefix
	id := fmt.Sprintf("aeh-%s", lojban.GetRandomID(5))
	if runner.run.container != "" {
		id = strings.TrimSuffix(runner.run.container, "-bitbucket")
	}

	switch {
	case runner.run.instanceBitbucket != nil:
		runner.upgrade(runner.run.instanceBitbucket.ID(), version)

	case runner.run.container != "":
		runner.useDatabase(id)

		runner.run.instanceBitbucket, err = bitbucket.StartExisting(
			bitbucket.StartExistingOpts{
				Container: runner.run.container,
				RunOpts: bitbucket.RunOpts{
					Version:  version,
					Database: runner.run.instanceDatabase,
					Network:  id,
				},
			},
		)
		runner.assert.NoError(err, "start existing container")

	default:
		runner.useDatabase(id)

		runner.run.instanceBitbucket, err = bitbucket.StartNew(
			bitbucket.StartNewOpts{
				ID: string(id),
				RunOpts: bitbucket.RunOpts{
					Version:  version,
					Database: runner.run.instanceDatabase,
					Network:  id,
				},
			},
		)
		runner.assert.NoError(err, "start new bitbucket container")
	}

	runner.run.container = runner.run.instanceBitbucket.Container()

	err = runner.run.instanceBitbucket.Configure(bitbucket.ConfigureOpts{
		License: BITBUCKET_DC_LICENSE_3H,
	})
	if err == nil {
		runner.notifyGotBitbucket()
	}

	runner.run.cleanupPrepare()

	runner.assert.NoError(err, "unable configure bitbucket")
}

func (runner *Runner) upgrade(id string, version string) {
	var (
		running   = semver.New(runner.run.instanceBitbucket.Version())
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

			err := runner.run.instanceBitbucket.Stop()
			runner.assert.NoError(err, "stop bitbucket")

			err = runner.run.instanceBitbucket.RemoveContainer()
			runner.assert.NoError(err, "remove previous container")

			runner.run.instanceBitbucket, err = bitbucket.StartNew(
				bitbucket.StartNewOpts{
					ID: string(id),
					RunOpts: bitbucket.RunOpts{
						Version:  version,
						Database: runner.run.instanceDatabase,
						Network:  id,
					},
				},
			)
			runner.assert.NoError(
				err,
				"upgrade bitbucket container",
			)
		} else {
			runner.assert.FailNowf(
				"change bitbucket version",
				"bitbucket instance cannot be downgraded: %s -> %s",
				running,
				requested,
			)
		}
	}
}

func (runner *Runner) notifyGotBitbucket() {
	select {
	case runner.run.gotBitbucket <- struct{}{}:
	default:
	}
}

func (runner *Runner) InstallAddon(version string, path string) string {
	key, err := runner.run.instanceBitbucket.Addons().Install(path)
	runner.assert.NoError(err, "install addon")

	addon, err := runner.run.instanceBitbucket.Addons().Get(key)
	runner.assert.NoError(err, "get addon information")

	if addon.Version != version {
		log.Debugf(
			nil,
			"{add-on} version downgrade requested: %s -> %s",
			addon.Version,
			version,
		)

		err := runner.run.instanceBitbucket.Addons().Uninstall(key)
		runner.assert.NoError(err, "uninstall add-on for downgrade")

		_, err = runner.run.instanceBitbucket.Addons().Install(path)
		runner.assert.NoError(err, "install addon")
	}

	err = runner.run.instanceBitbucket.Addons().SetLicense(
		key,
		ADDON_LICENSE_3H,
	)
	runner.assert.NoError(err, "set addon license")

	return key
}

func (runner *Runner) UninstallAddon(key string) {
	err := runner.run.instanceBitbucket.Addons().Uninstall(key)
	runner.assert.NoError(err, "install addon")
}

func (runner *Runner) Suite(suite Suite) {
	runner.suites = append(runner.suites, suite)
}

func (runner *Runner) Bitbucket() *bitbucket.Bitbucket {
	return runner.run.instanceBitbucket
}

func (runner *Runner) Database() database.Database {
	return runner.run.instanceDatabase
}

func (runner *Runner) Cleanup() error {
	log.Infof(
		karma.
			Describe("container/database", runner.run.instanceDatabase.Container()).
			Describe("container/bitbucket", runner.run.instanceBitbucket.Container()).
			Describe("volume/bitbucket", runner.run.instanceBitbucket.VolumeData()).
			Describe("volume/bitbucket/lib-native", runner.run.instanceBitbucket.VolumeLibNative()).
			Describe("volume/database", runner.run.instanceDatabase.Volume()),
		"{bitbucket} cleaning up resources",
	)

	err := runner.run.instanceBitbucket.Stop()
	if err != nil {
		return karma.Format(
			err,
			"stop bitbucket",
		)
	}

	err = runner.run.instanceBitbucket.RemoveContainer()
	if err != nil {
		return karma.Format(
			err,
			"remove bitbucket container",
		)
	}

	err = runner.run.instanceBitbucket.RemoveVolumeData()
	if err != nil {
		return karma.Format(
			err,
			"remove bitbucket data volume",
		)
	}

	err = runner.run.instanceBitbucket.RemoveVolumeLibNative()
	if err != nil {
		return karma.Format(
			err,
			"remove bitbucket lib-native volume",
		)
	}

	err = runner.run.instanceDatabase.Stop()
	if err != nil {
		return karma.Format(err, "stop database")
	}

	err = runner.run.instanceDatabase.RemoveContainer()
	if err != nil {
		return karma.Format(err, "remove database container")
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

	volumeData := ""

	volumeLibNative := ""
	if runner.run.instanceBitbucket != nil {
		volumeLibNative = runner.run.instanceBitbucket.VolumeLibNative()
	}

	facts := karma.
		Describe("work_dir", runner.run.workdir)

	if runner.run.instanceBitbucket != nil {
		facts = facts.
			Describe("network", runner.run.instanceBitbucket.Network()).
			Describe("container/bitbucket", runner.run.container).
			Describe("volume/bitbucket", volumeData).
			Describe("volume/bitbucket/lib-native", volumeLibNative)
	}

	if runner.run.instanceDatabase != nil {
		facts = facts.
			Describe("container/database", runner.run.instanceDatabase.Container()).
			Describe("volume/database", runner.run.instanceDatabase.Volume())
	}

	log.Infof(
		facts,
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

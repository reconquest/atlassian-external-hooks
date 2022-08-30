package runner

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/cluster"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/database"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
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

	run struct {
		volumes      string
		workdir      string
		identifier   string
		databaseKind string
		cluster      *cluster.Cluster
		bitbucket    bitbucket.Bitbucket
		database     database.Database
		cleanup      func()
		ready        chan struct{}
	}
}

func New(volumes string, cleanup func()) *Runner {
	var runner Runner

	runner.assert = assert.New(&runner)
	runner.run.ready = make(chan struct{}, 1)
	runner.run.cleanup = cleanup
	runner.run.volumes = volumes

	return &runner
}

func (runner *Runner) WaitBitbucket() []bitbucket.Bitbucket {
	fmt.Fprintf(os.Stderr, "XXXXXX runner.go:54 WAIT READY\n")
	<-runner.run.ready
	return runner.Bitbuckets()
}

func (runner *Runner) useDatabase(id string) {
	if runner.run.database == nil {
		database, err := database.Start(runner.run.databaseKind, id)
		runner.assert.NoError(err, "start database: %q", runner.run.databaseKind)

		runner.run.database = database
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

	runner.connect(id, runner.run.database.Container())
}

func (runner *Runner) connect(network string, container string) {
	err := exec.New(
		"docker", "network", "connect",
		network, runner.run.database.Container(),
	).Run()
	if err != nil {
		if strings.Contains(err.Error(), "already exists in network") {
			return
		}
	}

	runner.assert.NoError(err, "connect database to the docker network")
}

func (runner *Runner) upgrade(id string, version string) {
	var (
		running   = semver.New(runner.run.bitbucket.Version())
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

			err := runner.run.bitbucket.Stop()
			runner.assert.NoError(err, "stop bitbucket")

			err = runner.run.bitbucket.RemoveContainer()
			runner.assert.NoError(err, "remove previous container")

			runner.run.bitbucket, err = bitbucket.StartNew(
				bitbucket.StartNewOpts{
					ID: string(id),
					RunOpts: bitbucket.RunOpts{
						Version:  version,
						Database: runner.run.database,
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

func (runner *Runner) InstallAddon(version string, path string) string {
	key, err := runner.Bitbucket().Addons().Install(path)
	runner.assert.NoError(err, "install addon")

	addon, err := runner.Bitbucket().Addons().Get(key)
	runner.assert.NoError(err, "get addon information")

	if addon.Version != version {
		log.Debugf(
			nil,
			"{add-on} version downgrade requested: %s -> %s",
			addon.Version,
			version,
		)

		err := runner.Bitbucket().Addons().Uninstall(key)
		runner.assert.NoError(err, "uninstall add-on for downgrade")

		_, err = runner.Bitbucket().Addons().Install(path)
		runner.assert.NoError(err, "install addon")
	}

	err = runner.Bitbucket().Addons().SetLicense(
		key,
		ADDON_LICENSE_3H,
	)
	runner.assert.NoError(err, "set addon license")

	return key
}

func (runner *Runner) UninstallAddon(key string) {
	err := runner.Bitbucket().Addons().Uninstall(key)
	runner.assert.NoError(err, "install addon")
}

func (runner *Runner) Suite(suite Suite) {
	runner.suites = append(runner.suites, suite)
}

func (runner *Runner) ClusteringEnabled() bool {
	return runner.run.cluster != nil
}

func (runner *Runner) Bitbucket() bitbucket.Bitbucket {
	if runner.run.cluster != nil {
		return runner.run.cluster
	}

	return runner.run.bitbucket
}

func (runner *Runner) Database() database.Database {
	return runner.run.database
}

func (runner *Runner) Bitbuckets() []bitbucket.Bitbucket {
	if runner.run.cluster != nil {
		nodes := []bitbucket.Bitbucket{}
		for _, node := range runner.run.cluster.Nodes {
			nodes = append(nodes, node)
		}
		return nodes
	}

	if runner.run.bitbucket != nil {
		return []bitbucket.Bitbucket{runner.run.bitbucket}
	}

	return nil
}

func (runner *Runner) Cleanup() error {
	nodes := runner.Bitbuckets()
	for _, node := range nodes {
		log.Infof(
			karma.
				Describe(node.Container()+"/container", node.Container()).
				Describe(node.Container()+"/volume", node.VolumeData()).
				Describe("shared/database/container", runner.run.database.Container()).
				Describe("shared/database/volume", runner.run.database.Volume()),
			"{bitbucket} cleaning up resources",
		)

		err := node.Stop()
		if err != nil {
			return karma.Format(
				err,
				"stop bitbucket",
			)
		}

		err = node.RemoveContainer()
		if err != nil {
			return karma.Format(
				err,
				"remove bitbucket container",
			)
		}

		err = node.RemoveVolumes()
		if err != nil {
			return karma.Format(
				err,
				"remove bitbucket volumes",
			)
		}
	}

	err := runner.run.database.Stop()
	if err != nil {
		return karma.Format(err, "stop database")
	}

	err = runner.run.database.RemoveContainer()
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

	facts := karma.
		Describe("work_dir", runner.run.workdir)

	nodes := runner.Bitbuckets()
	if len(nodes) > 0 {
		facts = facts.Describe("shared/network", nodes[0].Network())

		for _, node := range nodes {
			facts = facts.Describe(node.Container()+"/container", node.Container()).
				Describe(node.Container()+"/volume", node.VolumeData())
		}
	}

	if runner.run.database != nil {
		facts = facts.
			Describe("shared/database/container", runner.run.database.Container()).
			Describe("shared/database/volume", runner.run.database.Volume())
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

package bitbucket

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/database"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

const (
	BITBUCKET_IMAGE    = "atlassian/bitbucket-server:%s"
	BITBUCKET_DATA_DIR = "/var/atlassian/application-data/bitbucket"
)

type StartExistingOpts struct {
	Container string

	RunOpts
}

type StartNewOpts struct {
	ID string

	RunOpts
}

type RunOpts struct {
	Version string
	// VolumeData string

	Database database.Database
	Network  string
	PortHTTP int
	PortSSH  int

	AdminUser     string
	AdminPassword string
}

type ConfigureOpts struct {
	License    string
	AdminEmail string
}

func StartNew(opts StartNewOpts) (*Bitbucket, error) {
	if opts.ID == "" {
		panic("opts.ID is empty")
	}

	instance := newInstance(opts.ID, ensureValidOpts(opts.RunOpts))
	instance.container = fmt.Sprintf("%s-bitbucket", opts.ID)

	log.Infof(
		karma.
			Describe("opts", opts),
		"{bitbucket %s} starting container",
		opts.Version,
	)

	err := instance.create()
	if err != nil {
		return nil, karma.Format(err, "create bitbucket container")
	}

	err = instance.connect()
	if err != nil {
		return nil, karma.Format(
			err,
			"connect to container",
		)
	}

	err = waitAndWatch(instance)
	if err != nil {
		return nil, err
	}

	return New(instance)
}

func StartExisting(opts StartExistingOpts) (*Bitbucket, error) {
	stdout, _, err := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f", "{{. | json}}",
		opts.Container,
	).NoStdLog().Output()
	if err != nil {
		return nil, karma.
			Describe("container", opts.Container).
			Format(
				err,
				"inspect container",
			)
	}

	var inspect struct {
		Config struct {
			Image string
		}

		Mounts []struct {
			Type        string
			Name        string
			Destination string
		}
	}

	err = json.Unmarshal(stdout, &inspect)
	if err != nil {
		return nil, karma.Format(
			err,
			"unmarshal inspect output",
		)
	}

	image := fmt.Sprintf(BITBUCKET_IMAGE, opts.Version)

	if image != inspect.Config.Image {
		return nil, karma.
			Describe("container", opts.Container).
			Describe("expected_image", image).
			Describe("running_image", inspect.Config.Image).
			Format(
				err,
				"existing container image mismatch",
			)
	}

	instance := newInstance(
		strings.TrimSuffix(opts.Container, "-bitbucket"),
		ensureValidOpts(opts.RunOpts),
	)

	instance.container = opts.Container

	log.Infof(
		karma.
			Describe("container", instance.container).
			Describe("opts", opts),
		"{bitbucket %s} re-using existing container",
		opts.Version,
	)

	err = instance.connect()
	if err != nil {
		return nil, karma.Format(
			err,
			"connect to container",
		)
	}

	err = waitAndWatch(instance)
	if err != nil {
		return nil, err
	}

	return New(instance)
}

func ensureValidOpts(opts RunOpts) RunOpts {
	if opts.PortHTTP == 0 {
		opts.PortHTTP = 7990
	}

	if opts.PortSSH == 0 {
		opts.PortSSH = 7999
	}

	if opts.AdminUser == "" {
		opts.AdminUser = "admin"
	}

	if opts.AdminPassword == "" {
		opts.AdminPassword = "admin"
	}

	if opts.Version == "" {
		panic("opts.Version is empty")
	}

	if opts.Network == "" {
		panic("opts.Network is empty")
	}

	if opts.Database == nil {
		panic("opts.Database is nil")
	}

	return opts
}

func newInstance(id string, opts RunOpts) *Instance {
	instance := &Instance{
		id:              id,
		version:         opts.Version,
		volumeData:      id + "-bitbucket-data",
		volumeLibNative: id + "-bitbucket-data-lib-native",
		database:        opts.Database,
		network:         opts.Network,
	}

	// this RunOpts should not be used to access the data, it's used only for
	// logging purposes as "the provided value when creating the instance"
	instance.opts.RunOpts = opts

	return instance
}

func waitAndWatch(instance *Instance) error {
	var err error

	instance.stacktraceLogs, err = instance.startLogReader(false)
	if err != nil {
		return karma.Format(err, "start log reader")
	}

	instance.testcaseLogs, err = instance.startLogReader(true)
	if err != nil {
		return karma.Format(err, "start log reader")
	}

	var message string

	for {
		status, err := instance.getStartupStatus()
		if err != nil {
			return karma.Format(
				err,
				"get container startup status",
			)
		}

		if status == nil {
			continue
		}

		if message != status.Progress.Message {
			log.Debugf(
				nil,
				"{bitbucket %s} setup: %3d%% %s | %s",
				instance.version,
				status.Progress.Percentage,
				strings.ToLower(status.State),
				status.Progress.Message,
			)

			message = status.Progress.Message
		}

		if status.State == "STARTED" {
			break
		}

		time.Sleep(time.Millisecond * 20)
	}

	return nil
}

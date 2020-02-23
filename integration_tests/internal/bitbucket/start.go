package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/karma-go"
)

const (
	BITBUCKET_IMAGE    = "atlassian/bitbucket-server:%s"
	BITBUCKET_DATA_DIR = "/var/atlassian/application-data/bitbucket"
)

type StartOpts struct {
	ContainerID string
	PortHTTP    int
	PortSSH     int

	AdminUser     string
	AdminPassword string
}

type ConfigureOpts struct {
	License    string
	AdminEmail string
}

func Start(version string, opts StartOpts) (*Bitbucket, error) {
	if opts.ContainerID != "" {
		return startExisting(version, opts)
	} else {
		return startNew(version, opts)
	}
}

func startNew(version string, opts StartOpts) (*Bitbucket, error) {
	return Volume(fmt.Sprintf("bitbucket-%s", lojban.GetRandomID(4))).Start(
		version,
		opts,
	)
}

func startExisting(version string, opts StartOpts) (*Bitbucket, error) {
	stdout, _, err := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f", "{{. | json}}",
		opts.ContainerID,
	).NoStdLog().Output()
	if err != nil {
		return nil, karma.
			Describe("container", opts.ContainerID).
			Format(
				err,
				"unable to inspect container",
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
			"unable to unmarshal inspect output",
		)
	}

	image := fmt.Sprintf(BITBUCKET_IMAGE, version)

	var volume string

	for _, mount := range inspect.Mounts {
		if mount.Destination == BITBUCKET_DATA_DIR {
			volume = mount.Name
			break
		}
	}

	if volume == "" {
		return nil, karma.
			Describe("container", opts.ContainerID).
			Format(
				err,
				"given container has no data volume",
			)
	}

	if image != inspect.Config.Image {
		return nil, karma.
			Describe("container", opts.ContainerID).
			Describe("expected_image", image).
			Describe("running_image", inspect.Config.Image).
			Format(
				err,
				"existing container image mismatch",
			)
	}

	return Volume(volume).Start(version, opts)
}

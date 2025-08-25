package mesh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/docker"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
)

const (
	MESH_IMAGE      = `atlassian/bitbucket-mesh`
	MESH_DATA_DIR   = `/var/atlassian/application-data/mesh`
	MESH_SHARED_DIR = "/var/atlassian/application-data/bitbucket"

	MESH_PROPERTIES = `# mesh.properties provided by external-hooks-test
hookscripts.gc.interval=1
hookscripts.gc.prune=1
mesh.logging.logger.com.atlassian.bitbucket.mesh=DEBUG
#grpc.server.ssl.cert-chain-path=` + MESH_DATA_DIR + `/config/ssl/cert.pem
#grpc.server.ssl.private-key-path=` + MESH_DATA_DIR + `/config/ssl/key.pem
`
)

var (
	errContainerNotFound = errors.New("container not found")
)

type Node struct {
	container string
	ip        string

	logs *docker.Logs

	volumes struct {
		data   string
		shared string
	}

	StartOpts
}

func (node *Node) IP() string {
	return node.ip
}

func (node *Node) Container() string {
	return node.container
}

type StartOpts struct {
	ID      string
	Replica int
	Volumes string
	Network string
	Version string
}

func Start(opts StartOpts) (*Node, error) {
	mesh := &Node{
		StartOpts: opts,
	}

	mesh.container = fmt.Sprintf("%s-mesh-%d", opts.ID, opts.Replica)

	mesh.volumes.data = filepath.Join(
		mesh.Volumes,
		fmt.Sprintf("%s-mesh-%d-data", opts.ID, opts.Replica),
	)

	mesh.volumes.shared = filepath.Join(mesh.Volumes, opts.ID+"-bitbucket-shared")

	err := mesh.inspect()
	switch {
	case err == errContainerNotFound:
		err := mesh.create(opts.Version)
		if err != nil {
			return nil, karma.Format(err, "create mesh node")
		}

		err = mesh.inspect()
		if err != nil {
			return nil, karma.Format(err, "inspect mesh node")
		}

	case err != nil:
		return nil, err
	}

	err = mesh.wait()
	if err != nil {
		return nil, karma.Format(err, "wait mesh node to become ready")
	}

	return mesh, nil
}

func (node *Node) Stop() error {
	err := exec.New(
		"docker",
		"kill",
		node.container,
	).Run()
	if err != nil {
		return karma.Format(
			err,
			"send docker stop",
		)
	}

	return exec.New("docker", "wait", node.container).Run()
}

func (node *Node) RemoveContainer() error {
	return exec.New(
		"docker",
		"rm", "-f",
		node.container,
	).Run()
}

func (node *Node) create(version string) error {
	// err := node.writeSSL()
	// if err != nil {
	// 	return karma.Format(err, "write ssl")
	// }

	err := node.writeProperties()
	if err != nil {
		return karma.Format(err, "write properties")
	}

	var initScript = []string{
		"set -euo pipefail",
		"apt update",
		"apt install -y git", // for whatever reason atlassian-provided mesh images come with incorrect git
		"cd " + MESH_DATA_DIR,
		"test -a mesh.mv.db && mv mesh.mv.db{,.bak}", // Workaround for Mesh H2 mandatory manual upgrade between versions
		"test -a mesh.trace.db && mv mesh.trace.db{,.bak}",
		"exec /opt/atlassian/mesh/bin/start-mesh.sh -fg", // exec is required to propagate INT signal from docker kill
	}

	execution := exec.New(
		"docker", "container", "create",
		"--network", node.Network,
		"--name", node.container,
		"-v", fmt.Sprintf("%s:%s", node.volumes.data, MESH_DATA_DIR),
		"-v", fmt.Sprintf(
			"%s:%s",
			node.volumes.shared,
			filepath.Join(MESH_SHARED_DIR, "shared"),
		),
		"--entrypoint", "/bin/bash",
		MESH_IMAGE+":"+version,
		"-c", strings.Join(initScript, ";"),
	)

	err = execution.Run()
	if err != nil {
		return karma.Format(
			err,
			"unable to create mesh container",
		)
	}

	return node.start()
}

func (node *Node) writeSSL() error {
	sslDir := filepath.Join(node.volumes.data, "config", "ssl")

	err := os.MkdirAll(sslDir, 0755)
	if err != nil {
		return karma.Format(
			err,
			"create data volume",
		)
	}

	execution := exec.New(
		"mkcert",
		"-cert-file", filepath.Join(sslDir, "cert.pem"),
		"-key-file", filepath.Join(sslDir, "key.pem"),
		"https://"+node.container+":7777",
	)

	err = execution.Run()
	if err != nil {
		return karma.Format(
			err,
			"unable to create ssl certificate",
		)
	}

	return nil
}

func (node *Node) writeProperties() error {
	err := os.MkdirAll(node.volumes.data, 0755)
	if err != nil {
		return karma.Format(
			err,
			"create data volume",
		)
	}

	path := filepath.Join(node.volumes.data, "mesh.properties")

	// Do not overwrite existing mesh config.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	err = os.WriteFile(
		path,
		[]byte(MESH_PROPERTIES),
		0644,
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to write mesh.properties",
		)
	}

	return nil
}

func (node *Node) start() error {
	execution := exec.New("docker", "container", "start", node.container)
	return execution.Run()
}

func (node *Node) wait() error {
	var err error
	node.logs, err = docker.ReadLogs(
		docker.ReadOpts{
			Container: node.container,
			Trace:     true,
			Tail:      1000,
			File:      MESH_DATA_DIR + "/log/atlassian-mesh.log",
		},
	)
	if err != nil {
		return karma.Format(err, "read logs")
	}

	waiter := docker.WaitLog(
		context.Background(),
		node.logs,
		func(line string) bool {
			if strings.Contains(line, "Ready to serve") {
				return true
			}

			return false
		},
		time.Second*30,
	)

	if !waiter.Await() {
		return errors.New("unable to wait for mesh node become ready")
	}

	return nil
}

func (node *Node) inspect() error {
	stdout, _, err := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f", "{{. | json}}",
		node.container,
	).NoStdLog().Output()
	if err != nil {
		if strings.Contains(err.Error(), "No such container:") {
			return errContainerNotFound
		}

		return karma.
			Describe("container", node.container).
			Format(
				err,
				"inspect container",
			)
	}

	var inspect struct {
		Config struct {
			Image string
		}

		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`

		Mounts []struct {
			Type        string
			Name        string
			Destination string
		}
	}

	err = json.Unmarshal(stdout, &inspect)
	if err != nil {
		return karma.Format(
			err,
			"unmarshal inspect output",
		)
	}

	ips := []string{}
	for _, network := range inspect.NetworkSettings.Networks {
		if network.IPAddress != "" {
			ips = append(ips, network.IPAddress)
		}
	}

	node.ip = ips[0]

	return nil
}

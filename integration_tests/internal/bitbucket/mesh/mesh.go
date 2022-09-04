package mesh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/docker"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
)

const (
	MESH_IMAGE    = `kovetskiy/atlassian-bitbucket-mesh`
	MESH_VERSION  = `1.3.1-1`
	MESH_DATA_DIR = `/srv/mesh/data`

	MESH_PROPERTIES = `# mesh.properties provided by external-hooks-test
hookscripts.gc.interval=1
hookscripts.gc.prune=1
#grpc.server.ssl.cert-chain-path=/srv/mesh/data/config/ssl/cert.pem
#grpc.server.ssl.private-key-path=/srv/mesh/data/config/ssl/key.pem
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
		data string
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

	err := mesh.inspect()
	switch {
	case err == errContainerNotFound:
		err := mesh.create()
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

func (node *Node) create() error {
	err := node.writeSSL()
	if err != nil {
		return karma.Format(err, "write ssl")
	}

	err = node.writeProperties()
	if err != nil {
		return karma.Format(err, "write properties")
	}

	execution := exec.New(
		"docker", "container", "create",
		"--network", node.Network,
		"--name", node.container,
		"-v", fmt.Sprintf("%s:%s", node.volumes.data, MESH_DATA_DIR),
		MESH_IMAGE+":"+MESH_VERSION,
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

	err = ioutil.WriteFile(
		filepath.Join(node.volumes.data, "mesh.properties"),
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
		if strings.Contains(err.Error(), "Error: No such container:") {
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

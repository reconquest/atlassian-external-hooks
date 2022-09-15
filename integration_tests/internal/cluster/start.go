package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket/mesh"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/docker"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

const (
	CLUSTER_SIZE = 3
)

type Cluster struct {
	*bitbucket.Node // points to Nodes[0]

	Nodes     []*bitbucket.Node
	MeshNodes []*mesh.Node
}

type StartOpts struct {
	ID      string
	Volumes string
	RunOpts bitbucket.RunOpts
}

func StartNew(opts StartOpts) (*Cluster, error) {
	opts.RunOpts.Properties = bitbucket.NewProperties().
		WithLicense(bitbucket.LICENSE_DATACENTER_3H).
		WithHazelcast()

	return clusterizeBitbucket(
		opts,
		func(replica int) (*bitbucket.Node, error) {
			return bitbucket.StartNew(bitbucket.StartNewOpts{
				ID:      opts.ID,
				Replica: &replica,
				RunOpts: opts.RunOpts,
				Volumes: opts.Volumes,
			})
		},
	)
}

func StartExisting(opts StartOpts) (*Cluster, error) {
	opts.RunOpts.Properties = bitbucket.NewProperties().
		WithLicense(bitbucket.LICENSE_DATACENTER_3H).
		WithHazelcast()

	return clusterizeBitbucket(
		opts,
		func(replica int) (*bitbucket.Node, error) {
			return bitbucket.StartExisting(bitbucket.StartExistingOpts{
				Container: fmt.Sprintf("%s-bitbucket-%d", opts.ID, replica),
				Replica:   &replica,
				RunOpts:   opts.RunOpts,
				Volumes:   opts.Volumes,
			})
		},
	)
}

func clusterize[T any](
	start func(replica int) (T, error),
) ([]T, error) {
	cluster := []T{}
	mutex := sync.Mutex{}

	pipeErr := make(chan error, CLUSTER_SIZE)
	done := make(chan struct{})

	starters := sync.WaitGroup{}
	for replica := 0; replica < CLUSTER_SIZE; replica++ {
		starters.Add(1)
		go func(replica int) {
			defer starters.Done()

			//time.Sleep(time.Second * 30 * time.Duration(replica))

			node, err := start(replica)
			if err != nil {
				pipeErr <- err
				return
			}

			mutex.Lock()
			cluster = append(cluster, node)
			mutex.Unlock()

		}(replica)
	}

	go func() {
		starters.Wait()
		close(done)
	}()

	select {
	case err := <-pipeErr:
		return nil, err
	case <-done:
		// assign the first node, so the entire clsuter behaves like
		// bitbucket.Node
	}

	return cluster, nil
}

func clusterizeBitbucket(
	opts StartOpts,
	start func(replica int) (*bitbucket.Node, error),
) (*Cluster, error) {
	var cluster Cluster
	var err error

	cluster.Nodes, err = clusterize(start)
	if err != nil {
		return nil, err
	}

	cluster.Node = cluster.Nodes[0]

	err = cluster.startMesh(opts)
	if err != nil {
		return nil, karma.Format(err, "unable to start mesh")
	}

	return &cluster, nil
}

func (cluster *Cluster) startMesh(opts StartOpts) error {
	var err error
	cluster.MeshNodes, err = clusterize(
		func(replica int) (*mesh.Node, error) {
			return mesh.Start(mesh.StartOpts{
				ID:      opts.ID,
				Replica: replica,
				Volumes: opts.Volumes,
				Network: opts.RunOpts.Network,
			})
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (cluster *Cluster) FlushLogs(kind bitbucket.LogsKind) {
	cluster.EachNode(func(node *bitbucket.Node) {
		node.FlushLogs(kind)
	})
}

func (cluster *Cluster) EachNode(fn func(node *bitbucket.Node)) {
	wg := &sync.WaitGroup{}
	for _, node := range cluster.Nodes {
		wg.Add(1)
		go func(node *bitbucket.Node) {
			defer wg.Done()
			fn(node)
		}(node)
	}
	wg.Wait()
}

func (cluster *Cluster) EachContainer(fn func(container string)) {
	wg := &sync.WaitGroup{}

	for _, node := range cluster.Nodes {
		wg.Add(1)
		go func(node *bitbucket.Node) {
			defer wg.Done()
			fn(node.Container())
		}(node)
	}

	for _, node := range cluster.MeshNodes {
		wg.Add(1)
		go func(node *mesh.Node) {
			defer wg.Done()
			fn(node.Container())
		}(node)
	}

	wg.Wait()
}

func (cluster *Cluster) AnyNode() *bitbucket.Node {
	return cluster.Nodes[rand.Intn(len(cluster.Nodes))]
}

func (cluster *Cluster) ConnectorURI(user *stash.User) string {
	return cluster.AnyNode().ConnectorURI(user)
}

func (cluster *Cluster) URI(path string) string {
	return cluster.AnyNode().URI(path)
}

func (cluster *Cluster) ClonePathSSH(repo, project string) string {
	return cluster.AnyNode().ClonePathSSH(repo, project)
}

func (cluster *Cluster) ClonePathHTTP(repo, project string) string {
	return cluster.AnyNode().ClonePathHTTP(repo, project)
}

func (cluster *Cluster) WriteFile(
	path string,
	content []byte,
	mode os.FileMode,
) error {
	var mutex sync.Mutex
	var errs []karma.Reason

	cluster.EachContainer(func(container string) {
		err := docker.WriteFile(
			container,
			bitbucket.BITBUCKET_DATA_DIR,
			path,
			content,
			mode,
		)
		if err != nil {
			mutex.Lock()
			errs = append(errs, err)
			mutex.Unlock()
		}
	})

	if len(errs) > 0 {
		return karma.Push("write file for each container", errs...)
	}

	return nil
}

func (cluster *Cluster) ReadFile(
	path string,
) (string, error) {
	var mutex sync.Mutex
	var errs []karma.Reason
	var results []string

	cluster.EachContainer(func(container string) {
		result, err := docker.ReadFile(
			container,
			path,
		)
		if err != nil {
			mutex.Lock()
			errs = append(errs, err)
			mutex.Unlock()
		} else {
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}
	})

	if len(results) > 0 {
		return results[0], nil
	}

	if len(errs) > 0 {
		return "", karma.Push("write file for each container", errs...)
	}

	return "", nil
}

func (cluster *Cluster) Verify() error {
	mutex := sync.Mutex{}
	errs := []karma.Reason{}

	cluster.EachNode(func(node *bitbucket.Node) {
		context := karma.
			Describe("node", node.Container()).
			Describe("ip", node.IP())

		cluster, err := node.Admin().GetCluster()
		if err != nil {
			mutex.Lock()
			errs = append(errs, context.Format(err, "request cluster info"))
			mutex.Unlock()

			return
		}

		if len(cluster.Nodes) != CLUSTER_SIZE {
			mutex.Lock()
			errs = append(errs, context.Format(
				fmt.Errorf(
					"expected %d nodes, got %d",
					CLUSTER_SIZE,
					len(cluster.Nodes),
				),
				"invalid cluster size",
			))
			mutex.Unlock()
		}
	})

	if len(errs) == 0 {
		return nil
	}

	return karma.Push("cluster verification failed", errs...)
}

func (cluster *Cluster) Configure() error {
	err := cluster.Node.Configure()
	if err != nil {
		return karma.Format(err, "unable to configure bitbucket")
	}

	meshNodes, err := cluster.Node.Admin().GetMeshNodes()
	if err != nil {
		return karma.Format(err, "unable to get mesh nodes")
	}

	alreadyRegistered := func(container string) bool {
		for _, node := range meshNodes {
			if node.Name == container {
				return true
			}
		}

		return false
	}

	for _, node := range cluster.MeshNodes {
		if alreadyRegistered(node.Container()) {
			log.Infof(nil, "mesh node %s already registered", node.Container())
			continue
		}

		created, err := cluster.Admin().CreateMeshNode(
			fmt.Sprintf("http://%s:7777", node.Container()),
		)
		if err != nil {
			return karma.Format(
				err,
				"unable to create mesh node for %s",
				node.Container(),
			)
		}

		log.Infof(
			karma.
				Describe("id", created.ID).
				Describe("name", created.Name).
				Describe("rpcUrl", created.RPCURL).
				Describe("offline", created.Offline).
				Describe("contaienr", node.Container()).
				Describe("ip", node.IP()),
			"mesh node created",
		)
	}

	err = cluster.Admin().EnableMesh()
	if err != nil {
		return karma.Format(err, "unable to enable mesh repository creation")
	}

	return nil
}

type ClusterLogWaiter struct {
	waiters []docker.LogWaiter

	context context.Context
	cancel  context.CancelFunc
}

func (clusterWaiter *ClusterLogWaiter) Wait(
	failer func(failureMessage string, msgAndArgs ...interface{}) bool,
	resource string,
	state string,
) {
	workers := sync.WaitGroup{}
	for _, waiter := range clusterWaiter.waiters {
		workers.Add(1)
		go func(waiter docker.LogWaiter) {
			defer workers.Done()
			waiter.Wait(failer, resource, state)

			clusterWaiter.cancel()
		}(waiter)
	}

	workers.Wait()
}

func (clusterWaiter *ClusterLogWaiter) Await() bool {
	var result atomic.Bool

	workers := sync.WaitGroup{}
	for _, waiter := range clusterWaiter.waiters {
		workers.Add(1)
		go func(waiter docker.LogWaiter) {
			defer workers.Done()
			if waiter.Await() {
				result.Store(true)
			}

			clusterWaiter.cancel()
		}(waiter)
	}

	workers.Wait()

	return result.Load()
}

func (cluster *Cluster) WaitLog(
	ctx context.Context,
	kind bitbucket.LogsKind,
	fn func(string) bool,
	duration time.Duration,
) docker.LogWaiter {
	ctx, cancel := context.WithCancel(ctx)

	clusterWaiter := &ClusterLogWaiter{
		context: ctx,
		cancel:  cancel,
	}

	cluster.EachNode(func(node *bitbucket.Node) {
		waiter := node.WaitLog(ctx, kind, fn, duration)

		clusterWaiter.waiters = append(
			clusterWaiter.waiters,
			waiter,
		)
	})

	return clusterWaiter
}

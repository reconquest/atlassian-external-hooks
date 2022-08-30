package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/karma-go"
)

const (
	CLUSTER_SIZE = 3
)

type Cluster struct {
	*bitbucket.Node // points to Nodes[0]

	Nodes []*bitbucket.Node
	mutex sync.Mutex
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

	return clusterize(
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

	return clusterize(
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

func clusterize(
	start func(replica int) (*bitbucket.Node, error),
) (*Cluster, error) {
	cluster := Cluster{}

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

			cluster.push(node)
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
		cluster.Node = cluster.Nodes[0]

		return &cluster, nil
	}
}

func (cluster *Cluster) push(node *bitbucket.Node) {
	cluster.mutex.Lock()
	defer cluster.mutex.Unlock()

	cluster.Nodes = append(cluster.Nodes, node)
}

func (cluster *Cluster) FlushLogs(kind bitbucket.LogsKind) {
	cluster.Each(func(node *bitbucket.Node) {
		node.FlushLogs(kind)
	})
}

func (cluster *Cluster) Each(fn func(node *bitbucket.Node)) {
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

func (cluster *Cluster) Any() *bitbucket.Node {
	return cluster.Nodes[rand.Intn(len(cluster.Nodes))]
}

func (cluster *Cluster) ConnectorURI(user *stash.User) string {
	return cluster.Any().ConnectorURI(user)
}

func (cluster *Cluster) URI(path string) string {
	return cluster.Any().URI(path)
}

func (cluster *Cluster) ClonePathSSH(repo, project string) string {
	return cluster.Any().ClonePathSSH(repo, project)
}

func (cluster *Cluster) ClonePathHTTP(repo, project string) string {
	return cluster.Any().ClonePathHTTP(repo, project)
}

func (cluster *Cluster) Verify() error {
	mutex := sync.Mutex{}
	errs := []karma.Reason{}

	cluster.Each(func(node *bitbucket.Node) {
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

type ClusterLogWaiter struct {
	waiters []bitbucket.LogWaiter

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
		go func(waiter bitbucket.LogWaiter) {
			defer workers.Done()
			waiter.Wait(failer, resource, state)

			clusterWaiter.cancel()
		}(waiter)
	}

	workers.Wait()
}

func (cluster *Cluster) WaitLog(
	ctx context.Context,
	kind bitbucket.LogsKind,
	fn func(string) bool,
	duration time.Duration,
) bitbucket.LogWaiter {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	clusterWaiter := &ClusterLogWaiter{
		context: ctx,
		cancel:  cancel,
	}

	cluster.Each(func(node *bitbucket.Node) {
		waiter := node.WaitLog(ctx, kind, fn, duration)

		clusterWaiter.waiters = append(
			clusterWaiter.waiters,
			waiter,
		)
	})

	return clusterWaiter
}

package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/reconquest/pom"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/status"
)

var version = "[manual build]"

var usage = `external-hooks-tests - run external hooks test suites.

Usage:
  external-hooks-test [options] --container=<container>
  external-hooks-test [options] [--keep]
  external-hooks-test [options] --clean
  external-hooks-test -h | --help

Options:
  -l --list                   List testcases.
  --clean                     Clean Docker resources.
  -C --container <container>  Use specified container.  
  -K --keep                   Keep work dir & bitbucket instance.
  -D --database <type>        Type of database to use.
                               [default: postgres]
  -u --skip-until <regexp>    Skip until specified pattern.
  --no-upgrade                Do not run suites with upgrades.
  --no-reproduce              Do not run suites with bug reproduces.
  -r --run <name>             Run only specified testcases.
  --no-randomize              Do not randomize tests order.
  --debug                     Set debug log level.
  --trace                     Set trace log level.
  --volumes <dir>             Directory for volumes. [default: .volumes]
  -h --help                   Show this help.
`

type Opts struct {
	FlagKeep        bool `docopt:"--keep"`
	FlagClean       bool `docopt:"--clean"`
	FlagTrace       bool `docopt:"--trace"`
	FlagDebug       bool `docopt:"--debug"`
	FlagNoUpgrade   bool `docopt:"--no-upgrade"`
	FlagNoReproduce bool `docopt:"--no-reproduce"`
	FlagList        bool `docopt:"--list"`
	FlagNoRandomize bool `docopt:"--no-randomize"`

	ValueContainer string `docopt:"--container"`
	ValueRun       string `docopt:"--run"`
	ValueDatabase  string `docopt:"--database"`
	ValueSkipUntil string `docopt:"--skip-until"`
	ValueVolumes   string `docopt:"--volumes"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	args, err := docopt.ParseArgs(usage, nil, "external-hooks-tests "+version)
	if err != nil {
		log.Fatal(err)
	}

	var opts Opts

	err = args.Bind(&opts)
	if err != nil {
		log.Fatal(err)
	}

	defer status.Destroy()

	switch {
	case opts.FlagDebug:
		log.SetLevel(log.LevelDebug)
	case opts.FlagTrace:
		log.SetLevel(log.LevelTrace)
	case opts.FlagClean:
		err := clean()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	rand.Seed(time.Now().UTC().UnixNano())

	workdir, err := ioutil.TempDir("", "external-hooks.test.")
	if err != nil {
		log.Fatalf(err, "create work dir")
	}

	ensureAddons()

	var (
		baseBitbucket = "6.2.0"
		latestAddon   = getAddon(getLatestVersionXML())
	)

	mode := ModeRun
	if opts.FlagList {
		mode = ModeList
	}

	if !opts.FlagNoReproduce {
		opts.FlagNoRandomize = true
	}

	suite := NewSuite(
		SuiteOpts{
			baseBitbucket: baseBitbucket,
			randomize:     mode == ModeRun && !opts.FlagNoRandomize,
			mode:          mode,
			skipUntil:     opts.ValueSkipUntil,
			filter: Filter{
				upgrade:   !opts.FlagNoUpgrade,
				reproduce: !opts.FlagNoReproduce,
				glob:      opts.ValueRun,
			},
		},
	)

	// TODO: add tests for different trigger configurations
	// TODO: add tests for BB 5.x.x

	run := runner.New(must(filepath.Abs(opts.ValueVolumes)), suite.CleanupHooks)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket:       baseBitbucket,
				AddonReproduced: getAddon("10.1.0"),
				AddonFixed:      latestAddon,
			},

			suite.TestBug_ProjectEnabledRepositoryOverriddenHooks_Reproduced,
			suite.TestBug_ProjectEnabledRepositoryOverriddenHooks_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket:       baseBitbucket,
				AddonReproduced: getAddon("10.0.0"),
				AddonFixed:      latestAddon,
			},

			suite.TestBug_ProjectHookCreatedBeforeRepository_Reproduced,
			suite.TestBug_ProjectHookCreatedBeforeRepository_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket:       baseBitbucket,
				AddonReproduced: getAddon("9.1.0"),
				AddonFixed:      latestAddon,
			},

			suite.TestBug_ProjectEnabledRepositoryDisabledHooks_Reproduced,
			suite.TestBug_ProjectEnabledRepositoryDisabledHooks_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket: baseBitbucket,
				Addon:     latestAddon,
			},
			suite.TestProjectHooks_DoNotCreateDisabledHooks,

			suite.TestHookScriptsLeak_NoLeakAfterRepositoryDelete,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket:       baseBitbucket,
				AddonReproduced: getAddon("10.2.1"),
				AddonFixed:      latestAddon,
			},

			suite.TestBug_UserWithoutProjectAccessModifiesInheritedHook_Reproduced,
			suite.TestBug_UserWithoutProjectAccessModifiesInheritedHook_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket:       baseBitbucket,
				AddonReproduced: getAddon("11.1.0"),
				AddonFixed:      latestAddon,
			},

			suite.TestBug_RepositoryHookCreatedBeforeProject_Reproduced,
			suite.TestBug_RepositoryHookCreatedBeforeProject_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket: baseBitbucket,
				Addon:     getAddon("12.0.1"),
			},
			suite.TestGlobalHooks,
			suite.TestGlobalHooks_PersonalRepositoriesFilter,
			suite.TestProjectHooks,
			suite.TestRepositoryHooks,
			suite.TestPersonalRepositoriesHooks,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				BitbucketFrom: baseBitbucket,
				BitbucketTo:   "6.9.0",
				Addon:         latestAddon,
			},
			suite.TestBitbucketUpgrade,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket: "7.0.0",
				Addon:     latestAddon,
			},
			suite.TestProjectHooks,
			suite.TestRepositoryHooks,
			suite.TestPersonalRepositoriesHooks,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				Bitbucket: "8.3.0",
				Cluster:   true,
				Addon:     latestAddon,
			},
			suite.TestGlobalHooks,
			suite.TestGlobalHooks_PersonalRepositoriesFilter,
			suite.TestProjectHooks,
			suite.TestRepositoryHooks,
			suite.TestPersonalRepositoriesHooks,
		),
	)

	run.Run(runner.RunOpts{
		Workdir:   workdir,
		Database:  opts.ValueDatabase,
		Randomize: mode == ModeRun && !opts.FlagNoRandomize,
		Container: opts.ValueContainer,
	})

	if !opts.FlagList {
		log.Infof(nil, "{run} all tests passed")
	}

	log.Debugf(nil, "{run} removing work dir: %s", workdir)
	err = os.RemoveAll(workdir)
	if err != nil {
		log.Errorf(err, "remove work dir")
	}

	if !opts.FlagKeep && opts.ValueContainer == "" {
		err := run.Cleanup()
		if err != nil {
			log.Fatalf(err, "cleanup runner")
		}
	} else {
		if run.Bitbucket() != nil {
			facts := karma.
				Describe("work_dir", workdir)
			nodes := run.Bitbuckets()
			if len(nodes) > 0 {
				facts = facts.Describe("shared/network", nodes[0].Network())

				for _, node := range nodes {
					facts = facts.Describe(node.Container()+"/container", node.Container()).
						Describe(node.Container()+"/volume", node.VolumeData())
				}
			}

			if run.Database() != nil {
				facts = facts.
					Describe("shared/database/container", run.Database().Container()).
					Describe("shared/database/volume", run.Database().Volume())
			}

			log.Infof(facts, "{run} following resources can be reused")
		}
	}
}

var builds = map[string]string{
	"12.0.1": "6702",
	"11.1.0": "6642",
	"10.2.2": "6592",
	"10.2.1": "6572",
	"10.1.0": "6532",
	"10.0.0": "6512",
	"9.1.0":  "6492",
}

func ensureAddons() {
	err := os.MkdirAll("builds", 0o755)
	if err != nil {
		log.Fatalf(err, "mkdir builds")
	}

	getters := &sync.WaitGroup{}
	for version := range builds {
		getters.Add(1)
		go func(version string) {
			defer getters.Done()
			getAddon(version)
		}(version)
	}

	getters.Wait()
}

func getAddon(version string) Addon {
	buildsPath := fmt.Sprintf("builds/external-hooks-%s.jar", version)

	_, err := os.Stat(buildsPath)
	if err == nil {
		return Addon{
			Version: version,
			Path:    buildsPath,
		}
	}

	if build, ok := builds[version]; ok {
		log.Infof(
			karma.Describe("build", build).Describe("version", version),
			"downloading add-on from Marketplace",
		)

		cmd := exec.New(
			"wget", "-O", buildsPath,
			fmt.Sprintf(
				"https://marketplace.atlassian.com/download/apps/1211631/version/%v",
				build,
			),
		)

		err := cmd.Run()
		if err != nil {
			log.Fatalf(
				karma.Describe("build", build).Reason(err),
				"download add-on %s from Marketplace to %q",
				version, buildsPath,
			)
		}

		return Addon{
			Version: version,
			Path:    buildsPath,
		}
	}

	targetPath := fmt.Sprintf("target/external-hooks-%s.jar", version)
	_, err = os.Stat(targetPath)
	if err != nil {
		log.Fatalf(
			err,
			"find add-on version %s at path %q and %q",
			version, buildsPath, targetPath,
		)
	}

	return Addon{
		Version: version,
		Path:    targetPath,
	}
}

func getLatestVersionXML() string {
	contents, err := ioutil.ReadFile("pom.xml")
	if err != nil {
		log.Fatalf(err, "read pom.xml")
	}

	model, err := pom.Unmarshal(contents)
	if err != nil {
		log.Fatalf(err, "unmarshal pom.xml")
	}

	version, err := model.Get("version")
	if err != nil {
		log.Fatalf(err, "read pom.xml version")
	}

	return version
}

func text(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

func clean() error {
	const prefix = "aeh-"

	filter := func(items []string) []string {
		dst := []string{}
		for _, item := range items {
			if strings.HasPrefix(item, prefix) {
				dst = append(dst, item)
			}
		}
		return dst
	}

	resources := struct {
		containers []string
		networks   []string
		volumes    []string
	}{}

	stdout, _, err := exec.New(
		"docker", "ps", "-a", "--format", "{{.Names}}",
	).NoStdLog().Output()
	if err != nil {
		return karma.Format(err, "get docker containers")
	}

	resources.containers = filter(strings.Split(strings.TrimSpace(string(stdout)), "\n"))

	stdout, _, err = exec.New(
		"docker", "volume", "ls", "--format", "{{.Name}}",
	).NoStdLog().Output()
	if err != nil {
		return karma.Format(err, "get docker volumes")
	}

	resources.volumes = filter(strings.Split(strings.TrimSpace(string(stdout)), "\n"))

	stdout, _, err = exec.New(
		"docker", "network", "ls", "--format", "{{.Name}}",
	).NoStdLog().Output()
	if err != nil {
		return karma.Format(err, "get docker volumes")
	}

	resources.networks = filter(strings.Split(strings.TrimSpace(string(stdout)), "\n"))

	context := karma.
		Describe("containers", resources.containers).
		Describe("volumes", resources.volumes).
		Describe("networks", resources.networks)

	log.Infof(context, "cleaning up resources")

	for _, container := range resources.containers {
		log.Infof(nil, "remove container %q", container)

		err = exec.New("docker", "rm", "-f", container).Run()
		if err != nil {
			log.Errorf(err, "remove container %q", container)
		}
	}

	for _, volume := range resources.volumes {
		log.Infof(nil, "remove volume %q", volume)

		err = exec.New("docker", "volume", "rm", volume).Run()
		if err != nil {
			log.Errorf(err, "remove volume %q", volume)
		}
	}

	for _, network := range resources.networks {
		log.Infof(nil, "remove network %q", network)

		err = exec.New("docker", "network", "rm", network).Run()
		if err != nil {
			log.Errorf(err, "remove network %q", network)
		}
	}

	return nil
}

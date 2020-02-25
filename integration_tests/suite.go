package main

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Suite struct {
	*runner.Runner
	*assert.Assertions
}

func NewSuite() *Suite {
	return &Suite{}
}

func (suite *Suite) Run(tests ...func()) runner.Suite {
	return func(run *runner.Runner, assert *assert.Assertions) {
		suite.Runner = run
		suite.Assertions = assert

		for _, test := range tests {
			name := runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
			name = strings.TrimPrefix(name, "main.(*Suite).")
			name = strings.TrimSuffix(name, "-fm")

			log.Infof(nil, "{test} running %s", name)

			test()
		}
	}
}

func (suite *Suite) ConfigureReceiveHook(
	key string,
	context *external_hooks.Context,
	name string,
	script []byte,
) *external_hooks.Hook {
	err := suite.Bitbucket().WriteFile(
		filepath.Join("shared", "external-hooks", name),
		script,
		0777,
	)
	suite.NoError(err, "should be able to write hook script to container")

	var settings = external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(name)

	var hook *external_hooks.Hook

	switch key {
	case external_hooks.HOOK_KEY_PRE_RECEIVE:
		hook = context.PreReceive(settings)
	case external_hooks.HOOK_KEY_POST_RECEIVE:
		hook = context.PostReceive(settings)
	}

	addon := external_hooks.Addon{
		BitbucketURI: suite.Bitbucket().GetConnectorURI(),
	}

	hook.Configure()
	err = addon.Enable(key, context)
	suite.NoError(err, "should be able to enable pre-receive hook")

	return hook
}

func (suite *Suite) ConfigurePreReceiveHook(
	context *external_hooks.Context,
	name string,
	script []byte,
) *external_hooks.Hook {
	return suite.ConfigureReceiveHook(
		external_hooks.HOOK_KEY_PRE_RECEIVE,
		context,
		name,
		script,
	)
}

func (suite *Suite) ConfigurePostReceiveHook(
	context *external_hooks.Context,
	name string,
	script []byte,
) *external_hooks.Hook {
	return suite.ConfigureReceiveHook(
		external_hooks.HOOK_KEY_POST_RECEIVE,
		context,
		name,
		script,
	)
}

func (suite *Suite) ExternalHooks() *external_hooks.Addon {
	return &external_hooks.Addon{
		BitbucketURI: suite.Bitbucket().GetConnectorURI(),
	}
}

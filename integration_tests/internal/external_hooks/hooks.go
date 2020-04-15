package external_hooks

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

const (
	HOOK_KEY_PRE_RECEIVE  = "com.ngs.stash.externalhooks.external-hooks:external-pre-receive-hook"
	HOOK_KEY_POST_RECEIVE = "com.ngs.stash.externalhooks.external-hooks:external-post-receive-hook"
	HOOK_KEY_MERGE_CHECK  = "com.ngs.stash.externalhooks.external-hooks:external-merge-check-hook"
)

type Addon struct {
	BitbucketURI string
}

func (addon *Addon) Register(
	key string,
	context *Context,
	settings *Settings,
) error {
	args := []string{
		key,
		"-e", settings.Executable,
	}

	if settings.Safe {
		args = append(args, "-s")
	}

	args = append(args, settings.Args...)

	return addon.command(context, "set", args...)
}

func (addon *Addon) Enable(key string, context *Context) error {
	return addon.command(context, "enable", key)
}

func (addon *Addon) Disable(key string, context *Context) error {
	return addon.command(context, "disable", key)
}

func (addon *Addon) OnProject(project string) *Context {
	return &Context{
		Addon:   addon,
		Project: project,
	}
}

func (addon *Addon) command(
	context *Context,
	command string,
	args ...string,
) error {
	args = append(
		[]string{
			command,
			"-b", addon.BitbucketURI,
			"-p", context.Project,
		},
		args...,
	)

	if context.Repository != "" {
		args = append(args, "-r", context.Repository)
	}

	return exec.New("bitbucket-external-hook", args...).Run()
}

type Settings struct {
	Safe       bool
	Executable string
	Args       []string

	Override *bool
}

func NewSettings() *Settings {
	return &Settings{}
}

func (settings *Settings) UseSafePath(enabled bool) *Settings {
	settings.Safe = enabled

	return settings
}

// UseOverride supported only on add-on version >= 10.2.0
func (settings *Settings) UseOverride(overridden bool) *Settings {
	settings.Override = &overridden

	return settings
}

func (settings *Settings) WithExecutable(executable string) *Settings {
	settings.Executable = executable

	return settings
}

func (settings *Settings) WithArgs(args ...string) *Settings {
	settings.Args = args

	return settings
}

type Hook struct {
	*Context

	key string
}

type Context struct {
	*Addon

	Project    string
	Repository string
}

func (context Context) OnRepository(repository string) *Context {
	context.Repository = repository

	return &context
}

func (context *Context) PreReceive() *Hook {
	return &Hook{
		context,
		HOOK_KEY_PRE_RECEIVE,
	}
}

func (context *Context) PostReceive() *Hook {
	return &Hook{
		context,
		HOOK_KEY_POST_RECEIVE,
	}
}

func (context *Context) MergeCheck() *Hook {
	return &Hook{
		context,
		HOOK_KEY_MERGE_CHECK,
	}
}

func (hook *Hook) Configure(settings *Settings) error {
	log.Debugf(
		karma.
			Describe("context", hook.Context).
			Describe("settings", settings),
		"{hook} configuring %s",
		hook.key,
	)

	return hook.Context.Register(hook.key, hook.Context, settings)
}

func (hook *Hook) Enable() error {
	log.Debugf(
		karma.Describe("context", hook.Context),
		"{hook} enabling %s",
		hook.key,
	)

	return hook.Context.Enable(hook.key, hook.Context)
}

func (hook *Hook) Disable() error {
	log.Debugf(
		karma.Describe("context", hook.Context),
		"{hook} disabling %s",
		hook.key,
	)

	return hook.Context.Disable(hook.key, hook.Context)
}

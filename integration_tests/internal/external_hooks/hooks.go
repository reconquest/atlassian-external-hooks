package external_hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

const (
	HOOK_KEY_PRE_RECEIVE  = "com.ngs.stash.externalhooks.external-hooks:external-pre-receive-hook"
	HOOK_KEY_POST_RECEIVE = "com.ngs.stash.externalhooks.external-hooks:external-post-receive-hook"
	HOOK_KEY_MERGE_CHECK  = "com.ngs.stash.externalhooks.external-hooks:external-merge-check-hook"
)

type RequestGlobalHooks struct {
	Safe    bool   `json:"safe_path"`
	Exe     string `json:"exe"`
	Params  string `json:"params"`
	Enabled bool   `json:"enabled"`
}

type ResponseGlobalHooksSetup struct {
	ErrorsForm   []string            `json:"errors_form"`
	ErrorsFields map[string][]string `json:"errors_fields"`
}

type ResponseFactoryHooks struct {
	ID       int64 `json:"id"`
	Started  bool  `json:"started"`
	Finished bool  `json:"finished"`
	Current  int64 `json:"current"`
	Total    int64 `json:"total"`
}

type Addon struct {
	BitbucketURI string
}

func (addon *Addon) call(
	method string,
	path string,
	payload interface{},
	response interface{},
) error {
	var encoded []byte
	var err error
	if payload != nil {
		encoded, err = json.Marshal(payload)
		if err != nil {
			return karma.Format(err, "json marshal")
		}
	}

	buffer := bytes.NewReader(encoded)

	log.Tracef(
		karma.Describe("payload", string(encoded)),
		"{http request} %s %s", method, path,
	)

	request, err := http.NewRequest(
		method,
		addon.BitbucketURI+path,
		buffer,
	)
	if err != nil {
		return err
	}

	request.Header.Add("Content-Type", "application/json")

	reply, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(reply.Body)
	if err != nil {
		return karma.Format(err, "read response body")
	}

	defer reply.Body.Close()

	log.Tracef(
		karma.
			Describe("status", reply.StatusCode).
			Describe("body", string(body)),
		"{http response} %s %s", method, path,
	)

	err = json.Unmarshal(body, response)
	if err != nil {
		return karma.
			Describe("body", string(body)).
			Describe("status", reply.StatusCode).
			Format(err, "json unmarshal")
	}

	return nil
}

func (addon *Addon) Register(
	key string,
	context *Context,
	settings *Settings,
) error {
	if !context.Global() {
		args := []string{
			key,
			"-e", settings.Exe,
		}

		if settings.Safe {
			args = append(args, "-s")
		}

		args = append(args, settings.Params...)

		return addon.command(context, "set", args...)
	}

	var reply ResponseGlobalHooksSetup
	err := addon.call(
		"PUT",
		"/rest/external-hooks/1.0/global-hooks/"+key,
		RequestGlobalHooks{
			Safe:    settings.Safe,
			Exe:     settings.Exe,
			Params:  strings.Join(settings.Params, "\r\n"),
			Enabled: true,
		},
		&reply,
	)
	if err != nil {
		return err
	}

	return addon.getReplyError(reply)
}

func (addon *Addon) getReplyError(reply ResponseGlobalHooksSetup) error {
	if len(reply.ErrorsFields) > 0 || len(reply.ErrorsForm) > 0 {
		return karma.
			Describe("errors_form", reply.ErrorsForm).
			Describe("erros_fields", reply.ErrorsFields).
			Reason("the add-on returned errors")
	}

	return nil
}

func (addon *Addon) Enable(key string, context *Context) error {
	if !context.Global() {
		return addon.command(context, "enable", key)
	}

	return nil
}

func (addon *Addon) Wait(context *Context) error {
	if !context.Global() {
		return nil
	}

	return addon.factoryApply()
}

func (addon *Addon) factoryApply() error {
	var reply ResponseFactoryHooks
	err := addon.call(
		"POST",
		"/rest/external-hooks/1.0/factory/hooks",
		nil,
		&reply,
	)
	if err != nil {
		return err
	}

	for !reply.Finished {
		err := addon.call(
			"GET",
			"/rest/external-hooks/1.0/factory/state/"+fmt.Sprint(reply.ID),
			nil,
			&reply,
		)
		if err != nil {
			return err
		}

		log.Debugf(
			karma.Describe(
				"current",
				reply.Current,
			).Describe(
				"total",
				reply.Total,
			),
			"waiting for factory, state id: %d",
			reply.ID,
		)

		time.Sleep(time.Millisecond * 50)
	}

	return nil
}

func (addon *Addon) Disable(key string, context *Context) error {
	if !context.Global() {
		return addon.command(context, "disable", key)
	}

	var reply ResponseGlobalHooksSetup
	err := addon.call(
		"PUT",
		"/rest/external-hooks/1.0/global-hooks/"+key,
		RequestGlobalHooks{
			Enabled: false,
		},
		&reply,
	)
	if err != nil {
		return err
	}

	return nil
}

func (addon *Addon) Inherit(key string, context *Context) error {
	if context.Global() {
		return errors.New(
			"global hooks can't inherit hook settings (it's already global)",
		)
	}

	return addon.command(context, "inherit", key)
}

func (addon *Addon) OnGlobal() *Context {
	return &Context{
		Addon: addon,
	}
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
	Safe   bool
	Exe    string
	Params []string
}

func NewSettings() *Settings {
	return &Settings{}
}

func (settings *Settings) UseSafePath(enabled bool) *Settings {
	settings.Safe = enabled

	return settings
}

func (settings *Settings) WithExe(exe string) *Settings {
	settings.Exe = exe

	return settings
}

func (settings *Settings) WithParams(args ...string) *Settings {
	settings.Params = args

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

func (context *Context) Global() bool {
	return context.Project == ""
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

func (hook *Hook) Wait() error {
	return hook.Context.Wait(hook.Context)
}

func (hook *Hook) Inherit() error {
	log.Debugf(
		karma.Describe("context", hook.Context),
		"{hook} inheriting %s",
		hook.key,
	)

	return hook.Context.Inherit(hook.key, hook.Context)
}

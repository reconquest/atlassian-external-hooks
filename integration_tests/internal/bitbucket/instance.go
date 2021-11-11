package bitbucket

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/database"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

type StartupStatus struct {
	State    string
	Progress struct {
		Message    string
		Percentage int
	}
}

type AtlToken struct {
	value   string
	cookies []*http.Cookie
}

type Logs struct {
	*sync.Cond

	lines []string
}

type Instance struct {
	id              string
	version         string
	container       string
	database        database.Database
	volumeData      string
	volumeLibNative string
	network         string
	ip              string
	opts            struct {
		RunOpts
		ConfigureOpts
	}

	stacktraceLogs *Logs
	testcaseLogs   *Logs
}

func (instance *Instance) ID() string {
	return instance.id
}

func (instance *Instance) Opts() struct {
	RunOpts
	ConfigureOpts
} {
	return instance.opts
}

func (instance *Instance) ConnectorURI(user *stash.User) string {
	var auth *url.Userinfo

	if user == nil {
		auth = url.UserPassword(
			instance.opts.AdminUser,
			instance.opts.AdminPassword,
		)
	} else {
		auth = url.UserPassword(user.Name, user.Password)
	}

	url := url.URL{
		Scheme: "http",
		User:   auth,
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortHTTP),
	}

	return url.String()
}

func (instance *Instance) URI(path string) string {
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortHTTP),
		Path:   path,
	}

	return url.String()
}

func (instance *Instance) ClonePathSSH(repo, project string) string {
	url := url.URL{
		Scheme: "ssh",
		User:   url.User("git"),
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortSSH),
		Path:   fmt.Sprintf("%s/%s.git", strings.ToLower(repo), project),
	}

	return url.String()
}

func (instance *Instance) ClonePathHTTP(repo, project string) string {
	return instance.URI(
		fmt.Sprintf(
			"scm/%s/%s.git",
			strings.ToLower(repo),
			project,
		),
	)
}

func (instance *Instance) Container() string {
	return instance.container
}

func (instance *Instance) Version() string {
	return instance.version
}

func (instance *Instance) ReadFile(path string) (string, error) {
	execution := exec.New(
		"docker", "exec", instance.container, "cat", path,
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return "", karma.Format(
			err,
			"get stdout pipe for docker exec",
		)
	}

	err = execution.Start()
	if err != nil {
		return "", karma.Format(
			err,
			"start docker cp",
		)
	}

	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	return string(data), execution.Wait()
}

func (instance *Instance) ListFiles(path string) ([]string, error) {
	execution := exec.New(
		"docker",
		"cp",
		fmt.Sprintf("%s:%s",
			instance.container,
			filepath.Join(instance.GetApplicationDataDir(), path),
		),
		"-",
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker cp",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker cp",
		)
	}

	files := []string{}

	reader := tar.NewReader(stdout)

	for {
		next, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, karma.Format(
				err,
				"read next file from docker cp tar output",
			)
		}

		files = append(
			files,
			strings.TrimPrefix(
				next.Name,
				filepath.Base(path)+"/",
			),
		)
	}

	err = execution.Wait()
	if err != nil {
		return nil, karma.Format(
			err,
			"finalize docker cp",
		)
	}

	// First item is always directory itself.
	return files[1:], nil
}

type File struct {
	Name     string
	Contents string
}

func (instance *Instance) ReadFiles(path string) ([]File, error) {
	execution := exec.New(
		"docker",
		"cp",
		fmt.Sprintf("%s:%s",
			instance.container,
			filepath.Join(instance.GetApplicationDataDir(), path),
		),
		"-",
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker cp",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker cp",
		)
	}

	files := []File{}

	reader := tar.NewReader(stdout)

	for {
		next, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, karma.Format(
				err,
				"read next file from docker cp tar output",
			)
		}

		name := strings.TrimPrefix(
			next.Name,
			filepath.Base(path)+"/",
		)

		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, karma.Format(
				err,
				"read next file contents from the docker cp tar output",
			)
		}

		files = append(
			files,
			File{
				Name:     name,
				Contents: string(contents),
			},
		)
	}

	err = execution.Wait()
	if err != nil {
		return nil, karma.Format(
			err,
			"finalize docker cp",
		)
	}

	// First item is always directory itself.
	return files[1:], nil
}

func (instance *Instance) WriteFile(
	path string,
	content []byte,
	mode os.FileMode,
) error {
	execution := exec.New(
		"docker",
		"cp",
		"-",
		fmt.Sprintf("%s:%s",
			instance.container,
			instance.GetApplicationDataDir(),
		),
	)

	err := execution.Start()
	if err != nil {
		return karma.Format(
			err,
			"start docker cp",
		)
	}

	stdin := execution.GetStdin()

	writer := tar.NewWriter(stdin)

	err = writer.WriteHeader(&tar.Header{
		Name: path,
		Mode: int64(mode),
		Size: int64(len(content)),
	})
	if err != nil {
		return karma.Format(
			err,
			"write file header",
		)
	}

	_, err = writer.Write(content)
	if err != nil {
		return karma.Format(
			err,
			"write file contents",
		)
	}

	err = writer.Close()
	if err != nil {
		return karma.Format(
			err,
			"close file",
		)
	}

	err = stdin.Close()
	if err != nil {
		return karma.Format(
			err,
			"close docker cp stdin",
		)
	}

	err = execution.Wait()
	if err != nil {
		return karma.Format(
			err,
			"complete docker cp",
		)
	}

	return nil
}

func (instance *Instance) Stop() error {
	err := exec.New(
		"docker",
		"kill",
		"-s", "INT",
		instance.container,
	).Run()
	if err != nil {
		return karma.Format(
			err,
			"send docker stop",
		)
	}

	return exec.New("docker", "wait", instance.container).Run()
}

func (instance *Instance) RemoveContainer() error {
	return exec.New(
		"docker",
		"rm", "-f",
		instance.container,
	).Run()
}

func (instance *Instance) RemoveVolumeData() error {
	return exec.New(
		"docker",
		"volume",
		"rm", "-f",
		instance.volumeData,
	).Run()
}

func (instance *Instance) RemoveVolumeLibNative() error {
	return exec.New(
		"docker",
		"volume",
		"rm", "-f",
		instance.volumeLibNative,
	).Run()
}

func (instance *Instance) Configure(opts ConfigureOpts) error {
	if opts.AdminEmail == "" {
		opts.AdminEmail = "we@reconquest.io"
	}

	instance.opts.ConfigureOpts = opts

	configured, err := instance.isConfigured()
	if err != nil {
		return err
	}

	if configured {
		return nil
	}

	token, err := instance.getAtlToken(nil)
	if err != nil {
		return err
	}

	// err = instance.configureDatabase(token)
	// if err != nil {
	//    return err
	//}

	err = instance.configureLicense(token)
	if err != nil {
		return err
	}

	err = instance.configureAdministrator(token)
	if err != nil {
		return err
	}

	return nil
}

func (instance *Instance) VolumeData() string {
	return instance.volumeData
}

func (instance *Instance) VolumeLibNative() string {
	return instance.volumeLibNative
}

func (instance *Instance) Network() string {
	return instance.network
}

func (instance *Instance) StacktraceLogs() *Logs {
	return instance.stacktraceLogs
}

func (instance *Instance) TestcaseLogs() *Logs {
	return instance.testcaseLogs
}

type LogEntryWaiter struct {
	duration time.Duration
	group    *sync.WaitGroup
}

func (waiter *LogEntryWaiter) Wait(
	failer func(failureMessage string, msgAndArgs ...interface{}) bool,
	resource string,
	state string,
) {
	done := make(chan struct{})
	go func() {
		waiter.group.Wait()
		close(done)
	}()

	select {
	case <-time.After(waiter.duration):
		failer(
			resource+" should be "+state+" but no such log message occurred",
			"duration: %s",
			waiter.duration,
		)
	case <-done:
	}
}

var DefaultWaitLogEntryDuration = time.Second * 10

func (instance *Instance) WaitLogEntry(fn func(string) bool) *LogEntryWaiter {
	return instance.WaitLogEntryContext(
		context.Background(),
		instance.testcaseLogs,
		fn,
		DefaultWaitLogEntryDuration,
	)
}

func (instance *Instance) WaitLogEntryContext(
	ctx context.Context,
	logs *Logs,
	fn func(string) bool,
	duration time.Duration,
) *LogEntryWaiter {
	waiter := sync.WaitGroup{}
	waiter.Add(1)

	cursor := 0
	prev := 0

	go func() {
		defer waiter.Done()

		for {
			logs.L.Lock()

			for {
				now := len(logs.lines)
				if now == prev {
					logs.Wait()
					continue
				}

				prev = now
				break
			}

			// FlushLogs() was called before
			if cursor > len(logs.lines)-1 {
				cursor = 0
			}

			lines := logs.lines[cursor:]
			logs.L.Unlock()

			cursor += len(lines)

			for _, line := range lines {
				if fn(line) {
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return &LogEntryWaiter{
		group:    &waiter,
		duration: duration,
	}
}

func (instance *Instance) FlushLogs(logs *Logs) {
	if instance == nil {
		return
	}
	logs.L.Lock()
	logs.lines = []string{}
	logs.L.Unlock()
}

func (instance *Instance) getAtlToken(
	response *http.Response,
) (*AtlToken, error) {
	if response == nil {
		var err error

		response, err = http.Get(instance.URI("/setup"))
		if err != nil {
			return nil, karma.Format(
				err,
				"request setup page for bitbucket instance",
			)
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, karma.Format(
			err,
			"read response body from bitbucket setup page",
		)
	}

	matches := regexp.MustCompile(
		`<input type="hidden" name="atl_token" value="([^"]+)">`,
	).FindStringSubmatch(string(body))

	if len(matches) == 0 {
		return nil, karma.Format(
			err,
			"match atl_token from bitbucket setup page",
		)
	}

	return &AtlToken{
		value:   matches[1],
		cookies: response.Cookies(),
	}, nil
}

func (instance *Instance) postSetupForm(
	form url.Values,
	token *AtlToken,
) error {
	form.Set("atl_token", token.value)

	request, err := http.NewRequest(
		http.MethodPost,
		instance.URI("/setup"),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return karma.Format(
			err,
			"create http request",
		)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for _, cookie := range token.cookies {
		request.AddCookie(cookie)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return karma.Format(
			err,
			"post setup form to bitbucket instance",
		)
	}

	if response.StatusCode != 200 {
		return karma.
			Describe("code", response.StatusCode).
			Reason(
				"unexpected status code after post to setup form",
			)
	}

	return nil
}

func (instance *Instance) isConfigured() (bool, error) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest(
		http.MethodGet,
		instance.URI("/setup"),
		nil,
	)
	if err != nil {
		return false, karma.Format(
			err,
			"create http request",
		)
	}

	response, err := client.Do(request)
	if err != nil {
		return false, karma.Format(
			err,
			"check is bitbucket already configured or not",
		)
	}

	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return true, nil
	} else {
		return false, nil
	}
}

func (instance *Instance) configureDatabase(
	token *AtlToken,
) error {
	form := url.Values{}
	form.Set("step", "database")
	form.Set("internal", "true")
	form.Set("locale", "en_US")
	form.Set("type", "postgres")

	return instance.postSetupForm(form, token)
}

func (instance *Instance) configureLicense(token *AtlToken) error {
	form := url.Values{}
	form.Set("step", "settings")
	form.Set("license", instance.opts.License)
	form.Set("applicationTitle", "Bitbucket")
	form.Set("baseUrl", instance.URI("/"))

	return instance.postSetupForm(form, token)
}

func (instance *Instance) configureAdministrator(token *AtlToken) error {
	form := url.Values{}
	form.Set("step", "user")
	form.Set("username", instance.opts.AdminUser)
	form.Set("fullname", instance.opts.AdminUser)
	form.Set("email", instance.opts.AdminEmail)
	form.Set("password", instance.opts.AdminPassword)
	form.Set("confirmPassword", instance.opts.AdminPassword)

	return instance.postSetupForm(form, token)
}

func (instance *Instance) GetApplicationDataDir() string {
	return BITBUCKET_DATA_DIR
}

func (instance *Instance) create() error {
	type M map[string]interface{}

	springApplicationConfig, _ := json.Marshal(M{
		"logging": M{
			"logger": M{
				"com.ngs.stash.externalhooks": "debug",
			},
		},
	})

	var (
		jdbcDriver   = instance.database.Driver()
		jdbcURL      = instance.database.URL()
		jdbcUser     = instance.database.User()
		jdbcPassword = instance.database.Password()
	)

	bitbucketDataDirLib, err := filepath.Abs(
		"./integration_tests/assets/bitbucket-data-dir-lib/",
	)
	if err != nil {
		return err
	}

	execution := exec.New(
		"docker", "container", "create",
		"--add-host=marketplace.atlassian.com:127.0.0.1",
		"--network", instance.opts.Network,
		"-e", "JDBC_DRIVER="+jdbcDriver,
		"-e", "JDBC_URL="+jdbcURL,
		"-e", "JDBC_USER="+jdbcUser,
		"-e", "JDBC_PASSWORD="+jdbcPassword,
		"-e", "ELASTICSEARCH_ENABLED=false",
		"-e", fmt.Sprintf(
			`SPRING_APPLICATION_JSON=%s`,
			string(springApplicationConfig),
		),
		// required for Oracle
		"-e", "TZ=Europe/Moscow",
		"-v", fmt.Sprintf(
			"%s:%s",
			instance.volumeData,
			instance.GetApplicationDataDir(),
		),
		"-v", fmt.Sprintf(
			"%s:%s",
			bitbucketDataDirLib,
			instance.GetApplicationDataDir()+"/lib",
		),
		"-v", fmt.Sprintf(
			"%s:%s/lib/native",
			instance.volumeLibNative,
			instance.GetApplicationDataDir(),
		),
		"--name", instance.container,
		fmt.Sprintf(BITBUCKET_IMAGE, instance.version),
	)

	err = execution.Run()
	if err != nil {
		return err
	}

	return instance.start()
}

func (instance *Instance) start() error {
	execution := exec.New("docker", "container", "start", instance.container)
	return execution.Run()
}

func (instance *Instance) connect() error {
	execution := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f",
		"{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
		instance.container,
	)

	stdout, _, err := execution.Output()
	if err != nil {
		return err
	}

	ips := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	if len(ips) == 0 {
		return karma.
			Describe("container", instance.container).
			Format(
				err,
				"no ip addresses found on container",
			)
	}

	instance.ip = ips[0]

	return nil
}

func (instance *Instance) getStartupStatus() (*StartupStatus, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		instance.URI("/system/startup"),
		nil,
	)
	if err != nil {
		return nil, karma.Format(
			err,
			"create http request",
		)
	}

	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		if err, ok := err.(*url.Error); ok {
			if _, ok := err.Err.(*net.OpError); ok {
				// skip network error while bitbucket is starting
				return nil, nil
			}

			if err.Err == io.EOF {
				// skip incomplete reads
				return nil, nil
			}

			if err.Err.Error() == "http: server closed idle connection" {
				return nil, nil
			}
		}

		return nil, karma.Format(
			err,
			"request startup status",
		)
	}

	defer response.Body.Close()

	var status StartupStatus

	err = json.NewDecoder(response.Body).Decode(&status)
	if err != nil {
		return nil, karma.Format(
			err,
			"decode startup status",
		)
	}

	return &status, nil
}

func (instance *Instance) startLogReader(output bool) (*Logs, error) {
	log.Debugf(
		nil,
		"{bitbucket} starting log reader for container %q",
		instance.container,
	)

	execution := exec.New(
		"docker",
		"logs", "-f",
		"--tail", "0",
		instance.container,
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker logs",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker logs",
		)
	}

	logs := &Logs{
		Cond: sync.NewCond(&sync.Mutex{}),
	}

	go func() {
		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			logs.L.Lock()

			logs.lines = append(
				logs.lines,
				scanner.Text(),
			)

			logs.Broadcast()

			logs.L.Unlock()

			if output {
				log.Tracef(
					nil, "{bitbucket %s} log | %s",
					instance.version,
					scanner.Text(),
				)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Errorf(nil, "error while reading bitbucket instance logs")
		}
	}()

	return logs, nil
}

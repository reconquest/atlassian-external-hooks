package status

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kovetskiy/lorg"
	"github.com/reconquest/barely"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/loreley"
	"github.com/reconquest/pkg/log"
	"github.com/reconquest/sign-go"
)

var (
	bar *barely.StatusBar

	status = &struct {
		CurrentTest   string
		LastTest      string
		LastDuration  string
		TotalDuration string

		Total int
		Done  int64

		Updated int64
	}{}

	started = false
)

func SetTotal(n int) {
	status.Total = n
	render()
}

func AddDone() {
	atomic.AddInt64(&status.Done, 1)
	render()
}

func SetCurrentTest(name string) {
	status.CurrentTest = name
	render()
}

func SetLastTest(name string) {
	status.LastTest = name
	render()
}

func SetLastDuration(duration time.Duration) {
	status.LastDuration = fmt.Sprintf(
		"%03dm %02ds",
		duration/time.Minute,
		(duration%time.Minute)/time.Second,
	)
	render()
}

func init() {
	format, err := loreley.CompileWithReset(
		` {bold}{bg 235}{fg 70}  {.Done}{fg 7}/{.Total} `+
			`{if .TotalDuration}{bg 7}{fg 16} {.TotalDuration} {end}`+
			`{bg 4}{fg 233} {.CurrentTest} `+
			`{if .LastTest}{reset}{bold} {.LastTest} {reset}({.LastDuration}){end}`,
		nil,
	)
	if err != nil {
		panic(err)
	}

	bar = barely.NewStatusBar(format.Template)

	bar.SetStatus(status)

	logger := log.GetLogger()

	mutex := sync.Mutex{}
	logger.SetDisplayer(func(level lorg.Level, hierarchy karma.Hierarchical) {
		mutex.Lock()
		bar.Clear(os.Stderr)
		cog.Display(logger, level, hierarchy)
	})
	logger.SetSender(func(lorg.Level, karma.Hierarchical) error {
		render()
		mutex.Unlock()
		return nil
	})

	go sign.Notify(func(os.Signal) bool {
		Destroy()
		os.Exit(1)
		return false
	}, syscall.SIGINT, syscall.SIGTERM)
}

func Destroy() {
	bar.Clear(os.Stderr)
}

func render() {
	if !started {
		started = true

		go func() {
			started := time.Now()
			for {
				time.Sleep(time.Second)
				elapsed := time.Since(started)
				status.TotalDuration = fmt.Sprintf(
					"%03dm %02ds",
					elapsed/time.Minute,
					(elapsed%time.Minute)/time.Second,
				)
			}
		}()
	}
	err := bar.Render(os.Stderr)
	if err != nil {
		log.Errorf(err, "statusbar render")
	}
}

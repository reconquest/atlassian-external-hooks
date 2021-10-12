package status

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/kovetskiy/lorg"
	"github.com/reconquest/barely"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/loreley"
	"github.com/reconquest/pkg/log"
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
	status.LastDuration = duration.String()
	render()
}

func init() {
	format, err := loreley.CompileWithReset(
		` {bg 3}{fg 70}  {.Done}{fg 0}/{.Total} `+
			`{bg 4}{fg 233}{bold} {.CurrentTest} `+
			`{bg 5}{fg 233}{bold} {.LastTest} {bg 6} {.LastDuration} `+
			`{bg 7} {.TotalDuration}`+
			`{bg 253}{fg 0} `+
			``,
		nil,
	)
	if err != nil {
		panic(err)
	}

	bar = barely.NewStatusBar(format.Template)

	bar.SetStatus(status)

	logger := log.GetLogger()

	logger.SetDisplayer(func(level lorg.Level, hierarchy karma.Hierarchical) {
		bar.Clear(os.Stderr)
		cog.Display(logger, level, hierarchy)
	})
	logger.SetSender(func(lorg.Level, karma.Hierarchical) error {
		render()
		return nil
	})
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
				status.TotalDuration = time.Since(started).String()
			}
		}()
	}
	err := bar.Render(os.Stderr)
	if err != nil {
		log.Errorf(err, "statusbar render")
	}
}

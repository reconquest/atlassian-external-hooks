package status

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kovetskiy/lorg"
	"github.com/reconquest/barely"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/loreley"
	"github.com/reconquest/pkg/log"
)

var (
	bar   *barely.StatusBar
	mutex = sync.Mutex{}

	status = &struct {
		CurrentTest  string
		LastTest     string
		LastDuration string

		Total int
		Done  int64

		Updated int64
	}{}
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
			`{bg 5}{fg 233}{bold} {.LastTest} {bg 6} {.LastDuration}`+
			`{bg 253}{fg 0} `+
			``,
		nil,
	)
	if err != nil {
		panic(err)
	}

	bar = barely.NewStatusBar(format.Template)

	bar.SetStatus(status)

	log.GetLogger().SetSender(func(lorg.Level, karma.Hierarchical) error {
		render()
		return nil
	})
}

func Destroy() {
	bar.Clear(os.Stderr)
}

func render() {
	mutex.Lock()
	err := bar.Render(os.Stderr)
	mutex.Unlock()
	if err != nil {
		log.Errorf(err, "statusbar render")
	}
}

package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/logdna/logdna-go/logger"
	"github.com/rnr-capital/newsfeed-backend/utils/flag"
)

// global accessible logger
var (
	LogV2 *LogDNA
)

// This init function is only for testing cases, where the entry point is not
// main function. Unit test will fail with nil pointer dereference if we don't
// init here.
func init() {
	// flag.ParseFlags()
	initLogDna()
}

type LogDNA struct {
	*logger.Logger
}

func (l *LogDNA) Infof(params ...interface{}) {
	strs := make([]string, len(params))

	for i, param := range params {
		strs[i] = fmt.Sprint(param)
	}

	l.Info(strings.Join(strs, ", "))
}

func (l *LogDNA) Debugf(params ...interface{}) {
	strs := make([]string, len(params))

	for i, param := range params {
		strs[i] = fmt.Sprint(param)
	}

	l.Debug(strings.Join(strs, ", "))
}

func (l *LogDNA) Errorf(params ...interface{}) {
	strs := make([]string, len(params))

	for i, param := range params {
		strs[i] = fmt.Sprint(param)
	}

	l.Error(strings.Join(strs, ", "))
}

func initLogDna() {
	key := "f9b80cc9e5176a1e1df36a4ad2a52eeb"

	// Configure your options with your desired level, hostname, app, ip address, mac address and environment.
	// Hostname is the only required field in your options- the rest are optional.
	options := logger.Options{
		Level: "debug",
	}
	env := os.Getenv("NEWSMUX_ENV")
	if len(env) == 0 {
		env = "unknown"
	}
	options.Hostname = "backend-" + env
	options.App = strings.ReplaceAll(*flag.ServiceName, "_", "-")
	fmt.Println("app", options.App)
	var err error
	logV2, err := logger.NewLogger(options, key)
	if err != nil {
		panic(err)
	}
	LogV2 = &LogDNA{
		logV2,
	}
	fmt.Println("LogDNA initialized")
}

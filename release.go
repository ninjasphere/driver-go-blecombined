// +build release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
	"github.com/ninjasphere/go-ninja/logger"
)

func debug(string, ...interface{}) {}

func init() {
	logger.GetLogger("").SetLogLevel(loggo.INFO)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "7b66d4013bdcd0541287fd9b00376253",
		ReleaseStage: "production",
	})
}

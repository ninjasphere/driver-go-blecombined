// +build release

package main

import "github.com/bugsnag/bugsnag-go"

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "7b66d4013bdcd0541287fd9b00376253",
		ReleaseStage: "production",
	})
}

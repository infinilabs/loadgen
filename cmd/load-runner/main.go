package main

import (
	"os"
	"time"

	"infini.sh/framework"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/util"
	"infini.sh/loadgen/cmd/load-runner/config"
)

func main() {
	terminalHeader := ("   __   ___  _      ___  ___   __    __\n")
	terminalHeader += ("  / /  /___\\/_\\    /   \\/ _ \\ /__\\/\\ \\ \\\n")
	terminalHeader += (" / /  //  ///_\\\\  / /\\ / /_\\//_\\ /  \\/ /\n")
	terminalHeader += ("/ /__/ \\_//  _  \\/ /_// /_\\\\//__/ /\\  /\n")
	terminalHeader += ("\\____|___/\\_/ \\_/___,'\\____/\\__/\\_\\ \\/\n\n")

	terminalFooter := ("")

	app := framework.NewApp("load-runner", "A testing suite runner",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.BuildNumber), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)
	defer app.Shutdown()

	appConfig := AppConfig{}
	if app.Setup(func() {
		environments := map[string]string{}
		ok, err := env.ParseConfig("env", &environments)
		if ok && err != nil {
			panic(err)
		}

		tests := []Test{}
		ok, err = env.ParseConfig("tests", &tests)
		if ok && err != nil {
			panic(err)
		}

		appConfig.Environments = environments
		appConfig.Tests = tests
		appConfig.Init()
	}, func() {
		go func() {
			startRunner(&appConfig)
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	}, nil) {
		app.Run()
	}
}

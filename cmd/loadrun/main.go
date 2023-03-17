package main

import (
	"os"
	"strings"

	"infini.sh/framework"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/util"
	"infini.sh/loadgen/cmd/loadrun/config"
)

const ()

func main() {
	terminalHeader := `   __   ___  _      ___  __           __` + "\n"
	terminalHeader += `  / /  /___\/_\    /   \/__\/\ /\  /\ \ \` + "\n"
	terminalHeader += ` / /  //  ///_\\  / /\ / \// / \ \/  \/ /` + "\n"
	terminalHeader += `/ /__/ \_//  _  \/ /_// _  \ \_/ / /\  /` + "\n"
	terminalHeader += `\____|___/\_/ \_/___,'\/ \_/\___/\_\ \/` + "\n"
	terminalFooter := ""
	app := framework.NewApp("loadrun", "A testing suite runner",
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
		environs := os.Environ()
		for _, env := range environs {
			kv := strings.Split(env, "=")
			if len(kv) == 2 {
				k, v := kv[0], kv[1]
				if _, ok := environments[k]; ok {
					environments[k] = v
				}
			}
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
			if !startRunner(&appConfig) {
				os.Exit(1)
			}
			os.Exit(0)
		}()
	}, nil) {
		app.Run()
	}
}

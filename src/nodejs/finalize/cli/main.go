package main

import (
	"nodejs/cache"
	"nodejs/finalize"
	_ "nodejs/hooks"
	"nodejs/npm"
	"nodejs/yarn"
	"os"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := libbuildpack.Logger{}
	logger.SetOutput(os.Stdout)

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.SetStagingEnvironment(); err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(11)
	}

	f := finalize.Finalizer{
		Stager: stager,
		Yarn: &yarn.Yarn{
			BuildDir: stager.BuildDir(),
			Command:  &libbuildpack.Command{},
			Log:      logger,
		},
		NPM: &npm.NPM{
			BuildDir: stager.BuildDir(),
			Command:  &libbuildpack.Command{},
			Log:      logger,
		},
		Manifest: manifest,
		Log:      logger,
		Cache: &cache.Cache{
			Stager:  stager,
			Command: &libbuildpack.Command{},
			Log:     logger,
		},
	}

	if err := finalize.Run(&f); err != nil {
		os.Exit(12)
	}

	if err := libbuildpack.RunAfterCompile(stager); err != nil {
		logger.Error("After Compile: %s", err.Error())
		os.Exit(13)
	}

	if err := stager.SetLaunchEnvironment(); err != nil {
		logger.Error("Unable to setup launch environment: %s", err.Error())
		os.Exit(14)
	}

	stager.StagingComplete()
}

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
	if err := libbuildpack.SetStagingEnvironment(stager.DepsDir()); err != nil {
		stager.Log.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(11)
	}

	f := finalize.Finalizer{
		Stager: stager,
		Yarn: &yarn.Yarn{
			BuildDir: stager.BuildDir(),
			Command:  libbuildpack.Command{},
			Logger:   logger,
		},
		NPM: &npm.NPM{
			BuildDir: stager.BuildDir(),
			Command:  libbuildpack.Command{},
			Logger:   logger,
		},
		Manifest: manifest,
		Log:      logger,
		Cache: &cache.Cache{
			Stager:  stager,
			Command: libbuildpack.Command{},
			Logger:  logger,
		},
	}

	if err := finalize.Run(&f); err != nil {
		os.Exit(12)
	}

	if err := libbuildpack.RunAfterCompile(stager); err != nil {
		stager.Log.Error("After Compile: %s", err.Error())
		os.Exit(13)
	}

	if err := libbuildpack.SetLaunchEnvironment(stager.DepsDir, stager.BuildDir); err != nil {
		stager.Log.Error("Unable to setup launch environment: %s", err.Error())
		os.Exit(14)
	}

	stager.StagingComplete()
}

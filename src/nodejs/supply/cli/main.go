package main

import (
	_ "nodejs/hooks"
	"nodejs/supply"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := &libbuildpack.Logger{}
	stager, err := libbuildpack.NewStager(os.Args[1:], logger))
	if err != nil {
		os.Exit(10)
	}

	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	err = libbuildpack.RunBeforeCompile(stager)
	if err != nil {
		logger.Error("Before Compile: %s", err.Error())
		os.Exit(12)
	}

	err = libbuildpack.SetStagingEnvironment(stager.DepsDir)
	if err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(13)
	}

	s := supply.Supplier{
		Stager: stager,
		Manifest: stager.Manifest(),
		Log: logger
		Command: libbuildpack.Command{}
	}

	err = supply.Run(&s)
	if err != nil {
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}
}

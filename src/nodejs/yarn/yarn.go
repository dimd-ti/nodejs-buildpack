package yarn

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type Yarn struct {
	BuildDir string
	Command  Command
	Logger   libbuildpack.Logger
}

func (y *Yarn) Build() error {
	y.Logger.Info("Installing node modules (yarn.lock)")

	npmOfflineCache := filepath.Join(y.BuildDir, "npm-packages-offline-cache")
	offline, err := libbuildpack.FileExists(npmOfflineCache)
	if err != nil {
		return err
	}

	installArgs := []string{"install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(y.BuildDir, ".cache/yarn")}
	checkArgs := []string{"check"}

	if offline {
		y.Logger.Info("Found yarn mirror directory %s", npmOfflineCache)
		if err := y.Command.Execute(y.BuildDir, os.Stdout, os.Stdout, "yarn", "config", "set", "yarn-offline-mirror", npmOfflineCache); err != nil {
			return err
		}
		y.Logger.Info("Running yarn in offline mode")

		installArgs = append(installArgs, "--offline")
		checkArgs = append(checkArgs, "--offline")
	} else {
		y.Logger.Info("Running yarn in online mode")
		y.Logger.Info("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror")
	}

	os.Setenv("npm_config_nodedir", os.Getenv("NODE_HOME"))
	defer os.Unsetenv("npm_config_nodedir")

	if err := y.Command.Execute(y.BuildDir, os.Stdout, os.Stdout, "yarn", installArgs...); err != nil {
		return err
	}

	if err := y.Command.Execute(y.BuildDir, ioutil.Discard, os.Stderr, "yarn", checkArgs...); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}
		y.Logger.Warning("yarn.lock is outdated")
	} else {
		y.Logger.Info("yarn.lock and package.json match")
	}

	return nil
}

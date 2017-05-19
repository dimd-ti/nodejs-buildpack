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
	Log      libbuildpack.Logger
}

func (y *Yarn) Build() error {
	y.Log.Info("Installing node modules (yarn.lock)")

	npmOfflineCache := filepath.Join(y.BuildDir, "npm-packages-offline-cache")
	offline, err := libbuildpack.FileExists(npmOfflineCache)
	if err != nil {
		return err
	}

	installArgs := []string{"install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(y.BuildDir, ".cache/yarn")}
	checkArgs := []string{"check"}

	if offline {
		y.Log.Info("Found yarn mirror directory %s", npmOfflineCache)
		if err := y.Command.Execute(y.BuildDir, os.Stdout, os.Stdout, "yarn", "config", "set", "yarn-offline-mirror", npmOfflineCache); err != nil {
			return err
		}
		y.Log.Info("Running yarn in offline mode")

		installArgs = append(installArgs, "--offline")
		checkArgs = append(checkArgs, "--offline")
	} else {
		y.Log.Info("Running yarn in online mode")
		y.Log.Info("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror")
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
		y.Log.Warning("yarn.lock is outdated")
	} else {
		y.Log.Info("yarn.lock and package.json match")
	}

	return nil
}

type StdoutAndLog struct {
	Logfile string
}

func (s *StdoutAndLog) Write(p []byte) (int, error) {
	n, err := os.Stdout.Write(p)
	if err != nil {
		return n, err
	}

	f, err := os.OpenFile(s.Logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()
	if err != nil {
		return 0, err
	}

	return f.Write(p)
}

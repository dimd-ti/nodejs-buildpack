package yarn

import (
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}
type Logger interface {
	Info(format string, args ...interface{})
	Warning(format string, args ...interface{})
	// Error(format string, args ...interface{})
	// BeginStep(format string, args ...interface{})
	// Debug(format string, args ...interface{})
	// Protip(tip string, help_url string)
}

type Yarn struct {
	BuildDir string
	Command  Command
	Logger   Logger
}

func (y *Yarn) Build() error {
	y.Logger.Info("Installing node modules (yarn.lock)")

	npmOfflineCache := filepath.Join(y.BuildDir, "npm-packages-offline-cache")
	offline, err := libbuildpack.FileExists(npmOfflineCache)
	if err != nil {
		return err
	}

	args := []string{"install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(y.BuildDir, ".cache/yarn")}

	if offline {
		y.Logger.Info("Found yarn mirror directory %s", npmOfflineCache)
		if err := y.Command.Execute(y.BuildDir, os.Stdout, os.Stdout, "yarn", "config", "set", "yarn-offline-mirror", npmOfflineCache); err != nil {
			return err
		}
		y.Logger.Info("Running yarn in offline mode")

		args = append(args, "--offline")
	} else {
		y.Logger.Info("Running yarn in online mode")
		y.Logger.Info("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror")
	}

	os.Setenv("npm_config_nodedir", os.Getenv("NODE_HOME"))
	defer os.Unsetenv("npm_config_nodedir")

	if err := y.Command.Execute(y.BuildDir, os.Stdout, os.Stdout, "yarn", args...); err != nil {
		return err
	}

	return nil
}
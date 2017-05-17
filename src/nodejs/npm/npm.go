package npm

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
}

type NPM struct {
	BuildDir string
	Command  Command
	Logger   Logger
}

func (n *NPM) Build() error {
	pkgExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "package.json"))
	if err != nil {
		return err
	}

	if !pkgExists {
		n.Logger.Info("Skipping (no package.json)")
		return nil
	}

	shrinkwrapExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "npm-shrinkwrap.json"))
	if err != nil {
		return err
	}

	if shrinkwrapExists {
		n.Logger.Info("Installing node modules (package.json + shrinkwrap)")
	} else {
		n.Logger.Info("Installing node modules (package.json)")
	}

	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(n.BuildDir, ".npmrc"), "--cache", filepath.Join(n.BuildDir, ".npm")}
	return n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", npmArgs...)
}

func (n *NPM) Rebuild() error {
	pkgExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "package.json"))
	if err != nil {
		return err
	}

	if !pkgExists {
		n.Logger.Info("Skipping (no package.json)")
		return nil
	}

	shrinkwrapExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "npm-shrinkwrap.json"))
	if err != nil {
		return err
	}

	n.Logger.Info("Rebuilding any native modules")
	if err := n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", "rebuild", "--nodedir="+os.Getenv("NODE_HOME")); err != nil {
		return err
	}

	if shrinkwrapExists {
		n.Logger.Info("Installing any new modules (package.json + shrinkwrap)")
	} else {
		n.Logger.Info("Installing any new modules (package.json)")
	}

	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(n.BuildDir, ".npmrc")}
	return n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", npmArgs...)
}

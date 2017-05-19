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

type NPM struct {
	BuildDir string
	Command  Command
	Logger   libbuildpack.Logger
}

func (n *NPM) Build() error {
	doBuild, source, err := n.doBuild()
	if err != nil {
		return err
	}
	if !doBuild {
		return nil
	}

	n.Logger.Info("Installing node modules (%s)", source)
	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(n.BuildDir, ".npmrc"), "--cache", filepath.Join(n.BuildDir, ".npm")}
	return n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", npmArgs...)
}

func (n *NPM) Rebuild() error {
	doBuild, source, err := n.doBuild()
	if err != nil {
		return err
	}
	if !doBuild {
		return nil
	}

	n.Logger.Info("Rebuilding any native modules")
	if err := n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", "rebuild", "--nodedir="+os.Getenv("NODE_HOME")); err != nil {
		return err
	}

	n.Logger.Info("Installing any new modules (%s)", source)
	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(n.BuildDir, ".npmrc")}
	return n.Command.Execute(n.BuildDir, os.Stdout, os.Stdout, "npm", npmArgs...)
}

func (n *NPM) doBuild() (bool, string, error) {
	pkgExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "package.json"))
	if err != nil {
		return false, "", err
	}

	if !pkgExists {
		n.Logger.Info("Skipping (no package.json)")
		return false, "", nil
	}

	shrinkwrapExists, err := libbuildpack.FileExists(filepath.Join(n.BuildDir, "npm-shrinkwrap.json"))
	if err != nil {
		return false, "", err
	}

	if shrinkwrapExists {
		return true, "package.json + shrinkwrap", nil
	}
	return true, "package.json", nil
}

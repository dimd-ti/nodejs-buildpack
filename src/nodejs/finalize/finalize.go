package finalize

import (
	"bytes"
	"io/ioutil"
	"nodejs/cache"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Yarn interface {
	Build() error
}
type NPM interface {
	Build() error
	Rebuild() error
}

type Manifest interface {
	RootDir() string
}

type Finalizer struct {
	Stager     *libbuildpack.Stager
	CacheDirs  []string
	PreBuild   string
	PostBuild  string
	NPM        NPM
	NPMRebuild bool
	Yarn       Yarn
	UseYarn    bool
	Manifest   Manifest
}

func Run(f *Finalizer) error {
	if err := f.ReadPackageJSON(); err != nil {
		f.Stager.Log.Error("Failed parsing package.json: %s", err.Error())
		return err
	}

	if err := f.TipVendorDependencies(); err != nil {
		f.Stager.Log.Error(err.Error())
		return err
	}

	f.ListNodeConfig(os.Environ())

	cacher, err := cache.New(f.Stager, f.CacheDirs)
	if err != nil {
		f.Stager.Log.Error("Unable to initialize cache: %s", err.Error())
		return err
	}

	if err := cacher.Restore(); err != nil {
		f.Stager.Log.Error("Unable to restore cache: %s", err.Error())
		return err
	}

	if err := f.BuildDependencies(); err != nil {
		f.Stager.Log.Error("Unable to build dependencies: %s", err.Error())
		return err
	}

	if err := cacher.Save(); err != nil {
		f.Stager.Log.Error("Unable to save cache: %s", err.Error())
		return err
	}

	if err := f.CopyProfileScripts(); err != nil {
		f.Stager.Log.Error("Unable to copy profile.d scripts: %s", err.Error())
		return err
	}

	f.ListDependencies()

	return nil
}

func (f *Finalizer) ReadPackageJSON() error {
	var err error
	var p struct {
		CacheDirs1 []string `json:"cacheDirectories"`
		CacheDirs2 []string `json:"cache_directories"`
		Scripts    struct {
			PreBuild  string `json:"heroku-prebuild"`
			PostBuild string `json:"heroku-postbuild"`
		} `json:"scripts"`
	}

	if f.UseYarn, err = libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir, "yarn.lock")); err != nil {
		return err
	}

	if f.NPMRebuild, err = libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir, "node_modules")); err != nil {
		return err
	}

	f.CacheDirs = []string{}

	if err := libbuildpack.NewJSON().Load(filepath.Join(f.Stager.BuildDir, "package.json"), &p); err != nil {
		if os.IsNotExist(err) {
			f.Stager.Log.Warning("No package.json found")
			return nil
		} else {
			return err
		}
	}

	if len(p.CacheDirs1) > 0 {
		f.CacheDirs = p.CacheDirs1
	} else if len(p.CacheDirs2) > 0 {
		f.CacheDirs = p.CacheDirs2
	}
	f.PreBuild = p.Scripts.PreBuild
	f.PostBuild = p.Scripts.PostBuild

	return nil
}

func (f *Finalizer) TipVendorDependencies() error {
	subdirs, err := hasSubdirs(filepath.Join(f.Stager.BuildDir, "node_modules"))
	if err != nil {
		return err
	}
	if !subdirs {
		f.Stager.Log.Protip("It is recommended to vendor the application's Node.js dependencies",
			"http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring")
	}

	return nil
}

func (f *Finalizer) ListNodeConfig(environment []string) {
	npmConfigProductionTrue := false
	nodeEnv := "production"

	for _, env := range environment {
		if strings.HasPrefix(env, "NPM_CONFIG_") || strings.HasPrefix(env, "YARN_") || strings.HasPrefix(env, "NODE_") {
			f.Stager.Log.Info(env)
		}

		if env == "NPM_CONFIG_PRODUCTION=true" {
			npmConfigProductionTrue = true
		}

		if strings.HasPrefix(env, "NODE_ENV=") {
			nodeEnv = env[9:]
		}
	}

	if npmConfigProductionTrue && nodeEnv != "production" {
		f.Stager.Log.Info("npm scripts will see NODE_ENV=production (not '%s')\nhttps://docs.npmjs.com/misc/config#production", nodeEnv)
	}
}

func (f *Finalizer) BuildDependencies() error {
	var tool string
	if f.UseYarn {
		tool = "yarn"
	} else {
		tool = "npm"
	}

	f.Stager.Log.BeginStep("Building dependencies")

	if err := f.runPrebuild(tool); err != nil {
		return err
	}

	if f.UseYarn {
		if err := f.Yarn.Build(); err != nil {
			return err
		}
	} else {
		if f.NPMRebuild {
			f.Stager.Log.Info("Prebuild detected (node_modules already exists)", f.PreBuild)
			if err := f.NPM.Rebuild(); err != nil {
				return err
			}
		} else {
			if err := f.NPM.Build(); err != nil {
				return err
			}
		}
	}

	if err := f.runPostbuild(tool); err != nil {
		return err
	}

	return nil
}

func (f *Finalizer) CopyProfileScripts() error {
	profiledDir := filepath.Join(f.Stager.DepDir(), "profile.d")
	if err := os.MkdirAll(profiledDir, 0755); err != nil {
		return err
	}
	return libbuildpack.CopyDirectory(filepath.Join(f.Manifest.RootDir(), "profile"), profiledDir)
}

func (f *Finalizer) ListDependencies() {
	if os.Getenv("NODE_VERBOSE") != "true" {
		return
	}

	if f.UseYarn {
		f.Stager.Command.Execute(f.Stager.BuildDir, os.Stdout, ioutil.Discard, "yarn", "list", "--depth=0")
	} else {
		f.Stager.Command.Execute(f.Stager.BuildDir, os.Stdout, ioutil.Discard, "npm", "ls", "--depth=0")
	}
}

func (f *Finalizer) runPrebuild(tool string) error {
	if f.PreBuild == "" {
		return nil
	}

	return f.runScript(f.PreBuild, tool)
}

func (f *Finalizer) runPostbuild(tool string) error {
	if f.PostBuild == "" {
		return nil
	}

	return f.runScript(f.PostBuild, tool)
}

func (f *Finalizer) runScript(script, tool string) error {
	args := []string{"run", script}
	if tool == "npm" {
		args = append(args, "--if-present")
	}

	f.Stager.Log.Info("Running %s (%s)", script, tool)

	return f.Stager.Command.Execute(f.Stager.BuildDir, os.Stdout, os.Stderr, tool, args...)

}

func hasSubdirs(path string) (bool, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	for _, file := range files {
		if file.IsDir() {
			return true, nil
		}
	}

	return false, nil
}

func (f *Finalizer) findVersion(binary string) (string, error) {
	buffer := new(bytes.Buffer)
	if err := f.Stager.Command.Execute("", buffer, ioutil.Discard, binary, "--version"); err != nil {
		return "", err
	}
	return strings.TrimSpace(buffer.String()), nil
}

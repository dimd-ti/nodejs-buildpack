package libbuildpack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Stager struct {
	buildDir string
	cacheDir string
	depsDir  string
	depsIdx  string
	manifest *Manifest
	log      Logger
}

func NewStager(args []string, logger Logger, manifest *Manifest) *Stager {
	buildDir := args[0]
	cacheDir := args[1]
	depsDir := ""
	depsIdx := ""

	if len(args) >= 4 {
		depsDir = args[2]
		depsIdx = args[3]
	}

	s := &Stager{buildDir: buildDir,
		cacheDir: cacheDir,
		depsDir:  depsDir,
		depsIdx:  depsIdx,
		manifest: manifest,
		log:      logger,
	}

	return s
}

func (s *Stager) DepDir() string {
	return filepath.Join(s.depsDir, s.depsIdx)
}

func (s *Stager) WriteConfigYml(config interface{}) error {
	if config == nil {
		config = map[interface{}]interface{}{}
	}
	data := map[string]interface{}{"name": s.manifest.Language(), "config": config}
	y := &YAML{}
	return y.Write(filepath.Join(s.DepDir(), "config.yml"), data)
}

func (s *Stager) WriteEnvFile(envVar, envVal string) error {
	envDir := filepath.Join(s.DepDir(), "env")

	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err

	}

	return ioutil.WriteFile(filepath.Join(envDir, envVar), []byte(envVal), 0644)
}

func (s *Stager) AddBinDependencyLink(destPath, sourceName string) error {
	binDir := filepath.Join(s.DepDir(), "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	relPath, err := filepath.Rel(binDir, destPath)
	if err != nil {
		return err
	}

	return os.Symlink(relPath, filepath.Join(binDir, sourceName))
}

func (s *Stager) LinkDirectoryInDepDir(destDir, depSubDir string) error {
	srcDir := filepath.Join(s.DepDir(), depSubDir)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(destDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		relPath, err := filepath.Rel(srcDir, filepath.Join(destDir, file.Name()))
		if err != nil {
			return err
		}

		if err := os.Symlink(relPath, filepath.Join(srcDir, file.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stager) CheckBuildpackValid() error {
	version, err := s.manifest.Version()
	if err != nil {
		s.log.Error("Could not determine buildpack version: %s", err.Error())
		return err
	}

	s.log.BeginStep("%s Buildpack version %s", strings.Title(s.manifest.Language()), version)

	err = s.manifest.CheckStackSupport()
	if err != nil {
		s.log.Error("Stack not supported by buildpack: %s", err.Error())
		return err
	}

	s.manifest.CheckBuildpackVersion(s.cacheDir)

	return nil
}

func (s *Stager) StagingComplete() {
	s.manifest.StoreBuildpackMetadata(s.cacheDir)
}

func (s *Stager) ClearCache() error {
	files, err := ioutil.ReadDir(s.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(filepath.Join(s.cacheDir, file.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Stager) ClearDepDir() error {
	files, err := ioutil.ReadDir(s.DepDir())
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Name() != "config.yml" {
			if err := os.RemoveAll(filepath.Join(s.DepDir(), file.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Stager) WriteProfileD(scriptName, scriptContents string) error {
	profileDir := filepath.Join(s.DepDir(), "profile.d")

	err := os.MkdirAll(profileDir, 0755)
	if err != nil {
		return err
	}

	return writeToFile(strings.NewReader(scriptContents), filepath.Join(profileDir, scriptName), 0755)
}

func (s *Stager) BuildDir() string {
	return s.buildDir
}

func (s *Stager) CacheDir() string {
	return s.cacheDir
}

func (s *Stager) DepsIdx() string {
	return s.depsIdx
}

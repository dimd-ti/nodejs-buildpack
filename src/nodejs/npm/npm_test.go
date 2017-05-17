package npm_test

import (
	"bytes"
	"io/ioutil"
	n "nodejs/npm"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=npm.go --destination=mocks_test.go --package=npm_test

var _ = Describe("Yarn", func() {
	var (
		err         error
		buildDir    string
		npm         *n.NPM
		logger      libbuildpack.Logger
		buffer      *bytes.Buffer
		mockCtrl    *gomock.Controller
		mockCommand *MockCommand
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger()
		logger.SetOutput(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)
	})

	JustBeforeEach(func() {
		npm = &n.NPM{
			BuildDir: buildDir,
			Logger:   logger,
			Command:  mockCommand,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("Build", func() {
		Context("package.json exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte("xxx"), 0644)).To(Succeed())
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "npm", "install", "--unsafe-perm", "--userconfig", filepath.Join(buildDir, ".npmrc"), "--cache", filepath.Join(buildDir, ".npm")).Return(nil)
			})

			Context("npm-shrinkwrap.json exists", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(buildDir, "npm-shrinkwrap.json"), []byte("yyy"), 0644)).To(Succeed())
				})

				It("runs the install, telling users about shrinkwrap", func() {
					Expect(npm.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Installing node modules (package.json + shrinkwrap)"))
				})
			})

			Context("npm-shrinkwrap.json does not exist", func() {
				It("runs the install", func() {
					Expect(npm.Build()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Installing node modules (package.json)"))
				})
			})
		})

		Context("package.json does not exist", func() {
			It("skips the install", func() {
				Expect(npm.Build()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Skipping (no package.json)"))
			})
		})
	})

	Describe("Rebuild", func() {
	})
})

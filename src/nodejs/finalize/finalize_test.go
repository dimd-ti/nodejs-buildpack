package finalize_test

import (
	"bytes"
	"io/ioutil"
	"nodejs/finalize"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=../vendor/github.com/cloudfoundry/libbuildpack/command_runner.go --destination=mocks_command_runner_test.go --package=finalize_test

var _ = Describe("Finalize", func() {
	var (
		err               error
		buildDir          string
		finalizer         *finalize.Finalizer
		logger            libbuildpack.Logger
		buffer            *bytes.Buffer
		mockCtrl          *gomock.Controller
		mockCommandRunner *MockCommandRunner
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger()
		logger.SetOutput(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommandRunner = NewMockCommandRunner(mockCtrl)

	})

	JustBeforeEach(func() {
		bps := &libbuildpack.Stager{
			BuildDir: buildDir,
			Log:      logger,
			Command:  mockCommandRunner,
		}

		finalizer = &finalize.Finalizer{
			Stager: bps,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("TipVendorDependencies", func() {
		Context("node_modules exists and has subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules", "exciting_module"), 0755)).To(BeNil())
			})

			It("does not log anything", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("node_modules exists and has NO subdirectories", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(buildDir, "node_modules"), 0755)).To(BeNil())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "node_modules", "a_file"), []byte("content"), 0644)).To(BeNil())
			})

			It("logs a pro tip", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
			})
		})

		Context("node_modules does not exist", func() {
			It("logs a pro tip", func() {
				Expect(finalizer.TipVendorDependencies()).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("PRO TIP: It is recommended to vendor the application's Node.js dependencies"))
				Expect(buffer.String()).To(ContainSubstring("http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring"))
			})
		})
	})
})
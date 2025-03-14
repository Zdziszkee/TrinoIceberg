package config_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configurations "github.com/zdziszkee/swift-codes/internal/configurations"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Loader Suite")
}

var _ = Describe("Config Loader", func() {
	BeforeEach(func() {
		// Ensure a clean environment so that env overrides take effect.
		os.Clearenv()
	})

	AfterEach(func() {
		// Clean up environment variables using double underscores.
		os.Unsetenv("APP_DATABASE__SERVER_URI")
		os.Unsetenv("APP_LOG__LEVEL")
	})

	It("should load default configuration when no file is provided", func() {
		cfg, err := configurations.Load("")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.AppName).To(Equal("swift-codes"))
		Expect(cfg.Log.Level).To(Equal("info"))
		Expect(cfg.Database.ServerURI).To(Equal("http://test:password@trino:8080"))
	})

	It("should override config values with environment variables", func() {
		// Now use double underscores so that the callback produces e.g. "database.server_uri"
		os.Setenv("APP_DATABASE__SERVER_URI", "http://override:pass@localhost:8080")
		os.Setenv("APP_LOG__LEVEL", "debug")
		cfg, err := configurations.Load("")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Database.ServerURI).To(Equal("http://override:pass@localhost:8080"))
		Expect(cfg.Log.Level).To(Equal("debug"))
	})

	It("should load configuration from a valid config file", func() {
		content := `
app_name = "test-app"

[log]
level = "warn"
format = "json"

[database]
server_uri = "https://file:pass@localhost:8443"
catalog = "file_catalog"
schema = "file_schema"
max_open_conns = 10
max_idle_conns = 5
conn_max_lifetime = "30m"

[data]
swift_codes_file = "/data/swift_codes.csv"
auto_load = false
`
		tmpFile, err := os.CreateTemp("", "config-*.toml")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write([]byte(content))
		Expect(err).NotTo(HaveOccurred())
		tmpFile.Close()

		cfg, err := configurations.Load(tmpFile.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.AppName).To(Equal("test-app"))
		Expect(cfg.Log.Level).To(Equal("warn"))
		Expect(cfg.Log.Format).To(Equal("json"))
		Expect(cfg.Database.ServerURI).To(Equal("https://file:pass@localhost:8443"))
		Expect(cfg.Database.Catalog).To(Equal("file_catalog"))
		Expect(cfg.Database.Schema).To(Equal("file_schema"))
		Expect(cfg.Database.MaxOpenConns).To(Equal(10))
		Expect(cfg.Database.ConnMaxLifetime).To(BeNumerically("~", 30*time.Minute, time.Second))
		Expect(cfg.Data.SwiftCodesFile).To(Equal("/data/swift_codes.csv"))
		Expect(cfg.Data.AutoLoad).To(BeFalse())
	})

	It("should validate mandatory fields and fail on invalid config", func() {
		// Set the value to empty using the double-underscore key so it overrides the default.
		os.Setenv("APP_DATABASE__SERVER_URI", "")
		_, err := configurations.Load("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("database server_uri cannot be empty"))
	})
})

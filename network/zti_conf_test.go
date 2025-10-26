package network

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestWriteZitiRouterConfig tests router config writing
func TestWriteZitiRouterConfig(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "router.yml")

		cfg := ZitiRouterConfig{
			Version: 3,
			Identity: ZitiIdentity{
				Cert:       "router.cert",
				ServerCert: "router.server.cert",
				Key:        "router.key",
				CA:         "ca.cert",
			},
			Ctrl: ZitiCtrlEndpoint{Endpoint: "tls:127.0.0.1:6262"},
			Dialers: []ZitiDialer{
				{Binding: "transport"},
			},
		}

		err := WriteZitiRouterConfig(filename, cfg)
		require.NoError(t, err)

		// Verify file exists
		assert.FileExists(t, filename)

		// Verify file content
		data, err := os.ReadFile(filename)
		require.NoError(t, err)

		var loaded ZitiRouterConfig
		err = yaml.Unmarshal(data, &loaded)
		require.NoError(t, err)

		assert.Equal(t, 3, loaded.Version)
		assert.Equal(t, "router.cert", loaded.Identity.Cert)
		assert.Equal(t, "tls:127.0.0.1:6262", loaded.Ctrl.Endpoint)
	})

	t.Run("invalid directory path", func(t *testing.T) {
		cfg := ZitiRouterConfig{Version: 3}

		err := WriteZitiRouterConfig("/nonexistent/dir/router.yml", cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create file")
	})

	t.Run("read-only directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Make directory read-only
		err := os.Chmod(tmpDir, 0444)
		require.NoError(t, err)
		defer os.Chmod(tmpDir, 0755) // Restore for cleanup

		cfg := ZitiRouterConfig{Version: 3}
		filename := filepath.Join(tmpDir, "router.yml")

		err = WriteZitiRouterConfig(filename, cfg)
		assert.Error(t, err)
	})
}

// TestWriteZitiControllerConfig tests controller config writing
func TestWriteZitiControllerConfig(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "controller.yml")

		cfg := ZitiControllerConfig{
			Version: 3,
			DB:      "ctrl.db",
			Identity: ZitiIdentity{
				Cert:       "ctrl-client.cert",
				ServerCert: "ctrl-server.cert",
				Key:        "ctrl.key",
				CA:         "ca.cert",
			},
			Ctrl: ZitiCtrlListener{Listener: "tls:127.0.0.1:6262"},
		}

		err := WriteZitiControllerConfig(filename, cfg)
		require.NoError(t, err)

		// Verify file exists
		assert.FileExists(t, filename)

		// Verify file content
		data, err := os.ReadFile(filename)
		require.NoError(t, err)

		var loaded ZitiControllerConfig
		err = yaml.Unmarshal(data, &loaded)
		require.NoError(t, err)

		assert.Equal(t, 3, loaded.Version)
		assert.Equal(t, "ctrl.db", loaded.DB)
		assert.Equal(t, "ctrl-client.cert", loaded.Identity.Cert)
	})

	t.Run("invalid directory path", func(t *testing.T) {
		cfg := ZitiControllerConfig{Version: 3}

		err := WriteZitiControllerConfig("/nonexistent/dir/controller.yml", cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create file")
	})
}

// TestZitiGenerateCtrlConfig tests controller config generation
func TestZitiGenerateCtrlConfig(t *testing.T) {
	t.Run("successful generation", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "ctrl-generated.yml")

		err := ZitiGenerateCtrlConfig(filename)
		require.NoError(t, err)

		// Verify file exists
		assert.FileExists(t, filename)

		// Verify file content
		data, err := os.ReadFile(filename)
		require.NoError(t, err)

		var cfg ZitiControllerConfig
		err = yaml.Unmarshal(data, &cfg)
		require.NoError(t, err)

		assert.Equal(t, 3, cfg.Version)
		assert.Equal(t, "ctrl.db", cfg.DB)
		assert.Equal(t, "tls:127.0.0.1:6262", cfg.Ctrl.Listener)
		assert.Len(t, cfg.Web, 1)
		assert.Equal(t, "all-apis-localhost", cfg.Web[0].Name)
	})

	t.Run("invalid path", func(t *testing.T) {
		err := ZitiGenerateCtrlConfig("/nonexistent/dir/ctrl.yml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to")
	})
}

// TestZitiGenerateRouterConfig tests router config generation
func TestZitiGenerateRouterConfig(t *testing.T) {
	t.Run("successful generation", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "router-generated.yml")

		err := ZitiGenerateRouterConfig(filename)
		require.NoError(t, err)

		// Verify file exists
		assert.FileExists(t, filename)

		// Verify file content
		data, err := os.ReadFile(filename)
		require.NoError(t, err)

		var cfg ZitiRouterConfig
		err = yaml.Unmarshal(data, &cfg)
		require.NoError(t, err)

		assert.Equal(t, 3, cfg.Version)
		assert.Equal(t, "tls:127.0.0.1:6262", cfg.Ctrl.Endpoint)
		assert.Len(t, cfg.Dialers, 2)
		assert.Len(t, cfg.Listeners, 2)
		assert.Equal(t, "US", cfg.Edge.CSR.Country)
		assert.Equal(t, "Charlotte", cfg.Edge.CSR.Locality)
	})

	t.Run("invalid path", func(t *testing.T) {
		err := ZitiGenerateRouterConfig("/nonexistent/dir/router.yml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to")
	})
}

// TestZitiConfigStructures tests the config data structures
func TestZitiConfigStructures(t *testing.T) {
	t.Run("ZitiIdentity", func(t *testing.T) {
		id := ZitiIdentity{
			Cert:       "cert.pem",
			ServerCert: "server.pem",
			Key:        "key.pem",
			CA:         "ca.pem",
		}

		assert.Equal(t, "cert.pem", id.Cert)
		assert.Equal(t, "server.pem", id.ServerCert)
		assert.Equal(t, "key.pem", id.Key)
		assert.Equal(t, "ca.pem", id.CA)
	})

	t.Run("ZitiCSR", func(t *testing.T) {
		csr := ZitiCSR{
			Country:            "US",
			Province:           "NC",
			Locality:           "Charlotte",
			Organization:       "OpenZiti",
			OrganizationalUnit: "Ziti",
			SANs: ZitiSANs{
				DNS: []string{"localhost", "example.com"},
				IP:  []string{"127.0.0.1", "192.168.1.1"},
			},
		}

		assert.Equal(t, "US", csr.Country)
		assert.Equal(t, "Charlotte", csr.Locality)
		assert.Len(t, csr.SANs.DNS, 2)
		assert.Len(t, csr.SANs.IP, 2)
	})

	t.Run("ZitiLinkConfig", func(t *testing.T) {
		link := ZitiLinkConfig{
			Listeners: []ZitiLinkListener{
				{
					Binding:   "transport",
					Bind:      "tls:0.0.0.0:6000",
					Advertise: "tls:127.0.0.1:6000",
				},
			},
			Dialers: []ZitiDialer{
				{Binding: "transport"},
			},
		}

		assert.Len(t, link.Listeners, 1)
		assert.Len(t, link.Dialers, 1)
		assert.Equal(t, "transport", link.Listeners[0].Binding)
	})
}

// TestZitiConfigYAMLEncoding tests YAML encoding/decoding
func TestZitiConfigYAMLEncoding(t *testing.T) {
	t.Run("router config round-trip", func(t *testing.T) {
		original := ZitiRouterConfig{
			Version: 3,
			Identity: ZitiIdentity{
				Cert: "test.cert",
				Key:  "test.key",
				CA:   "ca.cert",
			},
			Ctrl: ZitiCtrlEndpoint{Endpoint: "tls:localhost:6262"},
		}

		// Encode to YAML
		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		// Decode from YAML
		var decoded ZitiRouterConfig
		err = yaml.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Version, decoded.Version)
		assert.Equal(t, original.Identity.Cert, decoded.Identity.Cert)
		assert.Equal(t, original.Ctrl.Endpoint, decoded.Ctrl.Endpoint)
	})

	t.Run("controller config round-trip", func(t *testing.T) {
		original := ZitiControllerConfig{
			Version: 3,
			DB:      "test.db",
			Identity: ZitiIdentity{
				Cert: "ctrl.cert",
				Key:  "ctrl.key",
			},
		}

		// Encode to YAML
		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		// Decode from YAML
		var decoded ZitiControllerConfig
		err = yaml.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Version, decoded.Version)
		assert.Equal(t, original.DB, decoded.DB)
		assert.Equal(t, original.Identity.Cert, decoded.Identity.Cert)
	})
}

// BenchmarkWriteZitiRouterConfig benchmarks router config writing
func BenchmarkWriteZitiRouterConfig(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := ZitiRouterConfig{
		Version: 3,
		Identity: ZitiIdentity{
			Cert: "router.cert",
			Key:  "router.key",
			CA:   "ca.cert",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filepath.Join(tmpDir, "router.yml")
		_ = WriteZitiRouterConfig(filename, cfg)
		os.Remove(filename)
	}
}

// BenchmarkWriteZitiControllerConfig benchmarks controller config writing
func BenchmarkWriteZitiControllerConfig(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := ZitiControllerConfig{
		Version: 3,
		DB:      "ctrl.db",
		Identity: ZitiIdentity{
			Cert: "ctrl.cert",
			Key:  "ctrl.key",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filepath.Join(tmpDir, "controller.yml")
		_ = WriteZitiControllerConfig(filename, cfg)
		os.Remove(filename)
	}
}

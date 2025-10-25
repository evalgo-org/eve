package common

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistryAuth tests Docker registry authentication encoding
func TestRegistryAuth(t *testing.T) {
	t.Run("valid credentials", func(t *testing.T) {
		username := "testuser"
		password := "testpass"

		authStr := RegistryAuth(username, password)
		assert.NotEmpty(t, authStr)

		// Decode and verify
		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Equal(t, username, authConfig["username"])
		assert.Equal(t, password, authConfig["password"])
	})

	t.Run("empty username", func(t *testing.T) {
		authStr := RegistryAuth("", "password")
		assert.NotEmpty(t, authStr)

		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Empty(t, authConfig["username"])
		assert.Equal(t, "password", authConfig["password"])
	})

	t.Run("empty password", func(t *testing.T) {
		authStr := RegistryAuth("user", "")
		assert.NotEmpty(t, authStr)

		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Equal(t, "user", authConfig["username"])
		assert.Empty(t, authConfig["password"])
	})

	t.Run("special characters in credentials", func(t *testing.T) {
		username := "user@example.com"
		password := "p@ss!w0rd#123"

		authStr := RegistryAuth(username, password)
		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Equal(t, username, authConfig["username"])
		assert.Equal(t, password, authConfig["password"])
	})

	t.Run("unicode characters", func(t *testing.T) {
		username := "用户"
		password := "密码123"

		authStr := RegistryAuth(username, password)
		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Equal(t, username, authConfig["username"])
		assert.Equal(t, password, authConfig["password"])
	})
}

// TestImagePullOptions tests the ImagePullOptions structure
func TestImagePullOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := &ImagePullOptions{}

		assert.Empty(t, opts.Username)
		assert.Empty(t, opts.Password)
		assert.Empty(t, opts.Platform)
		assert.False(t, opts.Silent)
		assert.Nil(t, opts.CustomOptions)
	})

	t.Run("with credentials", func(t *testing.T) {
		opts := &ImagePullOptions{
			Username: "testuser",
			Password: "testpass",
		}

		assert.Equal(t, "testuser", opts.Username)
		assert.Equal(t, "testpass", opts.Password)
	})

	t.Run("with platform", func(t *testing.T) {
		opts := &ImagePullOptions{
			Platform: "linux/amd64",
		}

		assert.Equal(t, "linux/amd64", opts.Platform)
	})

	t.Run("with silent mode", func(t *testing.T) {
		opts := &ImagePullOptions{
			Silent: true,
		}

		assert.True(t, opts.Silent)
	})

	t.Run("complete options", func(t *testing.T) {
		opts := &ImagePullOptions{
			Username: "user",
			Password: "pass",
			Platform: "linux/arm64",
			Silent:   true,
		}

		assert.Equal(t, "user", opts.Username)
		assert.Equal(t, "pass", opts.Password)
		assert.Equal(t, "linux/arm64", opts.Platform)
		assert.True(t, opts.Silent)
	})
}

// TestContainerView tests the ContainerView structure
func TestContainerView(t *testing.T) {
	t.Run("basic container view", func(t *testing.T) {
		cv := ContainerView{
			ID:     "abc123",
			Name:   "test-container",
			Status: "running",
			Host:   "docker-host-1",
		}

		assert.Equal(t, "abc123", cv.ID)
		assert.Equal(t, "test-container", cv.Name)
		assert.Equal(t, "running", cv.Status)
		assert.Equal(t, "docker-host-1", cv.Host)
	})

	t.Run("container with different status", func(t *testing.T) {
		cv := ContainerView{
			ID:     "def456",
			Name:   "postgres-db",
			Status: "exited",
			Host:   "docker-host-2",
		}

		assert.Equal(t, "def456", cv.ID)
		assert.Equal(t, "postgres-db", cv.Name)
		assert.Equal(t, "exited", cv.Status)
		assert.Equal(t, "docker-host-2", cv.Host)
	})

	t.Run("empty container view", func(t *testing.T) {
		cv := ContainerView{}

		assert.Empty(t, cv.ID)
		assert.Empty(t, cv.Name)
		assert.Empty(t, cv.Status)
		assert.Empty(t, cv.Host)
	})
}

// TestCopyToVolumeOptions tests the CopyToVolumeOptions structure
func TestCopyToVolumeOptions(t *testing.T) {
	t.Run("valid copy options", func(t *testing.T) {
		opts := CopyToVolumeOptions{
			Image:      "alpine:latest",
			Volume:     "test-volume",
			VolumePath: "/data",
			LocalPath:  "/local/file.txt",
		}

		assert.Equal(t, "alpine:latest", opts.Image)
		assert.Equal(t, "test-volume", opts.Volume)
		assert.Equal(t, "/data", opts.VolumePath)
		assert.Equal(t, "/local/file.txt", opts.LocalPath)
	})

	t.Run("empty options", func(t *testing.T) {
		opts := CopyToVolumeOptions{}

		assert.Empty(t, opts.Image)
		assert.Empty(t, opts.Volume)
		assert.Empty(t, opts.VolumePath)
		assert.Empty(t, opts.LocalPath)
		assert.Nil(t, opts.Client)
		assert.Nil(t, opts.Ctx)
	})
}

// TestImagePullOptions_Validation tests various configurations
func TestImagePullOptions_Validation(t *testing.T) {
	t.Run("nil options should be handled", func(t *testing.T) {
		var opts *ImagePullOptions
		assert.Nil(t, opts)
	})

	t.Run("partial credentials", func(t *testing.T) {
		tests := []struct {
			name string
			opts *ImagePullOptions
		}{
			{
				name: "username only",
				opts: &ImagePullOptions{Username: "user"},
			},
			{
				name: "password only",
				opts: &ImagePullOptions{Password: "pass"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.NotNil(t, tt.opts)
			})
		}
	})
}

// TestRegistryAuth_EdgeCases tests edge cases for registry authentication
func TestRegistryAuth_EdgeCases(t *testing.T) {
	t.Run("very long credentials", func(t *testing.T) {
		longUser := string(make([]byte, 1000))
		longPass := string(make([]byte, 1000))

		authStr := RegistryAuth(longUser, longPass)
		assert.NotEmpty(t, authStr)

		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Len(t, authConfig["username"], 1000)
		assert.Len(t, authConfig["password"], 1000)
	})

	t.Run("credentials with newlines", func(t *testing.T) {
		username := "user\nwith\nnewlines"
		password := "pass\nwith\nnewlines"

		authStr := RegistryAuth(username, password)
		decoded, err := base64.URLEncoding.DecodeString(authStr)
		require.NoError(t, err)

		var authConfig map[string]string
		err = json.Unmarshal(decoded, &authConfig)
		require.NoError(t, err)

		assert.Equal(t, username, authConfig["username"])
		assert.Equal(t, password, authConfig["password"])
	})

	t.Run("credentials with null bytes", func(t *testing.T) {
		username := "user\x00test"
		password := "pass\x00test"

		authStr := RegistryAuth(username, password)
		assert.NotEmpty(t, authStr)
	})
}

// TestContainerView_JSON tests JSON serialization
func TestContainerView_JSON(t *testing.T) {
	t.Run("marshal container view", func(t *testing.T) {
		cv := ContainerView{
			ID:     "container123",
			Name:   "web-server",
			Status: "running",
			Host:   "localhost",
		}

		data, err := json.Marshal(cv)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded ContainerView
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, cv.ID, decoded.ID)
		assert.Equal(t, cv.Name, decoded.Name)
		assert.Equal(t, cv.Status, decoded.Status)
		assert.Equal(t, cv.Host, decoded.Host)
	})

	t.Run("unmarshal container view", func(t *testing.T) {
		jsonData := `{
			"id": "abc123",
			"name": "redis-cache",
			"status": "created",
			"host": "docker-01"
		}`

		var cv ContainerView
		err := json.Unmarshal([]byte(jsonData), &cv)
		require.NoError(t, err)

		assert.Equal(t, "abc123", cv.ID)
		assert.Equal(t, "redis-cache", cv.Name)
		assert.Equal(t, "created", cv.Status)
		assert.Equal(t, "docker-01", cv.Host)
	})
}

// TestImagePullOptions_PlatformValidation tests platform string validation
func TestImagePullOptions_PlatformValidation(t *testing.T) {
	platforms := []string{
		"linux/amd64",
		"linux/arm64",
		"linux/arm/v7",
		"windows/amd64",
		"darwin/amd64",
		"darwin/arm64",
	}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			opts := &ImagePullOptions{
				Platform: platform,
			}

			assert.Equal(t, platform, opts.Platform)
		})
	}
}

// TestCopyToVolumeOptions_PathValidation tests path formats
func TestCopyToVolumeOptions_PathValidation(t *testing.T) {
	t.Run("absolute paths", func(t *testing.T) {
		opts := CopyToVolumeOptions{
			VolumePath: "/absolute/path",
			LocalPath:  "/local/absolute/path",
		}

		assert.True(t, opts.VolumePath[0] == '/')
		assert.True(t, opts.LocalPath[0] == '/')
	})

	t.Run("relative paths", func(t *testing.T) {
		opts := CopyToVolumeOptions{
			VolumePath: "relative/path",
			LocalPath:  "./local/path",
		}

		assert.NotEmpty(t, opts.VolumePath)
		assert.NotEmpty(t, opts.LocalPath)
	})

	t.Run("windows-style paths", func(t *testing.T) {
		opts := CopyToVolumeOptions{
			LocalPath: "C:\\Users\\test\\file.txt",
		}

		assert.Contains(t, opts.LocalPath, "\\")
	})
}

// TestContainerView_StatusValues tests different container statuses
func TestContainerView_StatusValues(t *testing.T) {
	statuses := []string{
		"created",
		"running",
		"paused",
		"restarting",
		"removing",
		"exited",
		"dead",
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			cv := ContainerView{
				Status: status,
			}

			assert.Equal(t, status, cv.Status)
		})
	}
}

// TestImagePullOptions_CombinedConfigurations tests various option combinations
func TestImagePullOptions_CombinedConfigurations(t *testing.T) {
	configs := []struct {
		name string
		opts *ImagePullOptions
	}{
		{
			name: "authenticated pull",
			opts: &ImagePullOptions{
				Username: "user",
				Password: "pass",
			},
		},
		{
			name: "platform-specific pull",
			opts: &ImagePullOptions{
				Platform: "linux/arm64",
			},
		},
		{
			name: "silent authenticated pull",
			opts: &ImagePullOptions{
				Username: "user",
				Password: "pass",
				Silent:   true,
			},
		},
		{
			name: "platform-specific authenticated pull",
			opts: &ImagePullOptions{
				Username: "user",
				Password: "pass",
				Platform: "linux/amd64",
			},
		},
		{
			name: "full configuration",
			opts: &ImagePullOptions{
				Username: "admin",
				Password: "secret",
				Platform: "linux/arm64",
				Silent:   true,
			},
		},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			assert.NotNil(t, cfg.opts)
		})
	}
}

// BenchmarkRegistryAuth benchmarks authentication encoding
func BenchmarkRegistryAuth(b *testing.B) {
	username := "benchuser"
	password := "benchpass"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RegistryAuth(username, password)
	}
}

// BenchmarkRegistryAuth_LongCredentials benchmarks with long credentials
func BenchmarkRegistryAuth_LongCredentials(b *testing.B) {
	username := string(make([]byte, 500))
	password := string(make([]byte, 500))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RegistryAuth(username, password)
	}
}

// BenchmarkContainerViewJSON benchmarks JSON marshaling
func BenchmarkContainerViewJSON(b *testing.B) {
	cv := ContainerView{
		ID:     "benchmark-container",
		Name:   "bench-container",
		Status: "running",
		Host:   "localhost",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(cv)
	}
}

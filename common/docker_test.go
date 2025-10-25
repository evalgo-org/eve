package common

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// TestParseEnvFile tests environment file parsing
func TestParseEnvFile(t *testing.T) {
	t.Run("valid env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `# This is a comment
DATABASE_URL=postgres://localhost:5432/mydb
API_KEY=secret123
PORT=8080

# Another comment
DEBUG=true`

		err := os.WriteFile(envFile, []byte(content), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)

		expected := []string{
			"DATABASE_URL=postgres://localhost:5432/mydb",
			"API_KEY=secret123",
			"PORT=8080",
			"DEBUG=true",
		}

		assert.Equal(t, expected, envs)
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		err := os.WriteFile(envFile, []byte(""), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)
		assert.Empty(t, envs)
	})

	t.Run("only comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `# Comment 1
# Comment 2
# Comment 3`

		err := os.WriteFile(envFile, []byte(content), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)
		assert.Empty(t, envs)
	})

	t.Run("mixed empty lines and comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `
# Header comment

VAR1=value1

# Middle comment

VAR2=value2

`

		err := os.WriteFile(envFile, []byte(content), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)

		expected := []string{"VAR1=value1", "VAR2=value2"}
		assert.Equal(t, expected, envs)
	})

	t.Run("special characters in values", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `URL=https://example.com/path?query=value&other=test
PASSWORD=p@ss!word#123
JSON={"key":"value"}`

		err := os.WriteFile(envFile, []byte(content), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)

		assert.Len(t, envs, 3)
		assert.Contains(t, envs, `URL=https://example.com/path?query=value&other=test`)
		assert.Contains(t, envs, `PASSWORD=p@ss!word#123`)
		assert.Contains(t, envs, `JSON={"key":"value"}`)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		envs, err := parseEnvFile("/nonexistent/path/.env")
		assert.Error(t, err)
		assert.Nil(t, envs)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `  VAR1=value1
	VAR2=value2
VAR3=value3`

		err := os.WriteFile(envFile, []byte(content), 0644)
		require.NoError(t, err)

		envs, err := parseEnvFile(envFile)
		require.NoError(t, err)

		// Verify whitespace is trimmed
		expected := []string{"VAR1=value1", "VAR2=value2", "VAR3=value3"}
		assert.Equal(t, expected, envs)
	})
}

// TestAddFileToTar tests adding files to tar archives
func TestAddFileToTar(t *testing.T) {
	t.Run("add single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!"

		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		defer tw.Close()

		err = addFileToTar(tw, testFile, tmpDir, "dest")
		require.NoError(t, err)

		// Close writer to flush
		tw.Close()

		// Read tar and verify
		tr := tar.NewReader(buf)
		header, err := tr.Next()
		require.NoError(t, err)

		assert.Equal(t, "dest/test.txt", filepath.ToSlash(header.Name))
		assert.Equal(t, int64(len(content)), header.Size)

		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("add file with subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		testFile := filepath.Join(subDir, "nested.txt")
		content := "nested content"

		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		defer tw.Close()

		err = addFileToTar(tw, testFile, tmpDir, "target")
		require.NoError(t, err)

		tw.Close()

		// Verify tar contents
		tr := tar.NewReader(buf)
		header, err := tr.Next()
		require.NoError(t, err)

		assert.Equal(t, "target/subdir/nested.txt", filepath.ToSlash(header.Name))
	})

	t.Run("nonexistent file error", func(t *testing.T) {
		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		defer tw.Close()

		err := addFileToTar(tw, "/nonexistent/file.txt", "/tmp", "dest")
		assert.Error(t, err)
	})
}

// TestCreateTarArchive tests tar archive creation
func TestCreateTarArchive(t *testing.T) {
	t.Run("archive single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "file.txt")
		content := "test content"

		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		buf, err := createTarArchive(testFile, "destination")
		require.NoError(t, err)
		assert.NotNil(t, buf)

		// Verify tar contents
		tr := tar.NewReader(buf)
		header, err := tr.Next()
		require.NoError(t, err)

		// When archiving a single file, it gets placed under destFileName/filename
		assert.Equal(t, "destination/file.txt", filepath.ToSlash(header.Name))
		assert.Equal(t, int64(len(content)), header.Size)

		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("archive directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create directory structure
		err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
		require.NoError(t, err)

		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "subdir", "file2.txt")

		err = os.WriteFile(file1, []byte("content1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("content2"), 0644)
		require.NoError(t, err)

		buf, err := createTarArchive(tmpDir, "archive")
		require.NoError(t, err)
		assert.NotNil(t, buf)

		// Verify tar contains both files
		tr := tar.NewReader(buf)
		fileCount := 0
		fileNames := []string{}

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			fileCount++
			fileNames = append(fileNames, filepath.ToSlash(header.Name))
		}

		assert.Equal(t, 2, fileCount)
		assert.Contains(t, fileNames, "archive/file1.txt")
		assert.Contains(t, fileNames, "archive/subdir/file2.txt")
	})

	t.Run("archive empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		buf, err := createTarArchive(tmpDir, "empty")
		require.NoError(t, err)
		assert.NotNil(t, buf)

		// Verify tar is valid but empty
		tr := tar.NewReader(buf)
		_, err = tr.Next()
		assert.Equal(t, io.EOF, err)
	})

	t.Run("nonexistent path error", func(t *testing.T) {
		buf, err := createTarArchive("/nonexistent/path", "dest")
		assert.Error(t, err)
		assert.Nil(t, buf)
	})

	t.Run("archive with multiple nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create nested structure
		dirs := []string{
			filepath.Join(tmpDir, "a"),
			filepath.Join(tmpDir, "a", "b"),
			filepath.Join(tmpDir, "c"),
		}

		for _, dir := range dirs {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		// Create files in different locations
		files := []string{
			filepath.Join(tmpDir, "root.txt"),
			filepath.Join(tmpDir, "a", "a.txt"),
			filepath.Join(tmpDir, "a", "b", "b.txt"),
			filepath.Join(tmpDir, "c", "c.txt"),
		}

		for _, file := range files {
			err := os.WriteFile(file, []byte("content"), 0644)
			require.NoError(t, err)
		}

		buf, err := createTarArchive(tmpDir, "nested")
		require.NoError(t, err)

		// Count files in archive
		tr := tar.NewReader(buf)
		fileCount := 0

		for {
			_, err := tr.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			fileCount++
		}

		assert.Equal(t, 4, fileCount)
	})
}

// BenchmarkParseEnvFile benchmarks environment file parsing
func BenchmarkParseEnvFile(b *testing.B) {
	tmpDir := b.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := strings.Repeat("VAR=value\n", 100)
	os.WriteFile(envFile, []byte(content), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseEnvFile(envFile)
	}
}

// BenchmarkCreateTarArchive benchmarks tar archive creation
func BenchmarkCreateTarArchive(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("benchmark content"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = createTarArchive(testFile, "dest")
	}
}

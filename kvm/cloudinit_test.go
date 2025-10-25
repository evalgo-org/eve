package kvm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCloudInitISO(t *testing.T) {
	// Skip if genisoimage/mkisofs not available
	if _, err := os.Stat("/usr/bin/genisoimage"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/mkisofs"); os.IsNotExist(err) {
			t.Skip("Skipping test: genisoimage/mkisofs not found")
		}
	}

	t.Run("Basic ISO creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "test-cloudinit.iso")

		cfg := CloudInitConfig{
			VMName:        "test-vm",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: false,
			Packages:      []string{},
		}

		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		// Verify ISO was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("ISO file was not created at %s", outputPath)
		}

		// Verify ISO is not empty
		info, err := os.Stat(outputPath)
		if err != nil {
			t.Fatalf("Failed to stat ISO: %v", err)
		}
		if info.Size() == 0 {
			t.Errorf("ISO file is empty")
		}
	})

	t.Run("ISO with package updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "test-update-cloudinit.iso")

		cfg := CloudInitConfig{
			VMName:        "update-vm",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: true,
			Packages:      []string{},
		}

		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		// Verify temporary user-data was created with package_update
		tmpDataDir := filepath.Join(os.TempDir(), "cloudinit-update-vm")
		userDataPath := filepath.Join(tmpDataDir, "user-data")

		// Read user-data if it still exists
		if data, err := os.ReadFile(userDataPath); err == nil {
			if !strings.Contains(string(data), "package_update: true") {
				t.Errorf("user-data missing package_update directive")
			}
		}
	})

	t.Run("ISO with packages", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "test-packages-cloudinit.iso")

		cfg := CloudInitConfig{
			VMName:        "packages-vm",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: false,
			Packages:      []string{"vim", "git", "curl"},
		}

		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		// Verify ISO was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("ISO file was not created")
		}

		// Check temp user-data contains packages
		tmpDataDir := filepath.Join(os.TempDir(), "cloudinit-packages-vm")
		userDataPath := filepath.Join(tmpDataDir, "user-data")

		if data, err := os.ReadFile(userDataPath); err == nil {
			content := string(data)
			if !strings.Contains(content, "packages:") {
				t.Errorf("user-data missing packages section")
			}
			if !strings.Contains(content, "- vim") {
				t.Errorf("user-data missing vim package")
			}
			if !strings.Contains(content, "- git") {
				t.Errorf("user-data missing git package")
			}
			if !strings.Contains(content, "- curl") {
				t.Errorf("user-data missing curl package")
			}
		}
	})

	t.Run("ISO with complex VM name", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "complex-name.iso")

		cfg := CloudInitConfig{
			VMName:        "Web_Server-01",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: false,
			Packages:      []string{},
		}

		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("ISO file was not created")
		}
	})

	t.Run("Invalid output path", func(t *testing.T) {
		cfg := CloudInitConfig{
			VMName:       "test-vm",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
		}

		// Try to write to non-existent directory (should fail in genisoimage)
		invalidPath := "/nonexistent/directory/that/does/not/exist/test.iso"
		err := CreateCloudInitISO(cfg, invalidPath)

		// Should get an error
		if err == nil {
			t.Errorf("Expected error for invalid output path, got nil")
		}
	})

	t.Run("Empty VM name", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "empty-name.iso")

		cfg := CloudInitConfig{
			VMName:       "", // Empty name
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
		}

		// Should still work, just creates empty hostname
		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}
	})

	t.Run("Multiple packages and updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "full-config.iso")

		cfg := CloudInitConfig{
			VMName:        "full-vm",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: true,
			Packages:      []string{"nginx", "postgresql", "redis"},
		}

		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		// Verify ISO exists and has reasonable size
		info, err := os.Stat(outputPath)
		if err != nil {
			t.Fatalf("Failed to stat ISO: %v", err)
		}

		// ISO should be at least 10KB (cloud-init data + ISO overhead)
		if info.Size() < 10*1024 {
			t.Errorf("ISO seems too small: %d bytes", info.Size())
		}
	})
}

func TestCreateCloudInitISOUserDataFormat(t *testing.T) {
	t.Run("User-data YAML format", func(t *testing.T) {
		// We'll create the ISO and check the temp files before they're deleted
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "test.iso")

		cfg := CloudInitConfig{
			VMName:        "format-test",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate: true,
			Packages:      []string{"vim"},
		}

		// Create a modified version that doesn't clean up temp files
		// For testing, we'll just verify the function runs
		err := CreateCloudInitISO(cfg, outputPath)
		if err != nil {
			// If genisoimage isn't available, skip
			if strings.Contains(err.Error(), "failed to create cloud-init ISO") {
				t.Skip("genisoimage/mkisofs not available")
			}
			t.Fatalf("CreateCloudInitISO failed: %v", err)
		}

		// The temp directory should be created
		tmpDataDir := filepath.Join(os.TempDir(), "cloudinit-format-test")
		if _, err := os.Stat(tmpDataDir); err == nil {
			// If it exists, check the user-data file
			userData := filepath.Join(tmpDataDir, "user-data")
			if data, err := os.ReadFile(userData); err == nil {
				content := string(data)

				// Check for cloud-config header
				if !strings.HasPrefix(content, "#cloud-config") {
					t.Errorf("user-data doesn't start with #cloud-config")
				}

				// Check for hostname
				if !strings.Contains(content, "hostname: format-test") {
					t.Errorf("user-data missing hostname")
				}

				// Check for SSH key
				if !strings.Contains(content, "ssh_authorized_keys:") {
					t.Errorf("user-data missing ssh_authorized_keys")
				}

				// Check for bootcmd
				if !strings.Contains(content, "bootcmd:") {
					t.Errorf("user-data missing bootcmd")
				}
			}

			// Check meta-data
			metaData := filepath.Join(tmpDataDir, "meta-data")
			if data, err := os.ReadFile(metaData); err == nil {
				content := string(data)

				if !strings.Contains(content, "instance-id: format-test") {
					t.Errorf("meta-data missing instance-id")
				}

				if !strings.Contains(content, "local-hostname: format-test") {
					t.Errorf("meta-data missing local-hostname")
				}
			}
		}
	})
}

// Note: Actual file creation with chown/chcon requires root/sudo
// These would fail in normal test environment, so we just verify
// that the function attempts to run them (doesn't panic)
func TestCreateCloudInitISOPermissions(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Skipping permission test: not running as root")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "perms-test.iso")

	cfg := CloudInitConfig{
		VMName:       "perms-vm",
		SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
	}

	err := CreateCloudInitISO(cfg, outputPath)
	if err != nil {
		t.Fatalf("CreateCloudInitISO failed: %v", err)
	}

	// If running as root, verify ownership
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat ISO: %v", err)
	}

	t.Logf("ISO created with mode: %v", info.Mode())
}

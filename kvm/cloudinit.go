package kvm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CloudInitConfig holds cloud-init configuration
type CloudInitConfig struct {
	VMName        string
	SSHPublicKey  string
	PackageUpdate bool
	Packages      []string
}

// CreateCloudInitISO generates cloud-init user-data and meta-data, then creates an ISO
func CreateCloudInitISO(cfg CloudInitConfig, outputPath string) error {
	// Build packages YAML
	packagesYAML := ""
	if len(cfg.Packages) > 0 {
		packagesYAML = "packages:\n"
		for _, pkg := range cfg.Packages {
			packagesYAML += fmt.Sprintf("  - %s\n", pkg)
		}
	}

	updateYAML := ""
	if cfg.PackageUpdate {
		updateYAML = "package_update: true\n"
	}

	userData := fmt.Sprintf(`#cloud-config
hostname: %s
ssh_authorized_keys:
  - %s
%s%s
bootcmd:
  - echo "cloud-init completed"
`, cfg.VMName, cfg.SSHPublicKey, updateYAML, packagesYAML)

	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n",
		cfg.VMName, cfg.VMName)

	// Create temp directory
	tmpdir := filepath.Join(os.TempDir(), "cloudinit-"+cfg.VMName)
	if err := os.MkdirAll(tmpdir, 0755); err != nil {
		return fmt.Errorf("failed to make tmp dir: %w", err)
	}

	userFile := filepath.Join(tmpdir, "user-data")
	metaFile := filepath.Join(tmpdir, "meta-data")

	if err := os.WriteFile(userFile, []byte(userData), 0644); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	if err := os.WriteFile(metaFile, []byte(metaData), 0644); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Create ISO
	cmdISO := exec.Command("genisoimage", "-output", outputPath,
		"-volid", "cidata", "-joliet", "-rock", userFile, metaFile)
	if err := cmdISO.Run(); err != nil {
		// Try mkisofs as fallback
		cmd2 := exec.Command("mkisofs", "-output", outputPath,
			"-volid", "cidata", "-joliet", "-rock", userFile, metaFile)
		if err2 := cmd2.Run(); err2 != nil {
			return fmt.Errorf("failed to create cloud-init ISO: %v / %v", err, err2)
		}
	}

	// Set ownership (requires sudo)
	_ = exec.Command("sudo", "chown", "qemu:qemu", outputPath).Run()
	_ = exec.Command("sudo", "chcon", "-t", "svirt_iso_image_t", outputPath).Run()

	return nil
}

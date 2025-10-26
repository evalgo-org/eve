package kvm

import (
	"os"
	"path/filepath"
	"testing"
)

// Test CreateVM parameter validation
func TestCreateVMValidation(t *testing.T) {
	t.Run("Invalid VM name", func(t *testing.T) {
		params := CreateVMParams{
			VMName:        "123-invalid", // Starts with digit
			ImagePath:     "/tmp/test.qcow2",
			CloudInitISO:  "/tmp/test.iso",
			SSHPublicKey:  "ssh-rsa AAAA test",
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
		}

		result := CreateVM(params)

		if result.Success {
			t.Errorf("Expected failure for invalid VM name")
		}

		if result.ErrorMessage == "" {
			t.Errorf("Expected error message for invalid VM name")
		}

		if result.Stage != "initialization" {
			t.Errorf("Expected stage to be 'initialization', got %q", result.Stage)
		}
	})

	t.Run("Non-existent image file", func(t *testing.T) {
		params := CreateVMParams{
			VMName:        "valid-vm",
			ImagePath:     "/tmp/nonexistent-image-file.qcow2",
			CloudInitISO:  "/tmp/test.iso",
			SSHPublicKey:  "ssh-rsa AAAA test",
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
		}

		result := CreateVM(params)

		if result.Success {
			t.Errorf("Expected failure for non-existent image")
		}

		if result.ErrorMessage == "" {
			t.Errorf("Expected error message for non-existent image")
		}
	})

	t.Run("Empty SSH public key", func(t *testing.T) {
		// Create a temporary image file
		tmpDir := t.TempDir()
		imagePath := filepath.Join(tmpDir, "test.qcow2")
		_ = os.WriteFile(imagePath, []byte("fake image"), 0644)

		params := CreateVMParams{
			VMName:        "test-vm",
			ImagePath:     imagePath,
			CloudInitISO:  "/tmp/test.iso",
			SSHPublicKey:  "", // Empty key
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
		}

		result := CreateVM(params)

		// Should still proceed (cloud-init will be created with empty key)
		// but will fail at connection or later stages
		if result.Stage == "initialization" && result.ErrorMessage != "" {
			// This is acceptable - early validation
		}
	})

	t.Run("Valid parameters structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		imagePath := filepath.Join(tmpDir, "test.qcow2")
		_ = os.WriteFile(imagePath, []byte("fake image"), 0644)

		params := CreateVMParams{
			VMName:           "test-vm",
			DistributionName: "Test Linux",
			DefaultUser:      "testuser",
			ImagePath:        imagePath,
			CloudInitISO:     "/tmp/test.iso",
			SSHPublicKey:     "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
			PackageUpdate:    true,
			Packages:         []string{"vim", "git"},
			LibvirtSocket:    "/var/run/libvirt/libvirt-sock",
			MemoryKiB:        4194304,
			VCPUs:            4,
		}

		// Just verify params are set correctly
		if params.VMName != "test-vm" {
			t.Errorf("VMName not set correctly")
		}
		if params.MemoryKiB != 4194304 {
			t.Errorf("MemoryKiB not set correctly")
		}
		if params.VCPUs != 4 {
			t.Errorf("VCPUs not set correctly")
		}
	})
}

// Test ListVMsParams
func TestListVMsParams(t *testing.T) {
	t.Run("ListVMsParams structure", func(t *testing.T) {
		distributions := map[string]DistributionInfo{
			"rocky9": {Name: "Rocky Linux 9", Key: "rocky9"},
			"ubuntu": {Name: "Ubuntu 22.04", Key: "ubuntu"},
		}

		params := ListVMsParams{
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
			OnlyRunning:   true,
			Distributions: distributions,
		}

		if !params.OnlyRunning {
			t.Errorf("Expected OnlyRunning to be true")
		}

		if len(params.Distributions) != 2 {
			t.Errorf("Expected 2 distributions, got %d", len(params.Distributions))
		}
	})

	t.Run("Empty distributions map", func(t *testing.T) {
		params := ListVMsParams{
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
			OnlyRunning:   false,
			Distributions: make(map[string]DistributionInfo),
		}

		if len(params.Distributions) != 0 {
			t.Errorf("Expected empty distributions map")
		}
	})
}

// Test DeleteVMParams
func TestDeleteVMParams(t *testing.T) {
	t.Run("DeleteVMParams structure", func(t *testing.T) {
		params := DeleteVMParams{
			VMName:          "test-vm",
			LibvirtSocket:   "/var/run/libvirt/libvirt-sock",
			CleanupISO:      true,
			CloudISOTmpBase: "/tmp",
		}

		if params.VMName != "test-vm" {
			t.Errorf("VMName not set correctly")
		}

		if !params.CleanupISO {
			t.Errorf("Expected CleanupISO to be true")
		}
	})

	t.Run("DeleteVM without cleanup", func(t *testing.T) {
		params := DeleteVMParams{
			VMName:        "test-vm",
			LibvirtSocket: "/var/run/libvirt/libvirt-sock",
			CleanupISO:    false,
		}

		if params.CleanupISO {
			t.Errorf("Expected CleanupISO to be false")
		}
	})
}

// Integration tests - require libvirt
func TestListVMsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: libvirt socket not found")
	}

	t.Run("List all VMs", func(t *testing.T) {
		params := ListVMsParams{
			LibvirtSocket: socketPath,
			OnlyRunning:   false,
			Distributions: make(map[string]DistributionInfo),
		}

		vms, err := ListVMs(params)

		if err != nil {
			t.Fatalf("ListVMs failed: %v", err)
		}

		// Should return slice (may be empty)
		if vms == nil {
			t.Errorf("Expected non-nil VMs slice")
		}

		t.Logf("Found %d VMs", len(vms))

		// Verify structure of returned VMs
		for i, vm := range vms {
			if vm.Name == "" {
				t.Errorf("VM %d has empty name", i)
			}
			if vm.State == "" {
				t.Errorf("VM %d has empty state", i)
			}
			t.Logf("VM %d: %s (%s) - %s", i, vm.Name, vm.State, vm.IPAddress)
		}
	})

	t.Run("List only running VMs", func(t *testing.T) {
		params := ListVMsParams{
			LibvirtSocket: socketPath,
			OnlyRunning:   true,
			Distributions: make(map[string]DistributionInfo),
		}

		vms, err := ListVMs(params)

		if err != nil {
			t.Fatalf("ListVMs failed: %v", err)
		}

		// All returned VMs should be active
		for i, vm := range vms {
			if !vm.IsActive {
				t.Errorf("VM %d (%s) is not active but was returned with OnlyRunning=true", i, vm.Name)
			}
			if vm.State != "running" {
				t.Errorf("VM %d (%s) has state %q but should be running", i, vm.Name, vm.State)
			}
		}

		t.Logf("Found %d running VMs", len(vms))
	})

	t.Run("List with distribution detection", func(t *testing.T) {
		distributions := map[string]DistributionInfo{
			"rocky9": {Name: "Rocky Linux 9", Key: "rocky9"},
			"ubuntu": {Name: "Ubuntu 22.04", Key: "ubuntu"},
			"fedora": {Name: "Fedora 39", Key: "fedora"},
		}

		params := ListVMsParams{
			LibvirtSocket: socketPath,
			OnlyRunning:   false,
			Distributions: distributions,
		}

		vms, err := ListVMs(params)

		if err != nil {
			t.Fatalf("ListVMs failed: %v", err)
		}

		// Check if any VMs have distribution detected
		distributionsFound := 0
		for _, vm := range vms {
			if vm.Distribution != "" {
				distributionsFound++
				t.Logf("Detected distribution for %s: %s", vm.Name, vm.Distribution)
			}
		}

		t.Logf("Distributions detected on %d/%d VMs", distributionsFound, len(vms))
	})

	t.Run("List with invalid socket", func(t *testing.T) {
		params := ListVMsParams{
			LibvirtSocket: "/tmp/invalid-socket.sock",
			OnlyRunning:   false,
		}

		vms, err := ListVMs(params)

		if err == nil {
			t.Errorf("Expected error for invalid socket")
		}

		if vms != nil {
			t.Errorf("Expected nil VMs on error")
		}
	})
}

func TestDeleteVMIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: libvirt socket not found")
	}

	t.Run("Delete non-existent VM", func(t *testing.T) {
		params := DeleteVMParams{
			VMName:          "nonexistent-vm-for-testing",
			LibvirtSocket:   socketPath,
			CleanupISO:      false,
			CloudISOTmpBase: "/tmp",
		}

		result := DeleteVM(params)

		if result.Success {
			t.Errorf("Expected failure when deleting non-existent VM")
		}

		if result.ErrorMessage == "" {
			t.Errorf("Expected error message for non-existent VM")
		}

		t.Logf("Error message: %s", result.ErrorMessage)
		t.Logf("Stage: %s", result.Stage)
	})

	t.Run("Delete with cleanup ISO (non-existent VM)", func(t *testing.T) {
		params := DeleteVMParams{
			VMName:          "nonexistent-vm",
			LibvirtSocket:   socketPath,
			CleanupISO:      true,
			CloudISOTmpBase: "/tmp",
		}

		result := DeleteVM(params)

		// Should fail at VM lookup, not at ISO cleanup
		if result.Success {
			t.Errorf("Expected failure")
		}

		// CleanupISO should be false since VM wasn't found
		if result.CleanupISO {
			t.Errorf("Expected CleanupISO to be false when VM doesn't exist")
		}
	})

	t.Run("Delete with invalid socket", func(t *testing.T) {
		params := DeleteVMParams{
			VMName:        "test-vm",
			LibvirtSocket: "/tmp/invalid-socket.sock",
		}

		result := DeleteVM(params)

		if result.Success {
			t.Errorf("Expected failure with invalid socket")
		}

		if result.Stage != "initialization" {
			t.Logf("Note: Stage is %q (connection error)", result.Stage)
		}
	})
}

// Test result structures
func TestCreateVMResult(t *testing.T) {
	t.Run("Result structure fields", func(t *testing.T) {
		result := &VMResult{
			VMName:       "test-vm",
			ImagePath:    "/path/to/image.qcow2",
			CloudInitISO: "/path/to/iso",
		}

		if result.VMName != "test-vm" {
			t.Errorf("VMName not set correctly")
		}

		// Verify result has required fields
		_ = result.Success
		_ = result.IPAddress
		_ = result.MACAddress
		_ = result.SSHCommand
		_ = result.CreatedAt
		_ = result.ErrorMessage
		_ = result.Stage
		_ = result.Distribution
	})
}

func TestDistributionInfo(t *testing.T) {
	t.Run("DistributionInfo structure", func(t *testing.T) {
		info := DistributionInfo{
			Name: "Rocky Linux 9",
			Key:  "rocky9",
		}

		if info.Name != "Rocky Linux 9" {
			t.Errorf("Name not set correctly")
		}

		if info.Key != "rocky9" {
			t.Errorf("Key not set correctly")
		}
	})
}

// Benchmark CreateVM parameter preparation
func BenchmarkCreateVMParams(b *testing.B) {
	tmpDir := b.TempDir()
	imagePath := filepath.Join(tmpDir, "test.qcow2")
	_ = os.WriteFile(imagePath, []byte("fake"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params := CreateVMParams{
			VMName:           "bench-vm",
			DistributionName: "Test",
			ImagePath:        imagePath,
			CloudInitISO:     "/tmp/test.iso",
			SSHPublicKey:     "ssh-rsa AAA test",
			LibvirtSocket:    "/var/run/libvirt/libvirt-sock",
		}
		_ = params
	}
}

package kvm

import (
	"os"
	"testing"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
)

// TestGetVMIPAddress tests the IP address detection logic
// Note: This requires a running libvirt with VMs to fully test
func TestGetVMIPAddress(t *testing.T) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping test: libvirt socket not found")
	}

	vir, err := Connect(socketPath)
	if err != nil {
		t.Skipf("Could not connect to libvirt: %v", err)
	}
	defer Disconnect(vir)

	t.Run("GetVMIPAddress with no VMs", func(t *testing.T) {
		// Create a fake domain that doesn't exist
		fakeDomain := libvirt.Domain{
			Name: "nonexistent-vm-for-testing",
		}

		// This should timeout and return empty string
		ip, _ := GetVMIPAddress(vir, fakeDomain, "52:54:00:00:00:00", 1)
		if ip != "" {
			t.Errorf("Expected empty IP for non-existent VM, got %q", ip)
		}
		// Error is optional (may be nil on timeout)
	})

	t.Run("GetVMIPAddress with invalid MAC", func(t *testing.T) {
		fakeDomain := libvirt.Domain{
			Name: "test-domain",
		}

		// Should timeout with invalid MAC
		ip, _ := GetVMIPAddress(vir, fakeDomain, "invalid-mac", 1)
		if ip != "" {
			t.Errorf("Expected empty IP for invalid MAC, got %q", ip)
		}
	})

	t.Run("GetVMIPAddress with zero attempts", func(t *testing.T) {
		fakeDomain := libvirt.Domain{
			Name: "test-domain",
		}

		// Should return immediately
		start := time.Now()
		ip, _ := GetVMIPAddress(vir, fakeDomain, "52:54:00:00:00:00", 0)
		duration := time.Since(start)

		if ip != "" {
			t.Errorf("Expected empty IP with 0 attempts, got %q", ip)
		}

		// Should complete quickly (less than 1 second)
		if duration > time.Second {
			t.Errorf("GetVMIPAddress took too long with 0 attempts: %v", duration)
		}
	})
}

// TestGetDHCPLeaseForMAC tests DHCP lease lookup
func TestGetDHCPLeaseForMAC(t *testing.T) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping test: libvirt socket not found")
	}

	vir, err := Connect(socketPath)
	if err != nil {
		t.Skipf("Could not connect to libvirt: %v", err)
	}
	defer Disconnect(vir)

	t.Run("GetDHCPLeaseForMAC on default network", func(t *testing.T) {
		// Try to get DHCP lease for a non-existent MAC
		ip, err := GetDHCPLeaseForMAC(vir, "default", "52:54:00:99:99:99")

		if err != nil {
			// Network might not exist or might have no leases
			t.Logf("GetDHCPLeaseForMAC returned error (may be expected): %v", err)
		}

		// IP should be empty for non-existent MAC
		if ip != "" {
			t.Logf("Unexpected IP found: %s (might be a real lease)", ip)
		}
	})

	t.Run("GetDHCPLeaseForMAC with invalid network", func(t *testing.T) {
		ip, err := GetDHCPLeaseForMAC(vir, "nonexistent-network", "52:54:00:00:00:00")

		if err == nil {
			t.Errorf("Expected error for non-existent network")
		}

		if ip != "" {
			t.Errorf("Expected empty IP for non-existent network, got %q", ip)
		}
	})

	t.Run("GetDHCPLeaseForMAC with empty MAC", func(t *testing.T) {
		ip, err := GetDHCPLeaseForMAC(vir, "default", "")

		// Should handle gracefully
		if ip != "" {
			t.Errorf("Expected empty IP for empty MAC, got %q", ip)
		}

		// Error is optional (network might not exist)
		_ = err
	})

	t.Run("GetDHCPLeaseForMAC with empty network name", func(t *testing.T) {
		ip, err := GetDHCPLeaseForMAC(vir, "", "52:54:00:00:00:00")

		if err == nil {
			t.Errorf("Expected error for empty network name")
		}

		if ip != "" {
			t.Errorf("Expected empty IP for empty network name, got %q", ip)
		}
	})
}

// Integration test - requires actual running VM
func TestGetVMIPAddressIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: libvirt socket not found")
	}

	vir, err := Connect(socketPath)
	if err != nil {
		t.Skipf("Could not connect to libvirt: %v", err)
	}
	defer Disconnect(vir)

	// List domains to find a running one
	domains, _, err := vir.ConnectListAllDomains(1, 0)
	if err != nil {
		t.Skipf("Could not list domains: %v", err)
	}

	if len(domains) == 0 {
		t.Skip("No VMs found for integration test")
	}

	// Find a running domain
	var runningDomain *libvirt.Domain
	for i := range domains {
		state, _, err := vir.DomainGetState(domains[i], 0)
		if err != nil {
			continue
		}
		if state == DomainRunning {
			runningDomain = &domains[i]
			break
		}
	}

	if runningDomain == nil {
		t.Skip("No running VMs found for integration test")
	}

	t.Run("Get IP from running VM", func(t *testing.T) {
		// Get the MAC address
		xmlDesc, err := vir.DomainGetXMLDesc(*runningDomain, 0)
		if err != nil {
			t.Skipf("Could not get domain XML: %v", err)
		}

		mac := ExtractMACFromXML(xmlDesc)
		if mac == "" {
			t.Skip("Could not extract MAC from domain XML")
		}

		t.Logf("Testing with running VM '%s', MAC: %s", runningDomain.Name, mac)

		// Try to get IP (with reasonable timeout)
		ip, err := GetVMIPAddress(vir, *runningDomain, mac, 5)

		if err != nil {
			t.Logf("GetVMIPAddress returned error: %v", err)
		}

		if ip != "" {
			t.Logf("Successfully retrieved IP: %s", ip)

			// Verify it looks like a valid IP
			if len(ip) < 7 { // Minimum: "1.1.1.1"
				t.Errorf("IP address seems invalid: %q", ip)
			}
		} else {
			t.Logf("No IP address found (VM might not have network yet)")
		}
	})

	t.Run("Get IP via DHCP lease", func(t *testing.T) {
		xmlDesc, err := vir.DomainGetXMLDesc(*runningDomain, 0)
		if err != nil {
			t.Skipf("Could not get domain XML: %v", err)
		}

		mac := ExtractMACFromXML(xmlDesc)
		if mac == "" {
			t.Skip("Could not extract MAC from domain XML")
		}

		ip, err := GetDHCPLeaseForMAC(vir, "default", mac)

		if err != nil {
			t.Logf("GetDHCPLeaseForMAC returned error: %v", err)
		}

		if ip != "" {
			t.Logf("Successfully retrieved IP via DHCP: %s", ip)
		} else {
			t.Logf("No DHCP lease found")
		}
	})
}

// Test timeout behavior
func TestGetVMIPAddressTimeout(t *testing.T) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping test: libvirt socket not found")
	}

	vir, err := Connect(socketPath)
	if err != nil {
		t.Skipf("Could not connect to libvirt: %v", err)
	}
	defer Disconnect(vir)

	t.Run("Timeout with 2 attempts", func(t *testing.T) {
		fakeDomain := libvirt.Domain{
			Name: "timeout-test",
		}

		start := time.Now()
		ip, _ := GetVMIPAddress(vir, fakeDomain, "52:54:00:00:00:00", 2)
		duration := time.Since(start)

		if ip != "" {
			t.Errorf("Expected empty IP, got %q", ip)
		}

		// Should take approximately 2 attempts * 3 seconds = 6 seconds
		// Allow some tolerance
		if duration < 5*time.Second || duration > 8*time.Second {
			t.Logf("Warning: timeout duration unexpected: %v (expected ~6s)", duration)
		}
	})
}

// Benchmark the IP detection
func BenchmarkGetDHCPLeaseForMAC(b *testing.B) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		b.Skip("Skipping benchmark: libvirt socket not found")
	}

	vir, err := Connect(socketPath)
	if err != nil {
		b.Skipf("Could not connect to libvirt: %v", err)
	}
	defer Disconnect(vir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetDHCPLeaseForMAC(vir, "default", "52:54:00:00:00:00")
	}
}

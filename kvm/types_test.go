package kvm

import (
	"testing"
	"time"
)

func TestStateToString(t *testing.T) {
	tests := []struct {
		name     string
		state    int32
		expected string
	}{
		{
			name:     "No state",
			state:    DomainNoState,
			expected: "unknown (0)",
		},
		{
			name:     "Running state",
			state:    DomainRunning,
			expected: "running",
		},
		{
			name:     "Blocked state",
			state:    DomainBlocked,
			expected: "unknown (2)",
		},
		{
			name:     "Paused state",
			state:    DomainPaused,
			expected: "paused",
		},
		{
			name:     "Shutoff state",
			state:    DomainShutoff,
			expected: "shut off",
		},
		{
			name:     "Crashed state",
			state:    DomainCrashed,
			expected: "crashed",
		},
		{
			name:     "Unknown state (99)",
			state:    99,
			expected: "unknown (99)",
		},
		{
			name:     "Negative state",
			state:    -1,
			expected: "unknown (-1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StateToString(tt.state)
			if result != tt.expected {
				t.Errorf("StateToString(%d) = %q, want %q", tt.state, result, tt.expected)
			}
		})
	}
}

func TestVMResult(t *testing.T) {
	t.Run("VMResult structure", func(t *testing.T) {
		now := time.Now()
		result := VMResult{
			Success:      true,
			VMName:       "test-vm",
			IPAddress:    "192.168.122.100",
			MACAddress:   "52:54:00:12:34:56",
			ImagePath:    "/var/lib/libvirt/images/test.qcow2",
			CloudInitISO: "/tmp/test-cloudinit.iso",
			SSHCommand:   "ssh user@192.168.122.100",
			CreatedAt:    now,
			ErrorMessage: "",
			Stage:        "completed",
			Distribution: "Rocky Linux 9",
		}

		if result.Success != true {
			t.Errorf("Expected Success to be true")
		}
		if result.VMName != "test-vm" {
			t.Errorf("Expected VMName to be 'test-vm', got %q", result.VMName)
		}
		if result.IPAddress != "192.168.122.100" {
			t.Errorf("Expected IPAddress to be '192.168.122.100', got %q", result.IPAddress)
		}
		if result.Stage != "completed" {
			t.Errorf("Expected Stage to be 'completed', got %q", result.Stage)
		}
	})

	t.Run("VMResult failure", func(t *testing.T) {
		result := VMResult{
			Success:      false,
			ErrorMessage: "Connection failed",
			Stage:        "initialization",
		}

		if result.Success != false {
			t.Errorf("Expected Success to be false")
		}
		if result.ErrorMessage == "" {
			t.Errorf("Expected ErrorMessage to be set")
		}
	})
}

func TestVMInfo(t *testing.T) {
	t.Run("VMInfo structure", func(t *testing.T) {
		info := VMInfo{
			Name:         "my-vm",
			State:        "running",
			IPAddress:    "192.168.122.50",
			Distribution: "Ubuntu 22.04",
			IsActive:     true,
			CreatedAt:    time.Now(),
		}

		if info.Name != "my-vm" {
			t.Errorf("Expected Name to be 'my-vm', got %q", info.Name)
		}
		if !info.IsActive {
			t.Errorf("Expected IsActive to be true")
		}
	})
}

func TestDeleteResult(t *testing.T) {
	t.Run("DeleteResult successful", func(t *testing.T) {
		result := DeleteResult{
			Success:    true,
			VMName:     "test-vm",
			DeletedAt:  time.Now(),
			Stage:      "completed",
			CleanupISO: true,
			ISOPath:    "/tmp/test-vm-cloudinit.iso",
		}

		if !result.Success {
			t.Errorf("Expected Success to be true")
		}
		if !result.CleanupISO {
			t.Errorf("Expected CleanupISO to be true")
		}
	})

	t.Run("DeleteResult failure", func(t *testing.T) {
		result := DeleteResult{
			Success:      false,
			VMName:       "test-vm",
			ErrorMessage: "VM not found",
			Stage:        "initialization",
		}

		if result.Success {
			t.Errorf("Expected Success to be false")
		}
		if result.ErrorMessage == "" {
			t.Errorf("Expected ErrorMessage to be set")
		}
	})
}

func TestDomainConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int32
		expected int32
	}{
		{"DomainNoState", DomainNoState, 0},
		{"DomainRunning", DomainRunning, 1},
		{"DomainBlocked", DomainBlocked, 2},
		{"DomainPaused", DomainPaused, 3},
		{"DomainShutoff", DomainShutoff, 5},
		{"DomainCrashed", DomainCrashed, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

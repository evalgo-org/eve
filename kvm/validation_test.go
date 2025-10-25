package kvm

import "testing"

func TestIsValidVMName(t *testing.T) {
	tests := []struct {
		name     string
		vmName   string
		expected bool
	}{
		// Valid names
		{
			name:     "Simple lowercase name",
			vmName:   "myvm",
			expected: true,
		},
		{
			name:     "Name with dashes",
			vmName:   "my-vm",
			expected: true,
		},
		{
			name:     "Name with underscores",
			vmName:   "my_vm",
			expected: true,
		},
		{
			name:     "Name starting with underscore",
			vmName:   "_myvm",
			expected: true,
		},
		{
			name:     "Mixed case name",
			vmName:   "MyVM",
			expected: true,
		},
		{
			name:     "Name with numbers",
			vmName:   "vm123",
			expected: true,
		},
		{
			name:     "Complex valid name",
			vmName:   "Web_Server-01",
			expected: true,
		},
		{
			name:     "Single letter",
			vmName:   "v",
			expected: true,
		},
		{
			name:     "Max length (64 chars)",
			vmName:   "a123456789012345678901234567890123456789012345678901234567890123",
			expected: true,
		},

		// Invalid names
		{
			name:     "Empty name",
			vmName:   "",
			expected: false,
		},
		{
			name:     "Starting with digit",
			vmName:   "1vm",
			expected: false,
		},
		{
			name:     "Starting with dash",
			vmName:   "-myvm",
			expected: false,
		},
		{
			name:     "Contains space",
			vmName:   "my vm",
			expected: false,
		},
		{
			name:     "Contains dot",
			vmName:   "my.vm",
			expected: false,
		},
		{
			name:     "Contains special char (@)",
			vmName:   "my@vm",
			expected: false,
		},
		{
			name:     "Contains special char (#)",
			vmName:   "my#vm",
			expected: false,
		},
		{
			name:     "Contains slash",
			vmName:   "my/vm",
			expected: false,
		},
		{
			name:     "Too long (65 chars)",
			vmName:   "a1234567890123456789012345678901234567890123456789012345678901234",
			expected: false,
		},
		{
			name:     "Way too long (100 chars)",
			vmName:   "a123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			expected: false,
		},
		{
			name:     "Only numbers",
			vmName:   "123456",
			expected: false,
		},
		{
			name:     "Only dash",
			vmName:   "-",
			expected: false,
		},
		{
			name:     "Unicode characters",
			vmName:   "vm_ñ",
			expected: false,
		},
		{
			name:     "Tab character",
			vmName:   "my\tvm",
			expected: false,
		},
		{
			name:     "Newline character",
			vmName:   "my\nvm",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidVMName(tt.vmName)
			if result != tt.expected {
				t.Errorf("IsValidVMName(%q) = %v, want %v", tt.vmName, result, tt.expected)
			}
		})
	}
}

// Benchmark for performance testing
func BenchmarkIsValidVMName(b *testing.B) {
	testCases := []string{
		"valid-vm-name",
		"invalid vm name",
		"_valid_name_123",
		"123-invalid",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			IsValidVMName(tc)
		}
	}
}

func TestIsValidVMNameEdgeCases(t *testing.T) {
	t.Run("Exactly 64 characters", func(t *testing.T) {
		// Create exactly 64 char string
		name := "a"
		for i := 1; i < 64; i++ {
			name += "b"
		}
		if len(name) != 64 {
			t.Fatalf("Test setup error: name length is %d, expected 64", len(name))
		}
		if !IsValidVMName(name) {
			t.Errorf("Expected 64-char name to be valid")
		}
	})

	t.Run("Exactly 65 characters", func(t *testing.T) {
		// Create exactly 65 char string
		name := "a"
		for i := 1; i < 65; i++ {
			name += "b"
		}
		if len(name) != 65 {
			t.Fatalf("Test setup error: name length is %d, expected 65", len(name))
		}
		if IsValidVMName(name) {
			t.Errorf("Expected 65-char name to be invalid")
		}
	})
}

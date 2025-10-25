package kvm

import (
	"strings"
	"testing"
)

func TestExtractMACFromXML(t *testing.T) {
	tests := []struct {
		name     string
		xmlInput string
		expected string
	}{
		{
			name: "Standard MAC with single quotes",
			xmlInput: `<domain type="kvm">
  <devices>
    <interface type="network">
      <mac address='52:54:00:12:34:56'/>
      <source network="default"/>
    </interface>
  </devices>
</domain>`,
			expected: "52:54:00:12:34:56",
		},
		{
			name: "Standard MAC with double quotes",
			xmlInput: `<domain type="kvm">
  <devices>
    <interface type="network">
      <mac address="52:54:00:aa:bb:cc"/>
      <source network="default"/>
    </interface>
  </devices>
</domain>`,
			expected: "52:54:00:aa:bb:cc",
		},
		{
			name: "MAC on single line",
			xmlInput: `<mac address="52:54:00:de:ad:be"/>`,
			expected: "52:54:00:de:ad:be",
		},
		{
			name: "MAC with extra whitespace",
			xmlInput: `
  <mac address="52:54:00:11:22:33"/>
`,
			expected: "52:54:00:11:22:33",
		},
		{
			name: "Multiple interfaces - first MAC",
			xmlInput: `<domain>
  <devices>
    <interface type="network">
      <mac address="52:54:00:00:00:01"/>
    </interface>
    <interface type="network">
      <mac address="52:54:00:00:00:02"/>
    </interface>
  </devices>
</domain>`,
			expected: "52:54:00:00:00:01",
		},
		{
			name:     "No MAC address",
			xmlInput: `<domain><devices><interface type="network"/></devices></domain>`,
			expected: "",
		},
		{
			name:     "Empty string",
			xmlInput: "",
			expected: "",
		},
		{
			name:     "Invalid XML",
			xmlInput: "not xml at all",
			expected: "",
		},
		{
			name: "MAC in comment (simple parser finds first match)",
			xmlInput: `<!-- This is a comment without MAC -->
<interface><mac address="52:54:00:11:11:11"/></interface>`,
			expected: "52:54:00:11:11:11",
		},
		{
			name: "Uppercase MAC",
			xmlInput: `<mac address="52:54:00:AA:BB:CC"/>`,
			expected: "52:54:00:AA:BB:CC",
		},
		{
			name: "Mixed quotes priority (single quote found first)",
			xmlInput: `<mac address='52:54:00:11:11:11'/>`,
			expected: "52:54:00:11:11:11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMACFromXML(tt.xmlInput)
			if result != tt.expected {
				t.Errorf("ExtractMACFromXML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateDomainXML(t *testing.T) {
	t.Run("Basic domain XML generation", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "test-vm",
			MemoryKiB:    2097152,
			VCPUs:        2,
			ImagePath:    "/var/lib/libvirt/images/test.qcow2",
			CloudInitISO: "/tmp/test-cloudinit.iso",
			NetworkName:  "default",
		}

		xml := GenerateDomainXML(cfg)

		// Verify it contains expected elements
		expectedStrings := []string{
			"<domain type=\"kvm\">",
			"<name>test-vm</name>",
			"<memory unit=\"KiB\">2097152</memory>",
			"<vcpu placement=\"static\">2</vcpu>",
			"/var/lib/libvirt/images/test.qcow2",
			"/tmp/test-cloudinit.iso",
			"<source network=\"default\"/>",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(xml, expected) {
				t.Errorf("Generated XML missing expected string: %q", expected)
			}
		}
	})

	t.Run("Default values applied", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "minimal-vm",
			ImagePath:    "/path/to/image.qcow2",
			CloudInitISO: "/path/to/iso",
			// MemoryKiB, VCPUs, NetworkName not set - should use defaults
		}

		xml := GenerateDomainXML(cfg)

		// Should have default 2GB memory
		if !strings.Contains(xml, "<memory unit=\"KiB\">2097152</memory>") {
			t.Errorf("Expected default memory (2097152 KiB) not found")
		}

		// Should have default 2 VCPUs
		if !strings.Contains(xml, "<vcpu placement=\"static\">2</vcpu>") {
			t.Errorf("Expected default VCPUs (2) not found")
		}

		// Should have default network
		if !strings.Contains(xml, "<source network=\"default\"/>") {
			t.Errorf("Expected default network not found")
		}
	})

	t.Run("Custom resource allocation", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "custom-vm",
			MemoryKiB:    8388608, // 8GB
			VCPUs:        8,
			ImagePath:    "/path/to/image.qcow2",
			CloudInitISO: "/path/to/iso",
			NetworkName:  "br0",
		}

		xml := GenerateDomainXML(cfg)

		if !strings.Contains(xml, "<memory unit=\"KiB\">8388608</memory>") {
			t.Errorf("Expected custom memory (8388608 KiB) not found")
		}

		if !strings.Contains(xml, "<vcpu placement=\"static\">8</vcpu>") {
			t.Errorf("Expected custom VCPUs (8) not found")
		}

		if !strings.Contains(xml, "<source network=\"br0\"/>") {
			t.Errorf("Expected custom network (br0) not found")
		}
	})

	t.Run("Required XML structure elements", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "structure-test",
			ImagePath:    "/image.qcow2",
			CloudInitISO: "/cloudinit.iso",
		}

		xml := GenerateDomainXML(cfg)

		requiredElements := []string{
			"<?xml version='1.0'?>",
			"<domain type=\"kvm\">",
			"<os>",
			"<type arch=\"x86_64\"",
			"<features>",
			"<acpi/>",
			"<apic/>",
			"<cpu mode=\"host-passthrough\"",
			"<clock offset=\"utc\"/>",
			"<devices>",
			"<emulator>/usr/bin/qemu-system-x86_64</emulator>",
			"<disk type=\"file\" device=\"disk\">",
			"<driver name=\"qemu\" type=\"qcow2\"",
			"<disk type=\"file\" device=\"cdrom\">",
			"<interface type=\"network\">",
			"<model type=\"virtio\"/>",
			"<graphics type=\"spice\"",
			"<memballoon model=\"virtio\"/>",
			"</domain>",
		}

		for _, elem := range requiredElements {
			if !strings.Contains(xml, elem) {
				t.Errorf("Required XML element missing: %q", elem)
			}
		}
	})

	t.Run("Disk configuration", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "disk-test",
			ImagePath:    "/custom/path/disk.qcow2",
			CloudInitISO: "/custom/cloudinit.iso",
		}

		xml := GenerateDomainXML(cfg)

		// Main disk
		if !strings.Contains(xml, "<source file=\"/custom/path/disk.qcow2\"/>") {
			t.Errorf("Main disk path not found in XML")
		}
		if !strings.Contains(xml, "<target dev=\"vda\" bus=\"virtio\"/>") {
			t.Errorf("Main disk target not found in XML")
		}

		// Cloud-init ISO
		if !strings.Contains(xml, "<source file=\"/custom/cloudinit.iso\"/>") {
			t.Errorf("Cloud-init ISO path not found in XML")
		}
		if !strings.Contains(xml, "<target dev=\"sda\" bus=\"sata\"/>") {
			t.Errorf("Cloud-init ISO target not found in XML")
		}
		if !strings.Contains(xml, "<readonly/>") {
			t.Errorf("Cloud-init ISO should be readonly")
		}
	})

	t.Run("VM name in XML", func(t *testing.T) {
		names := []string{
			"simple",
			"with-dashes",
			"with_underscores",
			"MixedCase123",
		}

		for _, name := range names {
			cfg := DomainXMLConfig{
				Name:         name,
				ImagePath:    "/image.qcow2",
				CloudInitISO: "/iso",
			}

			xml := GenerateDomainXML(cfg)

			expectedTag := "<name>" + name + "</name>"
			if !strings.Contains(xml, expectedTag) {
				t.Errorf("Expected name tag %q not found in XML", expectedTag)
			}
		}
	})

	t.Run("XML is well-formed", func(t *testing.T) {
		cfg := DomainXMLConfig{
			Name:         "wellformed-test",
			ImagePath:    "/image.qcow2",
			CloudInitISO: "/iso",
		}

		xml := GenerateDomainXML(cfg)

		// Check it starts and ends correctly
		if !strings.HasPrefix(strings.TrimSpace(xml), "<?xml version='1.0'?>") {
			t.Errorf("XML doesn't start with XML declaration")
		}

		if !strings.HasSuffix(strings.TrimSpace(xml), "</domain>") {
			t.Errorf("XML doesn't end with </domain>")
		}

		// Ensure no obvious malformed tags
		openTags := strings.Count(xml, "<")
		closeTags := strings.Count(xml, ">")
		if openTags != closeTags {
			t.Errorf("Mismatched tag count: %d open, %d close", openTags, closeTags)
		}
	})
}

func BenchmarkExtractMACFromXML(b *testing.B) {
	xmlInput := `<domain type="kvm">
  <devices>
    <interface type="network">
      <mac address="52:54:00:12:34:56"/>
      <source network="default"/>
    </interface>
  </devices>
</domain>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractMACFromXML(xmlInput)
	}
}

func BenchmarkGenerateDomainXML(b *testing.B) {
	cfg := DomainXMLConfig{
		Name:         "benchmark-vm",
		MemoryKiB:    2097152,
		VCPUs:        2,
		ImagePath:    "/var/lib/libvirt/images/test.qcow2",
		CloudInitISO: "/tmp/test-cloudinit.iso",
		NetworkName:  "default",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateDomainXML(cfg)
	}
}

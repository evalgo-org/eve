package kvm

import (
	"fmt"
	"strings"
)

// DomainXMLConfig holds parameters for domain XML generation
type DomainXMLConfig struct {
	Name         string
	MemoryKiB    int
	VCPUs        int
	ImagePath    string
	CloudInitISO string
	NetworkName  string
}

// GenerateDomainXML creates a KVM domain XML definition
func GenerateDomainXML(cfg DomainXMLConfig) string {
	if cfg.MemoryKiB == 0 {
		cfg.MemoryKiB = 2097152 // 2GB default
	}
	if cfg.VCPUs == 0 {
		cfg.VCPUs = 2
	}
	if cfg.NetworkName == "" {
		cfg.NetworkName = "default"
	}

	return fmt.Sprintf(`<?xml version='1.0'?>
<domain type="kvm">
  <name>%s</name>
  <memory unit="KiB">%d</memory>
  <currentMemory unit="KiB">%d</currentMemory>
  <vcpu placement="static">%d</vcpu>
  <os>
    <type arch="x86_64" machine="pc-q35-9.2">hvm</type>
    <boot dev="hd"/>
  </os>
  <features><acpi/><apic/></features>
  <cpu mode="host-passthrough" check="none" migratable="on"/>
  <clock offset="utc"/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type="file" device="disk">
      <driver name="qemu" type="qcow2" discard="unmap"/>
      <source file="%s"/>
      <target dev="vda" bus="virtio"/>
    </disk>
    <disk type="file" device="cdrom">
      <driver name="qemu" type="raw"/>
      <source file="%s"/>
      <target dev="sda" bus="sata"/>
      <readonly/>
    </disk>
    <interface type="network">
      <source network="%s"/>
      <model type="virtio"/>
    </interface>
    <controller type="virtio-serial" index="0"/>
    <channel type="unix">
      <target type="virtio" name="org.qemu.guest_agent.0"/>
    </channel>
    <graphics type="spice" autoport="yes"/>
    <console type="pty">
      <target type="serial" port="0"/>
    </console>
    <memballoon model="virtio"/>
    <rng model="virtio">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
</domain>`, cfg.Name, cfg.MemoryKiB, cfg.MemoryKiB, cfg.VCPUs,
		cfg.ImagePath, cfg.CloudInitISO, cfg.NetworkName)
}

// ExtractMACFromXML parses MAC address from domain XML string
func ExtractMACFromXML(xmlStr string) string {
	lines := strings.Split(xmlStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "<mac address=") {
			start := strings.Index(line, "'")
			if start == -1 {
				start = strings.Index(line, "\"")
			}
			if start != -1 {
				end := strings.Index(line[start+1:], "'")
				if end == -1 {
					end = strings.Index(line[start+1:], "\"")
				}
				if end != -1 {
					return line[start+1 : start+1+end]
				}
			}
		}
	}
	return ""
}

# KVM Library

A Go library for managing KVM virtual machines via libvirt with cloud-init support.

## Overview

This library provides a high-level API for common KVM operations:

- **Create VMs** with cloud-init configuration
- **List VMs** with state and network information
- **Delete VMs** with cleanup options
- **Low-level utilities** for XML generation, validation, networking

## Installation

```bash
go get evalgo.org/kvm/kvm
```

## Quick Example

```go
package main

import (
    "fmt"
    "evalgo.org/kvm/kvm"
)

func main() {
    // Create a VM
    result := kvm.CreateVM(kvm.CreateVMParams{
        VMName:           "my-vm",
        DistributionName: "Rocky Linux 9",
        DefaultUser:      "rocky",
        ImagePath:        "/var/lib/libvirt/images/rocky.qcow2",
        CloudInitISO:     "/tmp/my-vm-cloudinit.iso",
        SSHPublicKey:     "ssh-rsa AAAA...",
        LibvirtSocket:    "/var/run/libvirt/libvirt-sock",
    })

    if result.Success {
        fmt.Printf("SSH: %s\n", result.SSHCommand)
    }
}
```

## Package Structure

```
kvm/
├── types.go         # Data structures and constants
├── connection.go    # Libvirt connection management
├── domain.go        # High-level VM operations (Create, List, Delete)
├── xml.go          # Domain XML generation and parsing
├── validation.go   # Input validation
├── cloudinit.go    # Cloud-init ISO creation
└── network.go      # Network and IP detection
```

## API Documentation

### High-Level API

#### CreateVM

Creates a new VM with cloud-init configuration.

```go
func CreateVM(params CreateVMParams) *VMResult
```

**Parameters:**

```go
type CreateVMParams struct {
    VMName           string   // VM name (validated)
    DistributionName string   // Display name for distribution
    DefaultUser      string   // Default SSH user
    ImagePath        string   // Path to qcow2 image (must exist)
    CloudInitISO     string   // Output path for cloud-init ISO
    SSHPublicKey     string   // SSH public key content
    PackageUpdate    bool     // Run package updates
    Packages         []string // Packages to install
    LibvirtSocket    string   // Libvirt socket path
    MemoryKiB        int      // Memory in KiB (default: 2GB)
    VCPUs            int      // Number of vCPUs (default: 2)
}
```

**Returns:**

```go
type VMResult struct {
    Success      bool      // Operation succeeded
    VMName       string    // VM name
    IPAddress    string    // Detected IP address
    MACAddress   string    // MAC address
    ImagePath    string    // Path to VM image
    CloudInitISO string    // Path to cloud-init ISO
    SSHCommand   string    // Ready-to-use SSH command
    CreatedAt    time.Time // Creation timestamp
    ErrorMessage string    // Error if failed
    Stage        string    // Current/failed stage
    Distribution string    // Distribution name
}
```

**Example:**

```go
result := kvm.CreateVM(kvm.CreateVMParams{
    VMName:        "web-server",
    ImagePath:     "/var/lib/libvirt/images/rocky.qcow2",
    SSHPublicKey:  string(keyData),
    LibvirtSocket: "/var/run/libvirt/libvirt-sock",
    MemoryKiB:     4194304, // 4GB
    VCPUs:         4,
})

if !result.Success {
    log.Fatalf("Failed at %s: %s", result.Stage, result.ErrorMessage)
}
```

---

#### ListVMs

Lists all or running VMs with detailed information.

```go
func ListVMs(params ListVMsParams) ([]VMInfo, error)
```

**Parameters:**

```go
type ListVMsParams struct {
    LibvirtSocket string
    OnlyRunning   bool
    Distributions map[string]DistributionInfo // Optional: for name detection
}
```

**Returns:**

```go
type VMInfo struct {
    Name         string    // VM name
    State        string    // running, paused, shut off, etc.
    IPAddress    string    // IP if available
    Distribution string    // Detected distribution
    CreatedAt    time.Time // Timestamp
    IsActive     bool      // Is VM running
}
```

**Example:**

```go
vms, err := kvm.ListVMs(kvm.ListVMsParams{
    LibvirtSocket: "/var/run/libvirt/libvirt-sock",
    OnlyRunning:   true,
})

for _, vm := range vms {
    fmt.Printf("%s: %s (%s)\n", vm.Name, vm.State, vm.IPAddress)
}
```

---

#### DeleteVM

Destroys and undefines a VM.

```go
func DeleteVM(params DeleteVMParams) *DeleteResult
```

**Parameters:**

```go
type DeleteVMParams struct {
    VMName          string // VM to delete
    LibvirtSocket   string // Libvirt socket path
    CleanupISO      bool   // Also delete cloud-init ISO
    CloudISOTmpBase string // Base directory for ISOs
}
```

**Returns:**

```go
type DeleteResult struct {
    Success      bool      // Operation succeeded
    VMName       string    // VM name
    DeletedAt    time.Time // Deletion timestamp
    ErrorMessage string    // Error if failed
    Stage        string    // Current/failed stage
    CleanupISO   bool      // ISO was cleaned up
    ISOPath      string    // Path to cleaned ISO
}
```

**Example:**

```go
result := kvm.DeleteVM(kvm.DeleteVMParams{
    VMName:          "web-server",
    LibvirtSocket:   "/var/run/libvirt/libvirt-sock",
    CleanupISO:      true,
    CloudISOTmpBase: "/tmp",
})
```

---

### Low-Level API

#### Connection Management

```go
// Connect to libvirt
vir, err := kvm.Connect("/var/run/libvirt/libvirt-sock")
defer kvm.Disconnect(vir)
```

#### Validation

```go
// Validate VM name (libvirt rules)
valid := kvm.IsValidVMName("my-vm") // true
valid = kvm.IsValidVMName("123-vm") // false (starts with digit)
```

#### XML Operations

```go
// Generate domain XML
xml := kvm.GenerateDomainXML(kvm.DomainXMLConfig{
    Name:         "my-vm",
    MemoryKiB:    4194304, // 4GB
    VCPUs:        4,
    ImagePath:    "/path/to/image.qcow2",
    CloudInitISO: "/path/to/cloudinit.iso",
})

// Extract MAC address from XML
mac := kvm.ExtractMACFromXML(domainXMLString)
```

#### Cloud-init

```go
// Create cloud-init ISO
err := kvm.CreateCloudInitISO(kvm.CloudInitConfig{
    VMName:        "my-vm",
    SSHPublicKey:  "ssh-rsa AAAA...",
    PackageUpdate: true,
    Packages:      []string{"vim", "git"},
}, "/tmp/cloudinit.iso")
```

#### Network Operations

```go
// Get VM IP address (with retries)
ip, err := kvm.GetVMIPAddress(vir, domain, "52:54:00:12:34:56", 40)

// Get IP from DHCP lease
ip, err := kvm.GetDHCPLeaseForMAC(vir, "default", "52:54:00:12:34:56")
```

#### State Conversion

```go
stateStr := kvm.StateToString(kvm.DomainRunning) // "running"
```

---

## Constants

```go
const (
    DomainNoState int32 = 0
    DomainRunning int32 = 1
    DomainBlocked int32 = 2
    DomainPaused  int32 = 3
    DomainShutoff int32 = 5
    DomainCrashed int32 = 6
)
```

## Error Handling

All high-level functions return result structs with:
- `Success` boolean
- `ErrorMessage` string with details
- `Stage` string indicating where failure occurred

```go
result := kvm.CreateVM(params)
if !result.Success {
    switch result.Stage {
    case "initialization":
        // Handle validation errors
    case "connected_to_libvirt":
        // Handle connection errors
    case "defining_domain":
        // Handle XML/definition errors
    default:
        log.Printf("Failed at %s: %s", result.Stage, result.ErrorMessage)
    }
}
```

## Requirements

- Go 1.24+
- Libvirt daemon running
- `genisoimage` or `mkisofs` for cloud-init ISO creation
- Appropriate permissions to access libvirt socket

## Examples

See the `/examples` directory for complete working examples:

- `01_create_vm.go` - Basic VM creation
- `02_list_vms.go` - List all VMs
- `03_delete_vm.go` - Delete a VM
- `04_complete_workflow.go` - Full lifecycle example
- `05_low_level_api.go` - Low-level API usage

## Thread Safety

The library is not thread-safe. Each libvirt connection should be used from a single goroutine, or protected with appropriate synchronization.

## License

See the main project LICENSE file.

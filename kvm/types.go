package kvm

import (
	"fmt"
	"time"
)

// VMResult represents the outcome of VM creation
type VMResult struct {
	Success      bool      `json:"success"`
	VMName       string    `json:"vm_name"`
	IPAddress    string    `json:"ip_address,omitempty"`
	MACAddress   string    `json:"mac_address,omitempty"`
	ImagePath    string    `json:"image_path"`
	CloudInitISO string    `json:"cloud_init_iso"`
	SSHCommand   string    `json:"ssh_command,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Stage        string    `json:"stage"`
	Distribution string    `json:"distribution,omitempty"`
}

// VMInfo represents information about a VM
type VMInfo struct {
	Name         string    `json:"name"`
	State        string    `json:"state"`
	IPAddress    string    `json:"ip_address,omitempty"`
	Distribution string    `json:"distribution,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	IsActive     bool      `json:"is_active"`
}

// DeleteResult represents the outcome of VM deletion
type DeleteResult struct {
	Success      bool      `json:"success"`
	VMName       string    `json:"vm_name"`
	DeletedAt    time.Time `json:"deleted_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Stage        string    `json:"stage"`
	CleanupISO   bool      `json:"cleanup_iso,omitempty"`
	ISOPath      string    `json:"iso_path,omitempty"`
}

// Domain state constants
const (
	DomainNoState int32 = 0
	DomainRunning int32 = 1
	DomainBlocked int32 = 2
	DomainPaused  int32 = 3
	DomainShutoff int32 = 5
	DomainCrashed int32 = 6
)

// StateToString converts domain state int to readable string
func StateToString(state int32) string {
	switch state {
	case DomainRunning:
		return "running"
	case DomainPaused:
		return "paused"
	case DomainShutoff:
		return "shut off"
	case DomainCrashed:
		return "crashed"
	default:
		return fmt.Sprintf("unknown (%d)", state)
	}
}

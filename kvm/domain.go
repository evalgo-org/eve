package kvm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
)

// CreateVMParams holds all parameters for VM creation
type CreateVMParams struct {
	VMName           string
	DistributionName string
	DefaultUser      string
	ImagePath        string
	CloudInitISO     string
	SSHPublicKey     string
	PackageUpdate    bool
	Packages         []string
	LibvirtSocket    string
	MemoryKiB        int // Optional: defaults to 2GB
	VCPUs            int // Optional: defaults to 2
}

// CreateVM orchestrates the full VM creation process
func CreateVM(params CreateVMParams) *VMResult {
	result := &VMResult{
		Distribution: params.DistributionName,
		CreatedAt:    time.Now(),
		Stage:        "initialization",
		VMName:       params.VMName,
		ImagePath:    params.ImagePath,
		CloudInitISO: params.CloudInitISO,
	}

	// Validate VM name
	if !IsValidVMName(params.VMName) {
		result.ErrorMessage = "Invalid VM name: must start with letter/underscore, contain only [a-zA-Z0-9_-], and be <=64 chars"
		return result
	}

	// Validate image exists
	if _, err := os.Stat(params.ImagePath); os.IsNotExist(err) {
		result.ErrorMessage = fmt.Sprintf("Image file not found: %s", params.ImagePath)
		return result
	}

	// Connect to libvirt
	vir, err := Connect(params.LibvirtSocket)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	defer Disconnect(vir)

	result.Stage = "connected_to_libvirt"

	// Create cloud-init ISO
	result.Stage = "preparing_cloud_init"
	cloudCfg := CloudInitConfig{
		VMName:        params.VMName,
		SSHPublicKey:  params.SSHPublicKey,
		PackageUpdate: params.PackageUpdate,
		Packages:      params.Packages,
	}
	if err := CreateCloudInitISO(cloudCfg, params.CloudInitISO); err != nil {
		result.ErrorMessage = err.Error()
		return result
	}

	result.Stage = "defining_domain"

	// Undefine if exists
	if dom, err := vir.DomainLookupByName(params.VMName); err == nil {
		_ = vir.DomainDestroy(dom)
		_ = vir.DomainUndefine(dom)
	}

	// Generate and define domain
	domainXML := GenerateDomainXML(DomainXMLConfig{
		Name:         params.VMName,
		ImagePath:    params.ImagePath,
		CloudInitISO: params.CloudInitISO,
		MemoryKiB:    params.MemoryKiB,
		VCPUs:        params.VCPUs,
	})

	dom, err := vir.DomainDefineXML(domainXML)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("DomainDefineXML failed: %v", err)
		return result
	}

	result.Stage = "starting_domain"
	if err := vir.DomainCreate(dom); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to start domain: %v", err)
		return result
	}

	result.Stage = "getting_network_info"
	domainXMLStr, err := vir.DomainGetXMLDesc(dom, 0)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get domain XML: %v", err)
		return result
	}

	vmMAC := ExtractMACFromXML(domainXMLStr)
	if vmMAC == "" {
		result.ErrorMessage = "failed to extract MAC address"
		return result
	}
	result.MACAddress = vmMAC

	result.Stage = "waiting_for_ip"
	vmIP, _ := GetVMIPAddress(vir, dom, vmMAC, 40)
	if vmIP == "" {
		result.Stage = "ip_detection_failed"
		result.ErrorMessage = fmt.Sprintf("Could not determine IP for MAC %s", vmMAC)
		return result
	}

	result.Stage = "completed"
	result.Success = true
	result.IPAddress = vmIP
	result.SSHCommand = fmt.Sprintf("ssh %s@%s", params.DefaultUser, vmIP)

	return result
}

// ListVMsParams holds parameters for listing VMs
type ListVMsParams struct {
	LibvirtSocket string
	OnlyRunning   bool
	Distributions map[string]DistributionInfo // For distribution detection
}

// DistributionInfo minimal info needed for distribution detection
type DistributionInfo struct {
	Name string
	Key  string
}

// ListVMs retrieves all or running VMs
func ListVMs(params ListVMsParams) ([]VMInfo, error) {
	vir, err := Connect(params.LibvirtSocket)
	if err != nil {
		return nil, err
	}
	defer Disconnect(vir)

	domains, _, err := vir.ConnectListAllDomains(1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	// Check if default network exists
	_, err = vir.NetworkLookupByName("default")
	networkExists := (err == nil)

	var vms []VMInfo
	for _, dom := range domains {
		state, _, err := vir.DomainGetState(dom, 0)
		if err != nil {
			continue
		}

		if params.OnlyRunning && state != DomainRunning {
			continue
		}

		name := dom.Name
		stateStr := StateToString(state)
		isActive := state == DomainRunning

		var ipAddr string
		if isActive && networkExists {
			xmlDesc, _ := vir.DomainGetXMLDesc(dom, 0)
			mac := ExtractMACFromXML(xmlDesc)
			ipAddr, _ = GetDHCPLeaseForMAC(vir, "default", mac)
		}

		// Distribution detection
		var distName string
		for key, dist := range params.Distributions {
			if strings.HasPrefix(name, key+"-") || name == key {
				distName = dist.Name
				break
			}
		}

		vms = append(vms, VMInfo{
			Name:         name,
			State:        stateStr,
			IPAddress:    ipAddr,
			Distribution: distName,
			IsActive:     isActive,
			CreatedAt:    time.Now(),
		})
	}

	return vms, nil
}

// DeleteVMParams holds parameters for VM deletion
type DeleteVMParams struct {
	VMName          string
	LibvirtSocket   string
	CleanupISO      bool
	CloudISOTmpBase string
}

// DeleteVM destroys and undefines a VM
func DeleteVM(params DeleteVMParams) *DeleteResult {
	result := &DeleteResult{
		VMName:    params.VMName,
		DeletedAt: time.Now(),
		Stage:     "initialization",
	}

	vir, err := Connect(params.LibvirtSocket)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	defer Disconnect(vir)

	result.Stage = "connected_to_libvirt"

	dom, err := vir.DomainLookupByName(params.VMName)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("VM '%s' not found: %v", params.VMName, err)
		return result
	}

	result.Stage = "destroying_domain"
	state, _, err := vir.DomainGetState(dom, 0)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get domain state: %v", err)
		return result
	}

	domainState := libvirt.DomainState(state)
	if domainState == libvirt.DomainRunning || domainState == libvirt.DomainPaused {
		if err := vir.DomainDestroy(dom); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to destroy VM: %v", err)
			return result
		}
	}

	result.Stage = "undefining_domain"
	if err := vir.DomainUndefine(dom); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to undefine VM: %v", err)
		return result
	}

	result.Stage = "cleanup"
	if params.CleanupISO {
		isoPath := filepath.Join(params.CloudISOTmpBase, fmt.Sprintf("%s-cloudinit.iso", params.VMName))
		if err := os.Remove(isoPath); err == nil {
			result.CleanupISO = true
			result.ISOPath = isoPath
		}
	}

	result.Success = true
	result.Stage = "completed"
	return result
}

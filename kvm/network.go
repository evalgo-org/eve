package kvm

import (
	"strings"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
)

// GetVMIPAddress attempts to detect VM IP address via DHCP leases
// Returns empty string if not found after maxAttempts
func GetVMIPAddress(vir *libvirt.Libvirt, dom libvirt.Domain, macAddress string, maxAttempts int) (string, error) {
	network, err := vir.NetworkLookupByName("default")
	if err != nil {
		return "", err
	}

	var vmIP string
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Try domain interface addresses first
		interfaces, _ := vir.DomainInterfaceAddresses(dom,
			uint32(libvirt.DomainInterfaceAddressesSrcLease), 0)
		if len(interfaces) > 0 {
			for _, iface := range interfaces {
				for _, addr := range iface.Addrs {
					if libvirt.IPAddrType(addr.Type) == libvirt.IPAddrTypeIpv4 {
						vmIP = addr.Addr
						break
					}
				}
				if vmIP != "" {
					break
				}
			}
		}

		// Fallback to DHCP leases
		if vmIP == "" {
			leases, _, _ := vir.NetworkGetDhcpLeases(network, libvirt.OptString{}, 0, 0)
			for _, lease := range leases {
				if len(lease.Mac) > 0 && strings.EqualFold(lease.Mac[0], macAddress) {
					vmIP = lease.Ipaddr
					break
				}
			}
		}

		if vmIP != "" {
			return vmIP, nil
		}

		time.Sleep(3 * time.Second)
	}

	return "", nil // Not found
}

// GetDHCPLeaseForMAC retrieves IP from DHCP leases by MAC address
func GetDHCPLeaseForMAC(vir *libvirt.Libvirt, networkName, macAddress string) (string, error) {
	network, err := vir.NetworkLookupByName(networkName)
	if err != nil {
		return "", err
	}

	leases, _, err := vir.NetworkGetDhcpLeases(network, libvirt.OptString{}, 0, 0)
	if err != nil {
		return "", err
	}

	for _, lease := range leases {
		for _, leaseMac := range lease.Mac {
			if leaseMac != "" && strings.EqualFold(leaseMac, macAddress) {
				return lease.Ipaddr, nil
			}
		}
	}

	return "", nil // Not found
}

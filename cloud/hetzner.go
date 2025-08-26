package cloud

import (
	"context"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	eve "eve.evalgo.org/common"
)

func HetznerServerCreate(token, sName, sType string) {
	client := hcloud.NewClient(hcloud.WithToken(token))
	sshKeys := make([]*hcloud.SSHKey, 1)
	sshKeys[0] = &hcloud.SSHKey{ID: 19739629, Name: "opunix@earth.overlay.services", PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMjQfx/zXodrYd9aM9NsiNHQR6PsH/gAiL5QAiE7YAvn opunix@earth.overlay.services"}
	if sType == "default" {
		// dedicated resources ccx13/8GB/2CPU, ccx23/16GB/4 CPU, ccx33/32GB/8CPU
		created, _, err := client.Server.Create(context.Background(), hcloud.ServerCreateOpts{
			Name:       sName,
			Image:      &hcloud.Image{Name: "alma-10", OSFlavor: "almalinux", OSVersion: "10", Architecture: "x86"},
			ServerType: &hcloud.ServerType{Name: "ccx13", CPUType: hcloud.CPUTypeDedicated},
			Location:   &hcloud.Location{Name: "nbg1"},
			SSHKeys:    sshKeys,
		})
		if err != nil {
			eve.Logger.Error("error ::: ", err)
		}
		eve.Logger.Info(created)
	}
}

func HetznerServerDelete(token, sName string) {
	client := hcloud.NewClient(hcloud.WithToken(token))
	server, _, err := client.Server.GetByName(context.Background(), sName)
	if err != nil {
		eve.Logger.Error(err)
		return
	}
	eve.Logger.Info(server)
	resp, _, err := client.Server.DeleteWithResult(context.Background(), server)
	if err != nil {
		eve.Logger.Info(err)
	}
	eve.Logger.Info(resp)
}

func HetznerServers(token string) {
	client := hcloud.NewClient(hcloud.WithToken(token))
	servers, _, _ := client.Server.List(context.Background(), hcloud.ServerListOpts{})
	for _, server := range servers {
		server, _, err := client.Server.GetByID(context.Background(), server.ID)
		if err != nil {
			eve.Logger.Error("error retrieving server: ", err)
		}
		if server != nil {
			// for _, price := range server.ServerType.Pricings {
			// 	eve.Logger.Info("pricing ", price.Monthly, "location: ", price.Location)
			// }
			eve.Logger.Info("server ", server.ID, " is called: ", server.Name, "location: ", server.Datacenter.Location)
		} else {
			eve.Logger.Info("server ", server.ID, " not found")
		}
	}
}

func HetznerPrices(token string) {
	client := hcloud.NewClient(hcloud.WithToken(token))
	prices, _, err := client.Pricing.Get(context.Background())
	if err != nil {
		eve.Logger.Info(err)
	}
	for _, price := range prices.ServerTypes {
		eve.Logger.Info(price.ServerType.Name)
		for _, p := range price.Pricings {
			eve.Logger.Info(p.Monthly, p.Location)
		}
	}
}

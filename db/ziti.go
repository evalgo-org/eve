package db

import (
	"context"
	"github.com/openziti/sdk-golang/ziti"
	"net"
	"net/http"
)

var (
	identityFile string       = ""
	cfg          *ziti.Config = nil
	zitiContext  ziti.Context = nil
	err          error        = nil
)

func ZitiSetup(identityFile, serviceName string) (*http.Transport, error) {
	cfg, err = ziti.NewConfigFromFile(identityFile)
	if err != nil {
		return nil, err
	}
	zitiContext, err = ziti.NewContext(cfg)
	if err != nil {
		return nil, err
	}
	zitiTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return zitiContext.Dial(serviceName)
		},
	}
	return zitiTransport, nil
}

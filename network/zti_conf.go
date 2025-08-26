package network

import (
	"os"

	"gopkg.in/yaml.v3"

	eve "eve.evalgo.org/common"
)

// https://openziti.io/docs/reference/configuration/controller
// https://openziti.io/docs/reference/configuration/router

//
// === Shared Structs ===
//

type ZitiIdentity struct {
	Cert       string `yaml:"cert"`
	ServerCert string `yaml:"server_cert"`
	Key        string `yaml:"key"`
	CA         string `yaml:"ca"`
}

type ZitiDialer struct {
	Binding string `yaml:"binding"`
}

type ZitiListener struct {
	Binding string `yaml:"binding"`
	Address string `yaml:"address"`
}

//
// === Controller-Specific ===
//

type ZitiCtrlListener struct {
	Listener string `yaml:"listener"`
}

type ZitiEdgeEnroll struct {
	Enrollment ZitiEnrollment `yaml:"enrollment"`
}

type ZitiEnrollment struct {
	SigningCert ZitiSigningCert `yaml:"signingCert"`
}

type ZitiSigningCert struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type ZitiWebListener struct {
	Name       string           `yaml:"name"`
	BindPoints []ZitiBindPoint  `yaml:"bindPoints"`
	APIs       []ZitiAPIBinding `yaml:"apis"`
}

type ZitiBindPoint struct {
	Interface string `yaml:"interface"`
	Address   string `yaml:"address"`
}

type ZitiAPIBinding struct {
	Binding string `yaml:"binding"`
}

//
// === Router-Specific ===
//

type ZitiCtrlEndpoint struct {
	Endpoint string `yaml:"endpoint"`
}

type ZitiEdgeCSR struct {
	CSR ZitiCSR `yaml:"csr"`
}

type ZitiCSR struct {
	Country            string   `yaml:"country"`
	Province           string   `yaml:"province"`
	Locality           string   `yaml:"locality"`
	Organization       string   `yaml:"organization"`
	OrganizationalUnit string   `yaml:"organizationalUnit"`
	SANs               ZitiSANs `yaml:"sans"`
}

type ZitiSANs struct {
	DNS []string `yaml:"dns"`
	IP  []string `yaml:"ip"`
}

type ZitiLinkConfig struct {
	Listeners []ZitiLinkListener `yaml:"listeners"`
	Dialers   []ZitiDialer       `yaml:"dialers"`
}

type ZitiLinkListener struct {
	Binding   string `yaml:"binding"`
	Bind      string `yaml:"bind"`
	Advertise string `yaml:"advertise"`
}

type ZitiRouterConfig struct {
	Version   int              `yaml:"v"`
	Identity  ZitiIdentity     `yaml:"identity"`
	Ctrl      ZitiCtrlEndpoint `yaml:"ctrl"`
	Dialers   []ZitiDialer     `yaml:"dialers"`
	Edge      ZitiEdgeCSR      `yaml:"edge"`
	Link      ZitiLinkConfig   `yaml:"link"`
	Listeners []ZitiListener   `yaml:"listeners"`
}

type ZitiControllerConfig struct {
	Version  int               `yaml:"v"`
	DB       string            `yaml:"db"`
	Identity ZitiIdentity      `yaml:"identity"`
	Ctrl     ZitiCtrlListener  `yaml:"ctrl"`
	Edge     ZitiEdgeEnroll    `yaml:"edge"`
	Web      []ZitiWebListener `yaml:"web"`
}

func WriteZitiRouterConfig(filename string, cfg ZitiRouterConfig) error {
	file, err := os.Create(filename)
	if err != nil {
		eve.Logger.Fatal("failed to create file ", filename, ":", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	if err := encoder.Encode(cfg); err != nil {
		eve.Logger.Fatal("failed to encode router config to YAML:", err)
	}

	eve.Logger.Info("Ziti router config written to ", filename, "\n")
	return nil
}

func WriteZitiControllerConfig(filename string, cfg ZitiControllerConfig) error {
	file, err := os.Create(filename)
	if err != nil {
		eve.Logger.Fatal("failed to create file ", filename, ":", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	if err := encoder.Encode(cfg); err != nil {
		eve.Logger.Fatal("failed to encode controller config to YAML:", err)
	}

	eve.Logger.Info("Ziti controller config written to ", filename, " \n")
	return nil
}

func ZitiGenerateCtrlConfig(outFile string) {
	ctrlCfg := ZitiControllerConfig{
		Version: 3,
		DB:      "ctrl.db",
		Identity: ZitiIdentity{
			Cert:       "ctrl-client.cert.pem",
			ServerCert: "ctrl-server.cert.pem",
			Key:        "ctrl.key.pem",
			CA:         "ca-chain.cert.pem",
		},
		Ctrl: ZitiCtrlListener{Listener: "tls:127.0.0.1:6262"},
		Edge: ZitiEdgeEnroll{
			Enrollment: ZitiEnrollment{
				SigningCert: ZitiSigningCert{
					Cert: "intermediate.cert.pem",
					Key:  "intermediate.key.pem",
				},
			},
		},
		Web: []ZitiWebListener{
			{
				Name: "all-apis-localhost",
				BindPoints: []ZitiBindPoint{
					{Interface: "127.0.0.1:1280", Address: "127.0.0.1:1280"},
				},
				APIs: []ZitiAPIBinding{
					{Binding: "fabric"},
					{Binding: "edge-management"},
					{Binding: "edge-client"},
				},
			},
		},
	}
	if err := WriteZitiControllerConfig(outFile, ctrlCfg); err != nil {
		eve.Logger.Fatal(err)
	}
}

func ZitiGenerateRouterConfig(outFile string) {
	routerCfg := ZitiRouterConfig{
		Version: 3,
		Identity: ZitiIdentity{
			Cert:       "router.cert.pem",
			ServerCert: "router.server.cert.pem",
			Key:        "router.key.pem",
			CA:         "ca-chain.cert.pem",
		},
		Ctrl: ZitiCtrlEndpoint{Endpoint: "tls:127.0.0.1:6262"},
		Dialers: []ZitiDialer{
			{Binding: "udp"},
			{Binding: "transport"},
		},
		Edge: ZitiEdgeCSR{
			CSR: ZitiCSR{
				Country:            "US",
				Province:           "NC",
				Locality:           "Charlotte",
				Organization:       "OpenZiti",
				OrganizationalUnit: "Ziti",
				SANs: ZitiSANs{
					DNS: []string{"localhost"},
					IP:  []string{"127.0.0.1"},
				},
			},
		},
		Link: ZitiLinkConfig{
			Listeners: []ZitiLinkListener{
				{
					Binding:   "transport",
					Bind:      "tls:127.0.0.1:6002",
					Advertise: "tls:127.0.0.1:6002",
				},
			},
			Dialers: []ZitiDialer{
				{Binding: "transport"},
			},
		},
		Listeners: []ZitiListener{
			{Binding: "edge", Address: "tls:0.0.0.0:3022"},
			{Binding: "transport", Address: "tls:0.0.0.0:7099"},
		},
	}
	if err := WriteZitiRouterConfig(outFile, routerCfg); err != nil {
		eve.Logger.Fatal(err)
	}
}

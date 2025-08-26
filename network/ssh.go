package network

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	// "io/ioutil"
	"crypto/x509"
	"encoding/pem"
	"os"

	eve "eve.evalgo.org/common"
)

func ssh_keyfile(privateKeyPath string, certKeyPath string) (ssh.Signer, error) {
	// privateKeyPassword := ""
	// pemBytes, err := ioutil.ReadFile(privateKeyPath)
	// if err != nil {
	// 	return nil, fmt.Errorf("Reading private key file failed %v", err)
	// }
	// signer, err := signerFromPem(pemBytes, []byte(privateKeyPassword))
	// if err != nil {
	// 	return nil, err
	// }
	// return signer, nil

	// parse the user's private key:
	pvtKeyBts, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(pvtKeyBts)
	if err != nil {
		return nil, err
	}

	// parse the user's certificate:
	certBts, err := os.ReadFile(certKeyPath)
	if err != nil {
		return nil, err
	}

	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBts)
	if err != nil {
		return nil, err
	}

	// create a signer using both the certificate and the private key:
	return ssh.NewCertSigner(cert.(*ssh.Certificate), signer)

}

func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {

	// read pem block
	err := errors.New("Pem decode failed, no key found")
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, err
	}

	// handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) {
		// decrypt PEM
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("Decrypting PEM block failed %v", err)
		}

		// get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("Creating signer from encrypted key failed %v", err)
		}

		return signer, nil
	} else {
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing plain private key failed %v", err)
		}

		return signer, nil
	}
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
		} else {
			return key, nil
		}
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing EC private key failed %v", err)
		} else {
			return key, nil
		}
	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
		} else {
			return key, nil
		}
	default:
		return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", block.Type)
	}
}

func SshExec(address string, username string, keyfile string, certfile string, cmd string) {
	var signer ssh.Signer
	var err error
	var config *ssh.ClientConfig
	if certfile == "" {
		signer, err = ssh_keyfile(keyfile, certfile)
		config = &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		signer, err = ssh_keyfile(keyfile, certfile)
		config = &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		eve.Logger.Fatal("Failed to dial: ", err)
	}
	session, err := client.NewSession()
	if err != nil {
		eve.Logger.Fatal("Failed to create session: ", err)
	}
	defer session.Close()
	// modes := ssh.TerminalModes{
	// 	ssh.ECHO:          0,     // disable echoing
	// 	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	// 	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	// }
	// if err := session.RequestPty("linux", 80, 40, modes); err != nil {
	// 	eve.Logger.Fatal("request for pseudo terminal failed: ", err)
	// }
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		eve.Logger.Fatal("ui", err)
	}
	fmt.Println(string(out))
}

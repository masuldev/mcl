package internal

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
)

func GetVolumeUsage(bastion, target string) error {
	bastionHost := bastion
	targetHost := target
	user := "ec2-user"
	keyPath := ""

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	bastionClient, err := ssh.Dial("tcp", bastionHost+":22", config)
	if err != nil {
		return err
	}

	defer bastionClient.Close()

	conn, err := bastionClient.Dial("tcp", targetHost+":22")
	if err != nil {
		return err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetHost+":22", config)
	if err != nil {
		return err
	}

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	session, err := targetClient.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	cmd := "df --output=pcent / | tail -1 | tr -dc '0-9'"
	err = session.Run(cmd)
	if err != nil {
		return err
	}

	fmt.Printf("Output: %s\n", stdoutBuf.String())
	return nil
}

package internal

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"strconv"
)

type (
	VolumeUsage struct {
		InstanceId string
		Usage      int
	}
)

func GetVolumeUsage(bastion *ssh.Client, target *Target) (*VolumeUsage, error) {
	user := "ec2-user"
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", FindHomeFolder(), target.KeyName)

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := bastion.Dial("tcp", target.PrivateIp+":22")
	if err != nil {
		return nil, err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, target.PrivateIp+":22", config)
	if err != nil {
		return nil, err
	}

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	session, err := targetClient.NewSession()
	if err != nil {
		return nil, err
	}

	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	cmd := "df --output=pcent / | tail -1 | tr -dc '0-9'"
	err = session.Run(cmd)
	if err != nil {
		return nil, err
	}

	convertedUsage, err := strconv.Atoi(stdoutBuf.String())
	//fmt.Println(convertedUsage)

	return &VolumeUsage{InstanceId: target.Id, Usage: convertedUsage}, nil
}

func ConnectionBastion(bastionHost, keyName string) (*ssh.Client, error) {
	user := "ec2-user"
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", FindHomeFolder(), keyName)
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return bastionClient, nil
}

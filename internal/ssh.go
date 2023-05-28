package internal

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"strconv"
	"strings"
)

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

func GetVolumeUsage(bastion *ssh.Client, target *Target) (int, error) {
	user := "ec2-user"
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", FindHomeFolder(), target.KeyName)

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return 0, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return 0, err
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
		return 0, err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, target.PrivateIp+":22", config)
	if err != nil {
		return 0, err
	}

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	session, err := targetClient.NewSession()
	if err != nil {
		return 0, err
	}

	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	cmd := "df --output=pcent / | tail -1 | tr -dc '0-9'"
	err = session.Run(cmd)
	if err != nil {
		return 0, err
	}

	convertedUsage, err := strconv.Atoi(stdoutBuf.String())

	return convertedUsage, nil
}

func ModifyLinuxVolume(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
	user := "ec2-user"
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", FindHomeFolder(), instance.KeyName)

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := bastion.Dial("tcp", instance.PrivateIp+":22")
	if err != nil {
		return "", err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, instance.PrivateIp+":22", config)
	if err != nil {
		return "", err
	}

	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	session, err := targetClient.NewSession()
	if err != nil {
		return "", err
	}

	defer session.Close()

	checkFileSystemCommand := fmt.Sprintf("sudo lsblk -f %s1 -o FSTYPE | tail -n 1", "/dev/xvda")
	resizeExtFileSystemCommand := fmt.Sprintf("sudo growpart %s 1 && sudo resize2fs %s1", "/dev/xvda", "/dev/xvda")
	resizeXfsFileSystemCommand := fmt.Sprintf("sudo growpart %s 1 && sudo xfs_growfs %s1", "/dev/xvda", "/dev/xvda")

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	err = session.Run(checkFileSystemCommand)
	if err != nil {
		return "", err
	}

	newSession, err := targetClient.NewSession()
	if err != nil {
		return "", err
	}

	defer newSession.Close()
	fileSystem := strings.TrimSpace(stdoutBuf.String())

	if fileSystem == "xfs" {
		err = newSession.Run(resizeXfsFileSystemCommand)
		if err != nil {
			return "", err
		}
	} else {
		err = newSession.Run(resizeExtFileSystemCommand)
		if err != nil {
			return "", err
		}
	}

	return instance.PrivateIp, nil
}

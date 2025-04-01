package internal

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultUser    = "ec2-user"
	defaultDevice  = "/dev/xvda"
	sshDialTimeout = 10 * time.Second
	maxRetries     = 3
	retryDelay     = 500 * time.Millisecond
)

// SSH 클라이언트 설정 캐시 (키: keyPath, 값: *ssh.ClientConfig)
var sshClientConfigCache sync.Map

// 재시도 헬퍼 함수
func retry(attempts int, delay time.Duration, operation func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = operation(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}

// 키 파일 경로를 기반으로 캐싱된 SSH 클라이언트 설정을 반환합니다.
func getSSHClientConfigCached(keyName string) (*ssh.ClientConfig, error) {
	home := FindHomeFolder()
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", home, keyName)
	if cfg, ok := sshClientConfigCache.Load(keyPath); ok {
		return cfg.(*ssh.ClientConfig), nil
	}
	cfg, err := newSSHClientConfig(keyName)
	if err != nil {
		return nil, err
	}
	sshClientConfigCache.Store(keyPath, cfg)
	return cfg, nil
}

func newSSHClientConfig(keyName string) (*ssh.ClientConfig, error) {
	home := FindHomeFolder()
	keyPath := fmt.Sprintf("%s/.ssh/%s.pem", home, keyName)
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            defaultUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshDialTimeout,
	}
	return config, nil
}

func ConnectionBastion(bastionHost, keyName string) (*ssh.Client, error) {
	config, err := getSSHClientConfigCached(keyName)
	if err != nil {
		return nil, err
	}

	var bastionClient *ssh.Client
	err = retry(maxRetries, retryDelay, func() error {
		var err error
		bastionClient, err = ssh.Dial("tcp", bastionHost+":22", config)
		return err
	})
	if err != nil {
		return nil, err
	}
	return bastionClient, nil
}

func GetVolumeUsage(bastion *ssh.Client, target *Target) (int, error) {
	config, err := getSSHClientConfigCached(target.KeyName)
	if err != nil {
		return 0, err
	}

	var conn net.Conn
	err = retry(maxRetries, retryDelay, func() error {
		var err error
		conn, err = bastion.Dial("tcp", target.PrivateIp+":22")
		return err
	})
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
	// 재시도 로직을 적용 (주의: 세션은 한 번 실행 후 재사용 불가할 수 있음)
	err = retry(maxRetries, retryDelay, func() error {
		return session.Run(cmd)
	})
	if err != nil {
		return 0, err
	}

	usageStr := strings.TrimSpace(stdoutBuf.String())
	convertedUsage, err := strconv.Atoi(usageStr)
	if err != nil {
		return 0, err
	}
	return convertedUsage, nil
}

func ModifyLinuxVolume(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
	config, err := getSSHClientConfigCached(instance.KeyName)
	if err != nil {
		return "", err
	}

	var conn net.Conn
	err = retry(maxRetries, retryDelay, func() error {
		var err error
		conn, err = bastion.Dial("tcp", instance.PrivateIp+":22")
		return err
	})
	if err != nil {
		return "", err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, instance.PrivateIp+":22", config)
	if err != nil {
		return "", err
	}
	targetClient := ssh.NewClient(ncc, chans, reqs)
	defer targetClient.Close()

	// 첫 번째 세션: 파일시스템 타입 확인
	session, err := targetClient.NewSession()
	if err != nil {
		return "", err
	}
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	checkFSCommand := fmt.Sprintf("sudo lsblk -f %s1 -o FSTYPE | tail -n 1", defaultDevice)
	err = retry(maxRetries, retryDelay, func() error {
		return session.Run(checkFSCommand)
	})
	if err != nil {
		session.Close()
		return "", err
	}
	session.Close()

	fileSystem := strings.TrimSpace(stdoutBuf.String())

	// 두 번째 세션: 파일시스템 확장 명령 실행
	newSession, err := targetClient.NewSession()
	if err != nil {
		return "", err
	}
	defer newSession.Close()

	resizeExtCmd := fmt.Sprintf("sudo growpart %s 1 && sudo resize2fs %s1", defaultDevice, defaultDevice)
	resizeXfsCmd := fmt.Sprintf("sudo growpart %s 1 && sudo xfs_growfs %s1", defaultDevice, defaultDevice)
	var resizeCmd string
	if fileSystem == "xfs" {
		resizeCmd = resizeXfsCmd
	} else {
		resizeCmd = resizeExtCmd
	}
	err = retry(maxRetries, retryDelay, func() error {
		return newSession.Run(resizeCmd)
	})
	if err != nil {
		return "", err
	}
	return instance.PrivateIp, nil
}

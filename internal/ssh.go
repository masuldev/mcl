package internal

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	defaultUser    = "ec2-user"
	defaultDevice  = "/dev/xvda"
	sshDialTimeout = 10 * time.Second
	maxRetries     = 3
	retryDelay     = 500 * time.Millisecond
	// 연결 풀링 관련 상수
	maxPoolSize     = 50
	connTimeout     = 30 * time.Second
	cleanupInterval = 5 * time.Minute
)

// SSH 연결 풀 구조체
type SSHConnectionPool struct {
	connections   map[string]*ssh.Client
	mutex         sync.RWMutex
	lastUsed      map[string]time.Time
	cleanupTicker *time.Ticker
}

// 전역 SSH 연결 풀
var sshConnectionPool = &SSHConnectionPool{
	connections: make(map[string]*ssh.Client),
	lastUsed:    make(map[string]time.Time),
}

// 연결 풀 초기화
func init() {
	sshConnectionPool.cleanupTicker = time.NewTicker(cleanupInterval)
	go sshConnectionPool.cleanupExpiredConnections()
}

// 연결 키 생성
func getConnectionKey(host, keyName string) string {
	return fmt.Sprintf("%s:%s", host, keyName)
}

// SSH 연결 풀에서 연결 가져오기
func (p *SSHConnectionPool) getConnection(host, keyName string) (*ssh.Client, error) {
	key := getConnectionKey(host, keyName)

	p.mutex.RLock()
	if conn, exists := p.connections[key]; exists {
		// 연결 상태 확인
		if conn.Conn != nil && conn.Conn.RemoteAddr() != nil {
			p.lastUsed[key] = time.Now()
			p.mutex.RUnlock()
			return conn, nil
		}
		// 연결이 끊어진 경우 제거
		delete(p.connections, key)
		delete(p.lastUsed, key)
	}
	p.mutex.RUnlock()

	// 새로운 연결 생성
	return p.createConnection(host, keyName)
}

// 새로운 SSH 연결 생성
func (p *SSHConnectionPool) createConnection(host, keyName string) (*ssh.Client, error) {
	config, err := getSSHClientConfigCached(keyName)
	if err != nil {
		return nil, err
	}

	var client *ssh.Client
	err = retry(maxRetries, retryDelay, func() error {
		var err error
		client, err = ssh.Dial("tcp", host+":22", config)
		return err
	})
	if err != nil {
		return nil, err
	}

	key := getConnectionKey(host, keyName)
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 풀 크기 제한 확인
	if len(p.connections) >= maxPoolSize {
		// 가장 오래된 연결 제거
		p.removeOldestConnection()
	}

	p.connections[key] = client
	p.lastUsed[key] = time.Now()

	return client, nil
}

// 가장 오래된 연결 제거
func (p *SSHConnectionPool) removeOldestConnection() {
	var oldestKey string
	var oldestTime time.Time

	for key, lastUsed := range p.lastUsed {
		if oldestKey == "" || lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = lastUsed
		}
	}

	if oldestKey != "" {
		if conn, exists := p.connections[oldestKey]; exists {
			conn.Close()
		}
		delete(p.connections, oldestKey)
		delete(p.lastUsed, oldestKey)
	}
}

// 만료된 연결 정리
func (p *SSHConnectionPool) cleanupExpiredConnections() {
	for range p.cleanupTicker.C {
		p.mutex.Lock()
		now := time.Now()
		for key, lastUsed := range p.lastUsed {
			if now.Sub(lastUsed) > connTimeout {
				if conn, exists := p.connections[key]; exists {
					conn.Close()
				}
				delete(p.connections, key)
				delete(p.lastUsed, key)
			}
		}
		p.mutex.Unlock()
	}
}

// 연결 풀 정리
func (p *SSHConnectionPool) cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, conn := range p.connections {
		conn.Close()
	}
	p.connections = make(map[string]*ssh.Client)
	p.lastUsed = make(map[string]time.Time)

	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}
}

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

// 개선된 Bastion 연결 함수 (연결 풀 사용)
func ConnectionBastion(bastionHost, keyName string) (*ssh.Client, error) {
	return sshConnectionPool.getConnection(bastionHost, keyName)
}

// 개선된 볼륨 사용량 조회 함수 (연결 풀 사용)
func GetVolumeUsage(bastion *ssh.Client, target *Target) (int, error) {
	// Bastion을 통한 타겟 서버 연결
	targetClient, err := getTargetConnection(bastion, target)
	if err != nil {
		return 0, err
	}
	defer targetClient.Close()

	session, err := targetClient.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	cmd := "df --output=pcent / | tail -1 | tr -dc '0-9'"

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

// 타겟 서버 연결 생성 (연결 풀 사용)
func getTargetConnection(bastion *ssh.Client, target *Target) (*ssh.Client, error) {
	config, err := getSSHClientConfigCached(target.KeyName)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	err = retry(maxRetries, retryDelay, func() error {
		var err error
		conn, err = bastion.Dial("tcp", target.PrivateIp+":22")
		return err
	})
	if err != nil {
		return nil, err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, target.PrivateIp+":22", config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(ncc, chans, reqs), nil
}

// 개선된 볼륨 수정 함수 (연결 풀 사용)
func ModifyLinuxVolume(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
	targetClient, err := getTargetConnection(bastion, instance)
	if err != nil {
		return "", err
	}
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

// 프로그램 종료 시 연결 풀 정리
func CleanupSSHConnections() {
	sshConnectionPool.cleanup()
}

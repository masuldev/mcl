package internal

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os/user"
	"time"
)

func GetVolumeUsageWithTimeout(f func(bastion *ssh.Client, target *Target) (int, error), timeout time.Duration, bastion *ssh.Client, target *Target) (int, error) {
	resultChan := make(chan int)
	errorChan := make(chan error)

	go func() {
		result, err := f(bastion, target)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return 0, err
	case <-time.After(timeout):
		return 0, fmt.Errorf("ssh connection timeout")
	}
}

func ModifyLinuxVolumeWithTimeout(f func(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error), timeout time.Duration, bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
	resultChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		result, err := f(bastion, volume, instance)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("Timeout InstanceId: %s\n", instance.Id)
	}
}

func FindHomeFolder() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

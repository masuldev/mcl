package internal

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os/user"
	"time"
)

func GetVolumeUsageWithTimeout(f func(bastion *ssh.Client, target *Target) (*VolumeUsage, error), timeout time.Duration, bastion *ssh.Client, target *Target) (*VolumeUsage, error) {
	resultChan := make(chan *VolumeUsage)
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
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("Timeout InstanceId: %s\n", target.Id)
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

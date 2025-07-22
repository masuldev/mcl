package internal

import (
	"context"
	"fmt"
	"os/user"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

func GetVolumeUsageWithTimeout(ctx context.Context, f func(bastion *ssh.Client, target *Target) (int, error), timeout time.Duration, bastion *ssh.Client, target *Target) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan int, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := f(bastion, target)
		select {
		case resultChan <- result:
		case <-ctx.Done():
			return
		}
		if err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return 0, err
	case <-ctx.Done():
		return 0, fmt.Errorf("ssh connection timeout or cancelled: %v", ctx.Err())
	}
}

func ModifyLinuxVolumeWithTimeout(ctx context.Context, f func(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error), timeout time.Duration, bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := f(bastion, volume, instance)
		if err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
				return
			}
			return
		}
		select {
		case resultChan <- result:
		case <-ctx.Done():
			return
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("Timeout or cancelled for InstanceId: %s, error: %v", instance.Id, ctx.Err())
	}
}

func retryWithContext(ctx context.Context, attempts int, delay time.Duration, operation func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %v", ctx.Err())
		default:
		}

		if err = operation(); err == nil {
			return nil
		}

		if i < attempts-1 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled during retry: %v", ctx.Err())
			}
		}
	}
	return err
}

func parallelWithContext(ctx context.Context, maxWorkers int, tasks []func() error) error {
	semaphore := make(chan struct{}, maxWorkers)
	errChan := make(chan error, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(task func() error) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errChan <- fmt.Errorf("operation cancelled: %v", ctx.Err())
				return
			}

			if err := task(); err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
					return
				}
			}
		}(task)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(errChan)
		for err := range errChan {
			return err
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("parallel operation cancelled: %v", ctx.Err())
	}
}

func FindHomeFolder() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

package internal

import (
	"context"
	"fmt"
	"os/user"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh"
)

// 로깅 스타일 상수
const (
	IconSuccess = "✓"
	IconWarning = "⚠️"
	IconError   = "❌"
	IconInfo    = "ℹ️"
)

// Success 메시지 출력
func LogSuccess(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	color.Green("%s %s", IconSuccess, message)
}

// Warning 메시지 출력
func LogWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	color.Yellow("%s %s", IconWarning, message)
}

// Error 메시지 출력
func LogError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	color.Red("%s %s", IconError, message)
}

// Info 메시지 출력
func LogInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	color.Cyan("%s %s", IconInfo, message)
}

// AWS 서비스 로그 출력 (일관된 형식)
func LogAwsService(cmd, region, name, id string, additional ...string) {
	base := fmt.Sprintf("%s: region: %s, name: %s, id: %s",
		color.CyanString(cmd),
		color.YellowString(region),
		color.YellowString(name),
		color.YellowString(id))

	if len(additional) > 0 {
		base += ", " + additional[0]
	}

	fmt.Println(base)
}

// AWS 서비스 상세 로그 출력
func LogAwsServiceDetail(cmd, region, name, id, endpoint, status, engine string) {
	fmt.Printf("%s: region: %s, name: %s, id: %s, endpoint: %s, status: %s, engine: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.YellowString(name),
		color.YellowString(id), color.BlueString(endpoint), color.GreenString(status),
		color.MagentaString(engine))
}

// EC2 인스턴스 로그 출력
func LogEC2Instance(cmd, region, name, id, publicIp, privateIp string) {
	fmt.Printf("%s: region: %s, name: %s, id: %s, publicIp: %s, privateIp: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.YellowString(name),
		color.YellowString(id), color.BlueString(publicIp), color.BlueString(privateIp))
}

// 볼륨 사용량 로그 출력
func LogVolumeUsage(cmd, instanceId, instanceName, instanceIp string, usage int) {
	fmt.Printf("%s: instance id: %s, instance name: %s, instance ip: %s, usage: %s\n",
		color.CyanString(cmd), color.YellowString(instanceId), color.YellowString(instanceName),
		color.MagentaString(instanceIp), color.GreenString("%d%%", usage))
}

// 볼륨 확장 로그 출력
func LogVolumeExpansion(cmd, instanceId, instanceName, volumeId string, oldSize, newSize int) {
	fmt.Printf("%s: instance id: %s, instance name: %s, volume id: %s, old size: %d GB, new size: %d GB\n",
		color.CyanString(cmd), color.YellowString(instanceId), color.YellowString(instanceName),
		color.YellowString(volumeId), oldSize, newSize)
}

// CloudFront 배포 로그 출력
func LogCloudFrontDistribution(cmd, region, name, domain, aliasInfo, status, comment string) {
	fmt.Printf("%s: region: %s, name: %s, domain: %s%s, status: %s, comment: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.YellowString(name),
		color.BlueString(domain), color.CyanString(aliasInfo), color.GreenString(status),
		color.MagentaString(comment))
}

// CloudFront 무효화 로그 출력
func LogCloudFrontInvalidation(cmd, distributionId string) {
	fmt.Printf("%s: distribution id: %s, invalidation: /*\n",
		color.CyanString(cmd), color.YellowString(distributionId))
}

// S3 버킷 로그 출력
func LogS3Bucket(cmd, region, bucketName, creationDate string) {
	fmt.Printf("%s: region: %s, bucket: %s, created: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.BlueString(bucketName),
		color.GreenString(creationDate))
}

// S3 객체 로그 출력
func LogS3Object(cmd, region, bucketName, objectKey, size, lastModified string) {
	fmt.Printf("%s: region: %s, bucket: %s, object: %s, size: %s, last modified: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.BlueString(bucketName),
		color.CyanString(objectKey), color.GreenString(size), color.MagentaString(lastModified))
}

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

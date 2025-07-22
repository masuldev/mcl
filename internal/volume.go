package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"golang.org/x/crypto/ssh"
)

type (
	TargetVolume struct {
		Id         string
		Size       int32
		NewSize    int64
		InstanceId string
		Device     string
	}

	VolumeInstanceMapping struct {
		Instance *Target
		Volume   *TargetVolume
	}
)

const (
	maxBatchSize = 199
)

// 객체 풀링을 위한 구조체
type VolumeProcessor struct {
	volumeTablePool sync.Pool
	instanceIdsPool sync.Pool
	volumeIdsPool   sync.Pool
}

// 전역 프로세서 인스턴스
var volumeProcessor = &VolumeProcessor{
	volumeTablePool: sync.Pool{
		New: func() interface{} {
			return make(map[string]*TargetVolume)
		},
	},
	instanceIdsPool: sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 100)
		},
	},
	volumeIdsPool: sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 1000)
		},
	},
}

// 메모리 최적화된 볼륨 조회 함수
func FindVolume(ctx context.Context, cfg aws.Config, instanceIds []string) (map[string]*TargetVolume, error) {
	client := ec2.NewFromConfig(cfg)

	// 객체 풀에서 재사용
	volumeTable := volumeProcessor.volumeTablePool.Get().(map[string]*TargetVolume)
	defer func() {
		// 맵 초기화 후 풀에 반환
		for k := range volumeTable {
			delete(volumeTable, k)
		}
		volumeProcessor.volumeTablePool.Put(volumeTable)
	}()

	// 인스턴스 ID 집합 생성 (메모리 효율적)
	instanceIdSet := make(map[string]struct{}, len(instanceIds))
	for _, id := range instanceIds {
		instanceIdSet[id] = struct{}{}
	}

	// 볼륨 ID 조회
	volumeIds := volumeProcessor.volumeIdsPool.Get().([]string)
	defer func() {
		volumeIds = volumeIds[:0] // 슬라이스 재사용
		volumeProcessor.volumeIdsPool.Put(volumeIds)
	}()

	allVolumeIds, err := FindVolumes(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// 배치 처리로 메모리 사용량 최적화
	for i := 0; i < len(allVolumeIds); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(allVolumeIds) {
			end = len(allVolumeIds)
		}
		batch := allVolumeIds[i:end]

		output, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("volume-id"),
					Values: batch,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		// 볼륨 정보 처리
		for _, vol := range output.Volumes {
			if len(vol.Attachments) > 0 {
				attachment := vol.Attachments[0]
				instanceId := aws.ToString(attachment.InstanceId)
				if _, exists := instanceIdSet[instanceId]; exists {
					volumeTable[instanceId] = &TargetVolume{
						Id:         aws.ToString(attachment.VolumeId),
						Size:       aws.ToInt32(vol.Size),
						InstanceId: instanceId,
						Device:     aws.ToString(attachment.Device),
					}
				}
			}
		}
	}

	// 결과 복사 (풀에서 반환되기 전에)
	result := make(map[string]*TargetVolume, len(volumeTable))
	for k, v := range volumeTable {
		result[k] = v
	}

	return result, nil
}

// 메모리 최적화된 볼륨 ID 조회
func FindVolumes(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(cfg)

	// 객체 풀에서 재사용
	volumeIds := volumeProcessor.volumeIdsPool.Get().([]string)
	defer func() {
		volumeIds = volumeIds[:0]
		volumeProcessor.volumeIdsPool.Put(volumeIds)
	}()

	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{
		MaxResults: aws.Int32(maxOutputResults),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vol := range page.Volumes {
			volumeIds = append(volumeIds, aws.ToString(vol.VolumeId))
		}
	}

	// 결과 복사
	result := make([]string, len(volumeIds))
	copy(result, volumeIds)
	return result, nil
}

// 메모리 최적화된 볼륨 확장
func ExpandVolume(ctx context.Context, cfg aws.Config, volumes map[string]*TargetVolume, incrementPercentage int) ([]*TargetVolume, error) {
	client := ec2.NewFromConfig(cfg)

	// 슬라이스 사전 할당으로 메모리 재할당 방지
	expandedVolumes := make([]*TargetVolume, 0, len(volumes))
	var errList []error

	for _, volume := range volumes {
		currentSize := volume.Size
		newSize := int64(float64(currentSize) * (1 + float64(incrementPercentage)/100))

		modifyVolumeInput := &ec2.ModifyVolumeInput{
			VolumeId: aws.String(volume.Id),
			Size:     aws.Int32(int32(newSize)),
		}

		_, err := client.ModifyVolume(ctx, modifyVolumeInput)
		if err != nil {
			errList = append(errList, fmt.Errorf("error modifying volume %s: %v", volume.Id, err))
			continue
		}

		err = waitUntilVolumeAvailable(ctx, client, volume.Id)
		if err != nil {
			errList = append(errList, fmt.Errorf("error waiting for volume %s to be available: %v", volume.Id, err))
		}

		volume.NewSize = newSize
		expandedVolumes = append(expandedVolumes, volume)
	}

	var retErr error
	if len(errList) > 0 {
		retErr = fmt.Errorf("following errors occurred: %v", errList)
	}

	return expandedVolumes, retErr
}

// 컨텍스트 취소 지원 개선된 볼륨 대기 함수
func waitUntilVolumeAvailable(ctx context.Context, client *ec2.Client, volumeId string) error {
	describeInput := &ec2.DescribeVolumesModificationsInput{
		VolumeIds: []string{volumeId},
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for volume %s to be available", volumeId)
		case <-ticker.C:
			output, err := client.DescribeVolumesModifications(ctx, describeInput)
			if err != nil {
				return fmt.Errorf("error describing volume %s: %v", volumeId, err)
			}
			if len(output.VolumesModifications) > 0 && output.VolumesModifications[0].ModificationState == types.VolumeModificationStateOptimizing {
				return nil
			}
		}
	}
}

// 메모리 최적화된 볼륨 확장 및 수정
func ExpandAndModifyVolumes(ctx context.Context, awsConfig aws.Config, instances map[string]*Target, targets []*Target, incrementPercentage int, bastionClient *ssh.Client) ([]VolumeInstanceMapping, error) {
	// 인스턴스 룩업 맵 생성 (메모리 효율적)
	instanceLookup := make(map[string]*Target, len(instances))
	for _, instance := range instances {
		instanceLookup[instance.Id] = instance
	}

	// 인스턴스 ID 슬라이스 생성 (객체 풀 사용)
	instanceIds := volumeProcessor.instanceIdsPool.Get().([]string)
	defer func() {
		instanceIds = instanceIds[:0]
		volumeProcessor.instanceIdsPool.Put(instanceIds)
	}()

	for _, target := range targets {
		instanceIds = append(instanceIds, target.Id)
	}

	volumes, err := FindVolume(ctx, awsConfig, instanceIds)
	if err != nil {
		return nil, fmt.Errorf("error finding volumes: %w", err)
	}

	expandedVolumes, err := ExpandVolume(ctx, awsConfig, volumes, incrementPercentage)
	if err != nil {
		PrintError(err)
	}

	// 볼륨 인스턴스 매핑 생성 (사전 할당)
	volumeInstanceMappings := make([]*VolumeInstanceMapping, 0, len(expandedVolumes))
	for _, volume := range expandedVolumes {
		if instance, ok := instanceLookup[volume.InstanceId]; ok {
			volumeInstanceMappings = append(volumeInstanceMappings, &VolumeInstanceMapping{
				Volume:   volume,
				Instance: instance,
			})
		}
	}

	// 결과 슬라이스 사전 할당
	volumeInstances := make([]VolumeInstanceMapping, 0, len(volumeInstanceMappings))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20)
	var mu sync.Mutex

	// 컨텍스트 취소 지원
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	for _, mapping := range volumeInstanceMappings {
		wg.Add(1)
		go func(mapping *VolumeInstanceMapping) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			_, err = ModifyLinuxVolumeWithTimeout(ctx, func(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
				return ModifyLinuxVolume(bastion, mapping.Volume, mapping.Instance)
			}, 10*time.Second, bastionClient, mapping.Volume, mapping.Instance)
			if err != nil {
				PrintError(WrapError(fmt.Errorf("cannot modify volume %s, instance id %s", err, mapping.Instance.Id)))
				return
			}

			mu.Lock()
			volumeInstances = append(volumeInstances, *mapping)
			mu.Unlock()
		}(mapping)
	}

	// 컨텍스트 취소 대기
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 정상 완료
	case <-ctx.Done():
		// 타임아웃 또는 취소
		return volumeInstances, fmt.Errorf("operation cancelled or timed out")
	}

	return volumeInstances, nil
}

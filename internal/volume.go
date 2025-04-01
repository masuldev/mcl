package internal

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"golang.org/x/crypto/ssh"
	"sync"
	"time"
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

const maxBatchSize = 199

func FindVolume(ctx context.Context, cfg aws.Config, instanceIds []string) (map[string]*TargetVolume, error) {
	client := ec2.NewFromConfig(cfg)
	volumeTable := make(map[string]*TargetVolume)

	instanceIdSet := make(map[string]struct{}, len(instanceIds))
	for _, id := range instanceIds {
		instanceIdSet[id] = struct{}{}
	}

	allVolumeIds, err := FindVolumes(ctx, cfg)
	if err != nil {
		return nil, err
	}

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

	return volumeTable, nil
}

func FindVolumes(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(cfg)
	var volumeIds []string

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

	return volumeIds, nil
}

func ExpandVolume(ctx context.Context, cfg aws.Config, volumes map[string]*TargetVolume, incrementPercentage int) ([]*TargetVolume, error) {
	var (
		expandedVolumes []*TargetVolume
		client          = ec2.NewFromConfig(cfg)
		errList         []error
	)

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

func ExpandAndModifyVolumes(ctx context.Context, awsConfig aws.Config, instances map[string]*Target, targets []*Target, incrementPercentage int, bastionClient *ssh.Client) ([]VolumeInstanceMapping, error) {
	instanceLookup := make(map[string]*Target)
	for _, instance := range instances {
		instanceLookup[instance.Id] = instance
	}

	var instanceIds []string
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

	var volumeInstanceMappings []*VolumeInstanceMapping
	for _, volume := range expandedVolumes {
		if instance, ok := instanceLookup[volume.InstanceId]; ok {
			volumeInstanceMappings = append(volumeInstanceMappings, &VolumeInstanceMapping{
				Volume:   volume,
				Instance: instance,
			})
		}
	}

	var volumeInstances []VolumeInstanceMapping
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20)
	var mu sync.Mutex

	for _, mapping := range volumeInstanceMappings {
		wg.Add(1)
		go func(mapping *VolumeInstanceMapping) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			_, err = ModifyLinuxVolumeWithTimeout(func(bastion *ssh.Client, volume *TargetVolume, instance *Target) (string, error) {
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
	wg.Wait()

	return volumeInstances, nil
}

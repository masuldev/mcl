package internal

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"time"
)

type (
	VolumeTarget struct {
	}
)

func CheckVolume() {
	// 볼륨을 전부 가져옴
	// 볼륨에 연결되어 있는 인스턴스에 사용량 조회
	// 볼륨에 연결되어 있는 인스턴스의 연결해야하는데 private ip만 있을 경우 bastion 호스트 선택창을 띄워줌
	// bastion 호스트를 선택 한 경우 bastion 호스트를 통해 사용량 조회에 들어감
	// 사용량이 일정이상 넘어가면 용량 증설
	// 완료되면 작업한 애들 목록 내보냄
}

type (
	TargetVolume struct {
		Id         string
		Size       int32
		InstanceId string
		Device     string
	}
)

func FindVolume(ctx context.Context, cfg aws.Config, instanceIds []string) (map[string]*TargetVolume, error) {
	var (
		client     = ec2.NewFromConfig(cfg)
		table      = make(map[string]*TargetVolume)
		outputFunc = func(table map[string]*TargetVolume, output *ec2.DescribeVolumesOutput) {
			for _, volume := range output.Volumes {
				if len(volume.Attachments) > 0 {
					for _, instanceId := range instanceIds {
						if aws.ToString(volume.Attachments[0].InstanceId) == instanceId {
							table[fmt.Sprintf("%s\t", instanceId)] = &TargetVolume{
								Id:         aws.ToString(volume.Attachments[0].VolumeId),
								Size:       aws.ToInt32(volume.Size),
								InstanceId: aws.ToString(volume.Attachments[0].InstanceId),
								Device:     aws.ToString(volume.Attachments[0].Device),
							}
						}
					}
				}
			}
		}
	)

	volumes, err := FindVolumes(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for len(volumes) > 0 {
		max := len(volumes)

		if max >= 200 {
			max = 199
		}
		output, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			Filters: []types.Filter{
				{Name: aws.String("volume-id"), Values: volumes[:max]},
			},
		})
		if err != nil {
			return nil, err
		}

		outputFunc(table, output)
		volumes = volumes[max:]
	}
	return table, nil
}

func FindVolumes(ctx context.Context, cfg aws.Config) ([]string, error) {
	var (
		volumes    []string
		client     = ec2.NewFromConfig(cfg)
		outputFunc = func(volumes []string, output *ec2.DescribeVolumesOutput) []string {
			for _, volume := range output.Volumes {
				volumes = append(volumes, aws.ToString(volume.VolumeId))
			}
			return volumes
		}
	)

	output, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{MaxResults: aws.Int32(maxOutputResults)})
	if err != nil {
		return nil, err
	}

	volumes = outputFunc(volumes, output)

	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}

			nextOutput, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
				NextToken:  aws.String(token),
				MaxResults: aws.Int32(maxOutputResults),
			})
			if err != nil {
				return nil, err
			}

			volumes = outputFunc(volumes, nextOutput)

			token = aws.ToString(nextOutput.NextToken)
		}
	}

	return volumes, nil
}

func ExpandVolume(ctx context.Context, cfg aws.Config, volumes map[string]*TargetVolume, incrementPercentage int) ([]*TargetVolume, error) {
	var (
		expandedVolumes []*TargetVolume
		client          = ec2.NewFromConfig(cfg)
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
			return nil, fmt.Errorf("error modifying volume size: %v", err)
		}

		err = waitUntilVolumeAvailable(ctx, client, volume.Id)
		if err != nil {
			return nil, fmt.Errorf("error waiting for volume to be available: %v", err)
		}

		expandedVolumes = append(expandedVolumes, volume)
	}

	return expandedVolumes, nil
}

func waitUntilVolumeAvailable(ctx context.Context, client *ec2.Client, volumeId string) error {
	var (
		describeVolumesModificationsInput = &ec2.DescribeVolumesModificationsInput{
			VolumeIds: []string{volumeId},
		}
	)

	for {
		output, err := client.DescribeVolumesModifications(ctx, describeVolumesModificationsInput)
		if err != nil {
			return fmt.Errorf("error describing volume: %v", err)
		}

		if len(output.VolumesModifications) > 0 {
			fmt.Println(output.VolumesModifications[0].ModificationState)
			if output.VolumesModifications[0].ModificationState == types.VolumeModificationStateOptimizing {
				fmt.Println(output.VolumesModifications[0].ModificationState)
				break
			}
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

package internal

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

func FindVolume(ctx context.Context, cfg aws.Config, instanceId string) (*TargetVolume, error) {
	var (
		client       = ec2.NewFromConfig(cfg)
		targetVolume *TargetVolume
		outputFunc   = func(targetVolume *TargetVolume, output *ec2.DescribeVolumesOutput) {
			for _, volume := range output.Volumes {
				if len(volume.Attachments) > 0 {
					if aws.ToString(volume.Attachments[0].InstanceId) == instanceId {
						targetVolume = &TargetVolume{
							Id:         aws.ToString(volume.Attachments[0].VolumeId),
							Size:       aws.ToInt32(volume.Size),
							InstanceId: aws.ToString(volume.Attachments[0].InstanceId),
							Device:     aws.ToString(volume.Attachments[0].Device),
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

		outputFunc(targetVolume, output)
		volumes = volumes[max:]
	}
	return targetVolume, nil
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

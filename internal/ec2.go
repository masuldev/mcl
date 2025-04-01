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
	Target struct {
		Id        string
		Name      string
		PublicIp  string
		PrivateIp string
		Group     string
		KeyName   string
	}
)

func FindInstance(ctx context.Context, cfg aws.Config) (map[string]*Target, error) {
	client := ec2.NewFromConfig(cfg)
	table := make(map[string]*Target)

	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(maxOutputResults),
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				var name, group string
				for _, tag := range instance.Tags {
					switch aws.ToString(tag.Key) {
					case "Name":
						name = aws.ToString(tag.Value)
					case "Server-Group":
						group = aws.ToString(tag.Value)
					}
				}

				instanceId := aws.ToString(instance.InstanceId)
				table[instanceId] = &Target{
					Id:        instanceId,
					Name:      name,
					PublicIp:  aws.ToString(instance.PublicIpAddress),
					PrivateIp: aws.ToString(instance.PrivateIpAddress),
					Group:     group,
					KeyName:   aws.ToString(instance.KeyName),
				}
			}
		}
	}
	return table, nil
}

func GetInstancesWithHighUsage(instances map[string]*Target, bastionClient *ssh.Client, thresholdPercentage int) ([]*Target, map[*Target]int, error) {
	type usageResult struct {
		target *Target
		usage  int
	}

	resultChan := make(chan usageResult, len(instances))
	semaphore := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, instance := range instances {
		wg.Add(1)
		go func(instance *Target) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			usage, err := GetVolumeUsageWithTimeout(func(bastion *ssh.Client, target *Target) (int, error) {
				return GetVolumeUsage(bastion, target)
			}, 10*time.Second, bastionClient, instance)
			if err != nil {
				PrintError(WrapError(fmt.Errorf("cannot get volume usage for instance id %s: %v", instance.Id, err)))
				return
			}

			if usage > thresholdPercentage {
				resultChan <- usageResult{target: instance, usage: usage}
			}
		}(instance)
	}
	wg.Wait()
	close(resultChan)

	var targets []*Target
	instanceUsageMapping := make(map[*Target]int)
	for res := range resultChan {
		targets = append(targets, res.target)
		instanceUsageMapping[res.target] = res.usage
	}

	return targets, instanceUsageMapping, nil
}

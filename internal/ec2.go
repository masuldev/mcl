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
	var (
		client     = ec2.NewFromConfig(cfg)
		table      = make(map[string]*Target)
		outputFunc = func(table map[string]*Target, output *ec2.DescribeInstancesOutput) {
			for _, reservation := range output.Reservations {
				for _, instance := range reservation.Instances {
					name := ""
					group := ""
					for _, tag := range instance.Tags {
						if aws.ToString(tag.Key) == "Name" {
							name = aws.ToString(tag.Value)
							break
						}
					}

					for _, tag := range instance.Tags {
						if aws.ToString(tag.Key) == "Server-Group" {
							group = aws.ToString(tag.Value)
							break
						}
					}

					table[fmt.Sprintf("%s\t(%s)", name, *instance.InstanceId)] = &Target{
						Id:        aws.ToString(instance.InstanceId),
						Name:      name,
						PublicIp:  aws.ToString(instance.PublicIpAddress),
						PrivateIp: aws.ToString(instance.PrivateIpAddress),
						Group:     group,
						KeyName:   aws.ToString(instance.KeyName),
					}
				}
			}
		}
	)

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(maxOutputResults),
		Filters: []types.Filter{
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})

	if err != nil {
		return nil, err
	}

	outputFunc(table, output)

	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}

			nextOutput, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				NextToken:  aws.String(token),
				MaxResults: aws.Int32(maxOutputResults),
				Filters: []types.Filter{
					{Name: aws.String("instance-state-name"), Values: []string{"running"}},
				},
			})

			if err != nil {
				return nil, err
			}

			outputFunc(table, nextOutput)
			token = aws.ToString(nextOutput.NextToken)
		}
	}

	return table, nil
}

func GetInstancesWithHighUsage(instances map[string]*Target, bastionClient *ssh.Client, thresholdPercentage int) ([]*Target, map[*Target]int, error) {
	var (
		targets []*Target
		wg      sync.WaitGroup
	)
	instanceIdUsageMapping := make(map[*Target]int)
	semaphore := make(chan struct{}, 20)

	mu := &sync.Mutex{}
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
				PrintError(WrapError(fmt.Errorf("cannot get volume usage %s, instance id: %s", err, instance.Id)))
				return
			}

			if (usage != 0) && (usage > thresholdPercentage) {
				mu.Lock()
				targets = append(targets, instance)
				instanceIdUsageMapping[instance] = usage
				mu.Unlock()
			}
		}(instance)
	}
	wg.Wait()

	return targets, instanceIdUsageMapping, nil
}

package internal

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type (
	Target struct {
		Id        string
		Name      string
		PublicIp  string
		PrivateIp string
		Group     string
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
					}
				}
			}
		}
	)

	instances, err := FindInstanceIds(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for len(instances) > 0 {
		max := len(instances)

		if max >= 200 {
			max = 199
		}
		output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{Name: aws.String("instance-state-name"), Values: []string{"running"}},
				{Name: aws.String("instance-id"), Values: instances[:max]},
			},
		})
		if err != nil {
			return nil, err
		}

		outputFunc(table, output)
		instances = instances[max:]
	}
	return table, nil
}

func FindInstanceIds(ctx context.Context, cfg aws.Config) ([]string, error) {
	var (
		instances  []string
		client     = ec2.NewFromConfig(cfg)
		outputFunc = func(instances []string, output *ec2.DescribeInstancesOutput) []string {
			for _, reservation := range output.Reservations {
				for _, instance := range reservation.Instances {
					instances = append(instances, aws.ToString(instance.InstanceId))
				}
			}
			return instances
		}
	)

	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{MaxResults: aws.Int32(maxOutputResults)})
	if err != nil {
		return nil, err
	}

	instances = outputFunc(instances, output)

	if aws.ToString(output.NextToken) != "" {
		token := aws.ToString(output.NextToken)
		for {
			if token == "" {
				break
			}

			nextOutput, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				NextToken:  aws.String(token),
				MaxResults: aws.Int32(maxOutputResults),
			})
			if err != nil {
				return nil, err
			}
			instances = outputFunc(instances, nextOutput)

			token = aws.ToString(nextOutput.NextToken)
		}
	}
	return instances, nil
}

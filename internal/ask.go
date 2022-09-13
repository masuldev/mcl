package internal

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
	"sort"
)

const (
	maxOutputResults = 30
)

var (
	defaultAwsRegions = []string{
		"af-south-1",
		"ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3",
		"ca-central-1",
		"eu-central-1", "eu-north-1", "eu-south-1", "eu-west-1", "eu-west-2", "eu-west-3",
		"me-south-1",
		"sa-east-1",
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	}

	defaultCertificateTime = []string{
		"5m", "10m", "30m", "60m", "3h", "6h", "12h",
	}
)

type (
	Target struct {
		Id        string
		Name      string
		PublicIp  string
		PrivateIp string
		Group     string
	}

	User struct {
		Name string
	}

	Region struct {
		Name string
	}

	Port struct {
		Remote string
		Local  string
	}

	Time struct {
		Name string
	}
)

func AskTime() (*Time, error) {
	var time string

	prompt := &survey.Select{
		Message: "Choose a time for Certificate Duration",
		Options: defaultCertificateTime,
	}

	if err := survey.AskOne(prompt, &time, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}
	return &Time{Name: time}, nil
}

func AskTarget(ctx context.Context, cfg aws.Config) (*Target, error) {
	table, err := FindInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	options := make([]string, 0, len(table))
	for k, _ := range table {
		options = append(options, k)
	}
	sort.Strings(options)
	if len(options) == 0 {
		return nil, fmt.Errorf("not found ec2 instance")
	}

	prompt := &survey.Select{
		Message: "Choose a target in AWS:",
		Options: options,
	}

	selectKey := ""
	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return table[selectKey], nil
}

func AskRegion(ctx context.Context, cfg aws.Config) (*Region, error) {
	var regions []string
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	})

	if err != nil {
		regions = make([]string, len(defaultAwsRegions))
		copy(regions, defaultAwsRegions)
	} else {
		regions = make([]string, len(output.Regions))
		for _, region := range output.Regions {
			regions = append(regions, aws.ToString(region.RegionName))
		}
	}

	sort.Strings(regions)

	var region string
	prompt := &survey.Select{
		Message: "Choose a region in AWS:",
		Options: regions,
	}

	if err := survey.AskOne(prompt, &region, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return &Region{Name: region}, nil
}

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

func PrintReady(cmd, region, name, id, publicIp, privateIp string) {
	fmt.Printf("%s: region: %s, name: %s, id: %s, publicIp: %s, privateIp: %s\n", color.CyanString(cmd), color.YellowString(region), color.YellowString(name), color.YellowString(id), color.BlueString(publicIp), color.BlueString(privateIp))
}

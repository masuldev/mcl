package internal

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

	Function struct {
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

func AskBastion(ctx context.Context, cfg aws.Config) (*Target, error) {
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
		Message: "Choose a bastion in AWS:",
		Options: options,
	}

	selectKey := ""
	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return table[selectKey], err
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

func AskVolume(ctx context.Context, cfg aws.Config) (*Function, error) {
	functions := []string{"Check", "Expansion"}

	var function string
	prompt := &survey.Select{
		Message: "Choose a function: ",
		Options: functions,
	}

	if err := survey.AskOne(prompt, &function, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return nil, err
	}

	return &Function{Name: function}, nil
}

func PrintEc2(cmd, region, name, id, publicIp, privateIp string) {
	fmt.Printf("%s: region: %s, name: %s, id: %s, publicIp: %s, privateIp: %s\n", color.CyanString(cmd), color.YellowString(region), color.YellowString(name), color.YellowString(id), color.BlueString(publicIp), color.BlueString(privateIp))
}

func PrintVolumeCheck(cmd, instanceId, instanceName string, usage int) {
	fmt.Printf("%s: instance id: %s, instance name: %s, usage: %s\n", color.CyanString(cmd), color.YellowString(instanceId), color.YellowString(instanceName), color.GreenString("%d", usage))
}

func PrintVolumeExpand(cmd, instanceId, volumeId string, size int32, newSize int64) {
	fmt.Printf("%s: instance id: %s, volume id: %s, size: %s, newSize: %s\n", color.CyanString(cmd), color.YellowString(instanceId), color.YellowString(volumeId), color.BlueString("%d", size), color.BlueString("%d", newSize))
}

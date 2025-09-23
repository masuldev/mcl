package internal

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/masuldev/mcl/internal/ui"
)

// AskTarget prompts user to select an EC2 instance target
func AskTarget(ctx context.Context, cfg aws.Config) (*Target, error) {
	instances, err := FindInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no EC2 instances found")
	}

	// Create display options
	options := make([]string, 0, len(instances))
	displayMap := make(map[string]*Target, len(instances))

	for _, target := range instances {
		option := fmt.Sprintf("%s (%s)", target.Name, target.Id)
		options = append(options, option)
		displayMap[option] = target
	}
	sort.Strings(options)

	// Use UI service
	promptService := ui.GetDefault()
	selected, err := promptService.Select(ctx, "Choose a target in AWS:", options)
	if err != nil {
		return nil, err
	}

	return displayMap[selected], nil
}

// AskRdsTarget prompts user to select an RDS instance target
func AskRdsTarget(ctx context.Context, cfg aws.Config) (*RdsTarget, error) {
	table, err := FindRdsInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if len(table) == 0 {
		return nil, fmt.Errorf("no RDS instances found")
	}

	// Create display options
	options := make([]string, 0, len(table))
	displayMap := make(map[string]*RdsTarget, len(table))

	for _, target := range table {
		option := fmt.Sprintf("%s (%s) - %s", target.Name, target.Id, target.Engine)
		options = append(options, option)
		displayMap[option] = target
	}
	sort.Strings(options)

	// Use UI service
	promptService := ui.GetDefault()
	selected, err := promptService.Select(ctx, "Choose RDS instance:", options)
	if err != nil {
		return nil, err
	}

	return displayMap[selected], nil
}

// AskS3Bucket prompts user to select an S3 bucket
func AskS3Bucket(ctx context.Context, cfg aws.Config) (*S3Bucket, error) {
	buckets, err := FindS3Buckets(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if len(buckets) == 0 {
		return nil, fmt.Errorf("no S3 buckets found")
	}

	// Create display options
	options := make([]string, 0, len(buckets))
	displayMap := make(map[string]*S3Bucket, len(buckets))

	for _, bucket := range buckets {
		option := fmt.Sprintf("%s (%s)", bucket.Name, bucket.Region)
		options = append(options, option)
		displayMap[option] = bucket
	}

	// Use UI service
	promptService := ui.GetDefault()
	selected, err := promptService.Select(ctx, "Choose S3 bucket:", options)
	if err != nil {
		return nil, err
	}

	return displayMap[selected], nil
}

// AskS3Object prompts user to select an S3 object
func AskS3Object(ctx context.Context, cfg aws.Config, bucketName, prefix string) (*S3Object, error) {
	objects, err := FindS3Objects(ctx, cfg, bucketName, prefix)
	if err != nil {
		return nil, err
	}

	if len(objects) == 0 {
		return nil, fmt.Errorf("no objects found in bucket %s with prefix %s", bucketName, prefix)
	}

	// Create display options
	options := make([]string, 0, len(objects))
	displayMap := make(map[string]*S3Object, len(objects))

	for _, object := range objects {
		sizeStr := FormatBytes(object.Size)
		option := fmt.Sprintf("%s (%s) - %s",
			object.Key,
			sizeStr,
			object.LastModified.Format("2006-01-02 15:04:05"))
		options = append(options, option)
		displayMap[option] = object
	}

	// Use UI service
	promptService := ui.GetDefault()
	selected, err := promptService.Select(ctx, "Choose a S3 object:", options)
	if err != nil {
		return nil, err
	}

	return displayMap[selected], nil
}

// Print functions for backward compatibility
func PrintEc2(cmd, region, name, id, publicIp, privateIp string) {
	LogEC2Instance(cmd, region, name, id, publicIp, privateIp)
}

func PrintVolumeCheck(cmd, instanceId, instanceName, instanceIp string, usage int) {
	LogVolumeUsage(cmd, instanceId, instanceName, instanceIp, usage)
}

func PrintVolumeExpand(cmd, instanceId, instanceName, volumeId string, oldSize int32, newSize int64) {
	LogVolumeExpansion(cmd, instanceId, instanceName, volumeId, int(oldSize), int(newSize))
}

// AskBastion prompts user to select a bastion host
func AskBastion(ctx context.Context, cfg aws.Config) (*Target, error) {
	return AskTarget(ctx, cfg) // Bastion is just another EC2 instance
}

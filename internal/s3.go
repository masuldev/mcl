package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
)

type (
	S3Bucket struct {
		Name         string
		CreationDate time.Time
		Region       string
	}

	S3Object struct {
		Key          string
		Size         int64
		LastModified time.Time
		StorageClass string
		ETag         string
	}
)

const (
	maxS3OutputResults = 1000
)

func FindS3Buckets(ctx context.Context, cfg aws.Config) ([]*S3Bucket, error) {
	client := s3.NewFromConfig(cfg)
	var buckets []*S3Bucket

	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, WrapError(err)
	}

	for _, bucket := range output.Buckets {
		// 버킷의 리전 확인
		region, err := getBucketRegion(ctx, client, aws.ToString(bucket.Name))
		if err != nil {
			// 리전 확인 실패 시 기본 리전 사용
			region = cfg.Region
		}

		buckets = append(buckets, &S3Bucket{
			Name:         aws.ToString(bucket.Name),
			CreationDate: aws.ToTime(bucket.CreationDate),
			Region:       region,
		})
	}

	return buckets, nil
}

func getBucketRegion(ctx context.Context, client *s3.Client, bucketName string) (string, error) {
	output, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", err
	}

	location := string(output.LocationConstraint)
	if location == "" {
		return "us-east-1", nil // 빈 문자열은 us-east-1을 의미
	}
	return location, nil
}

func FindS3Objects(ctx context.Context, cfg aws.Config, bucketName, prefix string) ([]*S3Object, error) {
	client := s3.NewFromConfig(cfg)
	var objects []*S3Object

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: aws.Int32(maxS3OutputResults),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	paginator := s3.NewListObjectsV2Paginator(client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, WrapError(err)
		}

		for _, object := range output.Contents {
			objects = append(objects, &S3Object{
				Key:          aws.ToString(object.Key),
				Size:         aws.ToInt64(object.Size),
				LastModified: aws.ToTime(object.LastModified),
				StorageClass: string(object.StorageClass),
				ETag:         aws.ToString(object.ETag),
			})
		}
	}

	return objects, nil
}

func AskS3Bucket(ctx context.Context, cfg aws.Config) (*S3Bucket, error) {
	buckets, err := FindS3Buckets(ctx, cfg)
	if err != nil {
		return nil, WrapError(err)
	}

	if len(buckets) == 0 {
		return nil, WrapError(fmt.Errorf("no S3 buckets found"))
	}

	displayMap := make(map[string]*S3Bucket, len(buckets))
	options := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		option := fmt.Sprintf("%s (%s) - Created: %s",
			bucket.Name,
			bucket.Region,
			bucket.CreationDate.Format("2006-01-02 15:04:05"))
		options = append(options, option)
		displayMap[option] = bucket
	}

	var selectKey string
	prompt := &survey.Select{
		Message: "Choose a S3 bucket:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, WrapError(err)
	}

	return displayMap[selectKey], nil
}

func AskS3Object(ctx context.Context, cfg aws.Config, bucketName string, prefix string) (*S3Object, error) {
	objects, err := FindS3Objects(ctx, cfg, bucketName, prefix)
	if err != nil {
		return nil, WrapError(err)
	}

	if len(objects) == 0 {
		return nil, WrapError(fmt.Errorf("no objects found in bucket %s with prefix %s", bucketName, prefix))
	}

	displayMap := make(map[string]*S3Object, len(objects))
	options := make([]string, 0, len(objects))
	for _, object := range objects {
		sizeStr := FormatBytes(object.Size)
		option := fmt.Sprintf("%s (%s) - %s",
			object.Key,
			sizeStr,
			object.LastModified.Format("2006-01-02 15:04:05"))
		options = append(options, option)
		displayMap[option] = object
	}

	var selectKey string
	prompt := &survey.Select{
		Message: "Choose a S3 object:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, WrapError(err)
	}

	return displayMap[selectKey], nil
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func PrintS3Bucket(service, region, bucketName, creationDate string) {
	color.Cyan("service: %s", service)
	color.Cyan("region: %s", region)
	color.Cyan("bucket: %s", bucketName)
	color.Cyan("created: %s", creationDate)
}

func PrintS3Object(service, region, bucketName, objectKey, size, lastModified string) {
	color.Cyan("service: %s", service)
	color.Cyan("region: %s", region)
	color.Cyan("bucket: %s", bucketName)
	color.Cyan("object: %s", objectKey)
	color.Cyan("size: %s", size)
	color.Cyan("last modified: %s", lastModified)
}

// 페이징을 지원하는 S3 버킷 조회
func FindS3BucketsWithPaging(ctx context.Context, cfg aws.Config, page int) ([]*S3Bucket, error) {
	client := s3.NewFromConfig(cfg)
	var buckets []*S3Bucket

	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, WrapError(err)
	}

	// 페이지 계산
	startIndex := (page - 1) * maxS3OutputResults
	endIndex := startIndex + maxS3OutputResults

	if startIndex >= len(output.Buckets) {
		return buckets, nil
	}

	if endIndex > len(output.Buckets) {
		endIndex = len(output.Buckets)
	}

	// 해당 페이지의 버킷들만 처리
	for i := startIndex; i < endIndex; i++ {
		bucket := output.Buckets[i]

		// 버킷의 리전 확인
		region, err := getBucketRegion(ctx, client, aws.ToString(bucket.Name))
		if err != nil {
			// 리전 확인 실패 시 기본 리전 사용
			region = cfg.Region
		}

		buckets = append(buckets, &S3Bucket{
			Name:         aws.ToString(bucket.Name),
			CreationDate: aws.ToTime(bucket.CreationDate),
			Region:       region,
		})
	}

	return buckets, nil
}

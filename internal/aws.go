package internal

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func NewConfig(ctx context.Context, key, secret, session, region, roleArn string) (aws.Config, error) {
	var (
		opts []func(*config.LoadOptions) error
		cfg  aws.Config
		err  error
	)

	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParam)
	}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	// 자격 증명 옵션 추가: key와 secret이 모두 제공된 경우에만 추가
	if key != "" && secret != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, session)))
	}

	// 한 번의 호출로 기본 구성 로드
	cfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	// roleArn이 제공된 경우 AssumeRole Provider 적용
	if roleArn != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(stsClient, roleArn))
	}

	return cfg, nil
}

func NewSharedConfig(ctx context.Context, profile string, sharedConfigFiles, sharedCredentialsFiles []string) (aws.Config, error) {
	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParam)
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithSharedConfigFiles(sharedConfigFiles),
		config.WithSharedCredentialsFiles(sharedCredentialsFiles),
	)

	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	return cfg, nil
}

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

	if key == "" || secret == "" {
		cfg, err = config.LoadDefaultConfig(ctx, opts...)
	} else {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, session)))
		cfg, err = config.LoadDefaultConfig(ctx, opts...)
	}
	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	if roleArn != "" {
		sts := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts, roleArn))
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

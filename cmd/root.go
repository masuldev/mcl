package cmd

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Credential struct {
	awsProfile string
	awsConfig  *aws.Config
}

const (
	defaultProfile = "default"
)

var (
	rootCmd = &cobra.Command{
		Use:   "mcl",
		Short: "mcl is interactive CLI that select AWS Service or Auth Service",
		Long:  "mcl is interactive CLI that select AWS Service or Auth Service",
	}

	version                 string
	credential              *Credential
	credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())
)

func Execute(version string) {
	rootCmd.Version = version

	err := rootCmd.Execute()
	if err != nil {
		internal.RealPanic(err)
	}
}

func checkConfig() {
	credential = &Credential{}

	awsProfile := viper.GetString("profile")
	if awsProfile == "" {
		awsProfile = os.Getenv("AWS_PROFILE")
		if awsProfile == "" {
			awsProfile = defaultProfile
		}
	}
	credential.awsProfile = awsProfile

	awsRegion := viper.GetString("region")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	if credential.awsConfig == nil {
		var temporaryCredentials aws.Credentials
		var temporaryConfig aws.Config

		// 3. 환경변수에 AWS_ACCESS_KEY_ID/SECRET가 있는 경우 먼저 사용합니다.
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
			temporaryConfig, err = internal.NewConfig(ctx,
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				os.Getenv("AWS_SESSION_TOKEN"),
				awsRegion,
				os.Getenv("AWS_ROLE_ARN"))
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(ctx)
			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
				internal.RealPanic(internal.WrapError(fmt.Errorf("invalid global environments: %v", err)))
			}
		} else {
			// 4. 환경변수가 없으면 Shared Config를 사용합니다.
			temporaryConfig, err = internal.NewSharedConfig(ctx, credential.awsProfile, []string{config.DefaultSharedConfigFilename()}, nil)
			if err == nil {
				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(ctx)
			}
			if err != nil || temporaryCredentials.Expired() ||
				temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
				// 보조 Shared Config (credentials 파일 포함)로 재시도합니다.
				temporaryConfig, err = internal.NewSharedConfig(ctx, credential.awsProfile, []string{config.DefaultSharedConfigFilename()}, []string{config.DefaultSharedCredentialsFilename()})
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(ctx)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				if temporaryCredentials.Expired() ||
					temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
					internal.RealPanic(internal.WrapError(err))
				}
				if awsRegion == "" {
					awsRegion = temporaryConfig.Region
				}
			}
		}

		// 5. 임시 크리덴셜 파일을 생성하되, 이미 파일이 존재하면 재작성하지 않습니다.
		mfaCredentialFormat := "[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n"
		temporaryCredentialsString := fmt.Sprintf(mfaCredentialFormat,
			credential.awsProfile,
			temporaryCredentials.AccessKeyID,
			temporaryCredentials.SecretAccessKey,
			temporaryCredentials.SessionToken)

		if _, err := os.Stat(credentialWithTemporary); os.IsNotExist(err) {
			if err := os.WriteFile(credentialWithTemporary, []byte(temporaryCredentialsString), 0600); err != nil {
				internal.RealPanic(internal.WrapError(err))
			}
		}
		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialWithTemporary)

		// 6. 새로운 Shared Config를 로드합니다.
		awsConfig, err := internal.NewSharedConfig(ctx, credential.awsProfile, nil, []string{credentialWithTemporary})
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
		credential.awsConfig = &awsConfig
	}

	// 7. 리전 정보를 업데이트합니다.
	if awsRegion != "" {
		credential.awsConfig.Region = awsRegion
	}

	if credential.awsConfig.Region == "" {
		askRegion, err := internal.AskRegion(ctx, *credential.awsConfig)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
		credential.awsConfig.Region = askRegion.Name
	}
	color.Cyan("region: %s", credential.awsConfig.Region)
}

func init() {
	cobra.OnInitialize(checkConfig)

	rootCmd.PersistentFlags().StringP("profile", "p", "", "profile")
	rootCmd.PersistentFlags().StringP("region", "r", "", "region")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}

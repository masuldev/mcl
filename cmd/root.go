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
	start := time.Now()

	credential = &Credential{}

	awsProfile := viper.GetString("profile")
	if awsProfile == "" {
		if os.Getenv("AWS_PROFILE") != "" {
			awsProfile = os.Getenv("AWS_PROFILE")
		} else {
			awsProfile = defaultProfile
		}
	}
	credential.awsProfile = awsProfile

	awsRegion := viper.GetString("region")

	var err error

	if credential.awsConfig == nil {
		var temporaryCredentials aws.Credentials
		var temporaryConfig aws.Config

		if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
			temporaryConfig, err = internal.NewConfig(context.Background(),
				os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"),
				os.Getenv("AWS_SESSION_TOKEN"), awsRegion, os.Getenv("AWS_ROLE_ARN"))

			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			if err != nil || temporaryCredentials.Expired() || temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
				internal.RealPanic(internal.WrapError(fmt.Errorf("err: invalid global environments %s", err.Error())))
			}
		} else {
			temporaryConfig, err = internal.NewSharedConfig(context.Background(), credential.awsProfile, []string{config.DefaultSharedConfigFilename()}, []string{})
			if err == nil {
				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
			}

			if err != nil || temporaryCredentials.Expired() || temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
				temporaryConfig, err = internal.NewSharedConfig(context.Background(), credential.awsProfile, []string{config.DefaultSharedConfigFilename()}, []string{config.DefaultSharedCredentialsFilename()})
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}

				temporaryCredentials, err = temporaryConfig.Credentials.Retrieve(context.Background())
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}

				if temporaryCredentials.Expired() || temporaryCredentials.AccessKeyID == "" || temporaryCredentials.SecretAccessKey == "" {
					internal.RealPanic(internal.WrapError(err))
				}

				if awsRegion == "" {
					awsRegion = temporaryConfig.Region
				}
			}
		}

		var mfaCredentialFormat = "[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n"

		temporaryCredentialsString := fmt.Sprintf(mfaCredentialFormat, credential.awsProfile, temporaryCredentials.AccessKeyID, temporaryCredentials.SecretAccessKey, temporaryCredentials.SessionToken)
		if err := os.WriteFile(credentialWithTemporary, []byte(temporaryCredentialsString), 0600); err != nil {
			internal.RealPanic(internal.WrapError(err))
		}

		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialWithTemporary)
		awsConfig, err := internal.NewSharedConfig(context.Background(), credential.awsProfile, []string{}, []string{credentialWithTemporary})
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}

		credential.awsConfig = &awsConfig
	}

	if awsRegion != "" {
		credential.awsConfig.Region = awsRegion
	}

	if credential.awsConfig.Region == "" {
		askRegion, err := internal.AskRegion(context.Background(), *credential.awsConfig)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
		credential.awsConfig.Region = askRegion.Name
	}
	color.Cyan("region: %s", credential.awsConfig.Region)
	end := time.Since(start)
	fmt.Println("1초가 걸린 시간:", end)
}

func init() {
	cobra.OnInitialize(checkConfig)

	rootCmd.PersistentFlags().StringP("profile", "p", "", "profile")
	rootCmd.PersistentFlags().StringP("region", "r", "", "region")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}

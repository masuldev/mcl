package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	// credential을 요구하지 않는 명령어들 확인
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" || arg == "--help" || arg == "-h" {
			// credential 없이 실행
			err := rootCmd.Execute()
			if err != nil {
				internal.RealPanic(err)
			}
			return
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		internal.RealPanic(err)
	}
}

// 전역 AWS Config 설정
func SetGlobalAwsConfig(cfg aws.Config) {
	if credential == nil {
		credential = &Credential{}
	}
	credential.awsConfig = &cfg
}

// 전역 Region 설정
func SetGlobalRegion(region string) {
	if credential != nil && credential.awsConfig != nil {
		credential.awsConfig.Region = region
	}
}

// 전역 AWS Config 반환
func GetGlobalAwsConfig() *aws.Config {
	if credential == nil || credential.awsConfig == nil {
		return nil
	}
	return credential.awsConfig
}

// 전역 Region 반환
func GetGlobalRegion() string {
	if credential != nil && credential.awsConfig != nil {
		return credential.awsConfig.Region
	}
	return ""
}

func init() {
	rootCmd.PersistentFlags().StringP("profile", "p", "", "profile")
	rootCmd.PersistentFlags().StringP("region", "r", "", "region")

	// --version 플래그 지원
	rootCmd.Flags().BoolP("version", "v", false, "Print the version and exit")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
}

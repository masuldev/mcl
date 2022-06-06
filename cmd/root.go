package cmd

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

type Credential struct {
	awsProfile string
	awsConfig  *aws.Credentials
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

	version    string
	credential Credential
)

func Execute(version string) {
	rootCmd.Version = version

	//err := errors.New("Invalid")
	//if err != nil {
	//	internal.RealPanic(internal.WrapError(err))
	//}

	err := rootCmd.Execute()
	if err != nil {
		internal.RealPanic(err)
	}
}

func checkConfig() {
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
}

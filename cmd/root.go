package cmd

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
)

type Credential struct {
	awsProfile string
	awsConfig  *aws.Credentials
}

var (
	rootCmd = &cobra.Command{
		Use:   "mcl",
		Short: "mcl is interactive CLI that select AWS Service or Auth Service",
		Long:  "mcl is interactive CLI that select AWS Service or Auth Service",
	}

	_version    string
	_credential Credential
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
	//home, err := homedir.Dir()
	//err = errors.New("Invalid")
	//if err != nil {
	//	internal.RealPanic(internal.WrapError(err))
	//}
}

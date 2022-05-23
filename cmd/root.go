package cmd

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

type Credential struct {
	awsProfile string
	awsConfig  *aws.Credentials
}

var (
	rootCmd = &cobra.Command{
		Use:   "mcl",
		Short: "mcl is interactive CLI tool that select AWS Service or Auth Service or Connecting Ec2 Instance",
		Long:  "mcl is interactive CLI tool that select AWS Service or Auth Service or Connecting Ec2 Instance",
	}
)

func Execute(version string) {
	rootCmd.Version = version

	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

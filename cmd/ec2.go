package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	startEc2Command = &cobra.Command{
		Use:   "ec2",
		Short: "Exec `ec2 list` under AWS with interactive CLI",
		Long:  "Exec `ec2 list` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				target *internal.Target
				err    error
			)
			ctx := context.Background()

			// 전역 AWS Config 사용
			awsConfig := GetGlobalAwsConfig()
			if awsConfig == nil {
				internal.RealPanic(fmt.Errorf("AWS config not initialized"))
			}

			argTarget := strings.TrimSpace(viper.GetString("ec2-target"))
			if argTarget != "" {
				table, err := internal.FindInstance(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				for _, t := range table {
					if t.Id == argTarget {
						target = t
						break
					}
				}
			}

			argGroup := strings.TrimSpace(viper.GetString("ec2-group"))
			if argGroup != "" {
				table, err := internal.FindInstance(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}

				for _, t := range table {
					if t.Group == argGroup {
						target = t
						break
					}
				}
			}

			if target == nil {
				target, err = internal.AskTarget(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			internal.PrintEc2("ec2", awsConfig.Region, target.Name, target.Id, target.PublicIp, target.PrivateIp)
		},
	}
)

func init() {
	startEc2Command.Flags().StringP("target", "t", "", "ec2 instanceId")
	startEc2Command.Flags().StringP("group", "g", "", "ec2 instance server group")
	viper.BindPFlag("ec2-target", startEc2Command.Flags().Lookup("target"))
	viper.BindPFlag("ec2-group", startEc2Command.Flags().Lookup("group"))

	rootCmd.AddCommand(startEc2Command)
}

package cmd

import (
	"context"
	"fmt"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
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

			argTarget := strings.TrimSpace(viper.GetString("ec2-target"))
			if argTarget != "" {
				table, err := internal.FindInstance(ctx, *credential.awsConfig)
				fmt.Println(credential.awsConfig.Region)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				for _, t := range table {
					if t.Name == argTarget {
						target = t
						break
					}
				}
			}

			if target == nil {
				target, err = internal.AskTarget(ctx, *credential.awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			internal.PrintReady("ec2", credential.awsConfig.Region, target.Name)
		},
	}
)

func init() {
	startEc2Command.Flags().StringP("target", "t", "", "ec2 instanceId")
	viper.BindPFlag("ec2-target", startEc2Command.Flags().Lookup("target"))

	rootCmd.AddCommand(startEc2Command)
}

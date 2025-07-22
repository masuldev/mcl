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
	startRdsCommand = &cobra.Command{
		Use:   "rds",
		Short: "Exec `rds list` under AWS with interactive CLI",
		Long:  "Exec `rds list` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				target *internal.RdsTarget
				err    error
			)
			ctx := context.Background()

			// 전역 AWS Config 사용
			awsConfig := GetGlobalAwsConfig()
			if awsConfig == nil {
				internal.RealPanic(fmt.Errorf("AWS config not initialized"))
			}

			argTarget := strings.TrimSpace(viper.GetString("rds-target"))
			if argTarget != "" {
				table, err := internal.FindRdsInstance(ctx, *awsConfig)
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

			if target == nil {
				target, err = internal.AskRdsTarget(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			internal.PrintRds("rds", awsConfig.Region, target.Name, target.Id, target.Endpoint, target.Status, target.Engine)
		},
	}
)

func init() {
	startRdsCommand.Flags().StringP("target", "t", "", "rds instanceId")
	viper.BindPFlag("rds-target", startRdsCommand.Flags().Lookup("target"))

	rootCmd.AddCommand(startRdsCommand)
}

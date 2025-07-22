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
	startElastiCacheCommand = &cobra.Command{
		Use:   "elasticache",
		Short: "Exec `elasticache list` under AWS with interactive CLI",
		Long:  "Exec `elasticache list` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				target *internal.ElastiCacheTarget
				err    error
			)
			ctx := context.Background()

			// 전역 AWS Config 사용
			awsConfig := GetGlobalAwsConfig()
			if awsConfig == nil {
				internal.RealPanic(fmt.Errorf("AWS config not initialized"))
			}

			argTarget := strings.TrimSpace(viper.GetString("elasticache-target"))
			if argTarget != "" {
				table, err := internal.FindElastiCacheCluster(ctx, *awsConfig)
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
				target, err = internal.AskElastiCacheTarget(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			internal.PrintElastiCache("elasticache", awsConfig.Region, target.Name, target.Id, target.Endpoint, target.Status, target.Engine, target.Port)
		},
	}
)

func init() {
	startElastiCacheCommand.Flags().StringP("target", "t", "", "elasticache clusterId")
	viper.BindPFlag("elasticache-target", startElastiCacheCommand.Flags().Lookup("target"))

	rootCmd.AddCommand(startElastiCacheCommand)
}

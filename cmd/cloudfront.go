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
	startCloudFrontCommand = &cobra.Command{
		Use:   "cloudfront",
		Short: "Exec `cloudfront list` under AWS with interactive CLI",
		Long:  "Exec `cloudfront list` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				target *internal.CloudFrontTarget
				err    error
			)
			ctx := context.Background()

			// 전역 AWS Config 사용
			awsConfig := GetGlobalAwsConfig()
			if awsConfig == nil {
				internal.RealPanic(fmt.Errorf("AWS config not initialized"))
			}

			argTarget := strings.TrimSpace(viper.GetString("cloudfront-target"))
			if argTarget != "" {
				table, err := internal.FindCloudFrontDistribution(ctx, *awsConfig)
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
				target, err = internal.AskCloudFrontTarget(ctx, *awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			// invalidation 옵션이 있는지 확인
			invalidation := viper.GetBool("cloudfront-invalidation")
			if invalidation {
				err = internal.CreateCloudFrontInvalidation(ctx, *awsConfig, target.Id)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				internal.PrintCloudFrontInvalidation("cloudfront", target.Id)
			}

			internal.PrintCloudFront("cloudfront", awsConfig.Region, target.Name, target.Id, target.Domain, target.Status, target.Comment, target.Aliases)
		},
	}
)

func init() {
	startCloudFrontCommand.Flags().StringP("target", "t", "", "cloudfront distributionId")
	startCloudFrontCommand.Flags().BoolP("invalidation", "i", false, "create invalidation /* for selected distribution")
	viper.BindPFlag("cloudfront-target", startCloudFrontCommand.Flags().Lookup("target"))
	viper.BindPFlag("cloudfront-invalidation", startCloudFrontCommand.Flags().Lookup("invalidation"))

	rootCmd.AddCommand(startCloudFrontCommand)
}

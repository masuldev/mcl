package cmd

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

type (
	VolumeInstanceMapping struct {
		Instance *internal.Target
		Volume   *internal.TargetVolume
	}
)

var (
	startVolumeCommand = &cobra.Command{
		Use:   "volume",
		Short: "Exec `volume action` under AWS with interactive CLI",
		Long:  "Exec `volume action` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err error
				//volumes []types.Volume
			)

			ctx := context.Background()

			argFunction := strings.TrimSpace(viper.GetString("volume-function"))
			if argFunction == "" {
				fmt.Println(color.HiMagentaString("# mcl runs with the 'check' option since the '-f' option was not specified."))
			}

			IncrementPercentage := viper.GetInt("volume-increment")
			if IncrementPercentage == 0 {
				IncrementPercentage = 30
			}
			ThresholdPercentage := viper.GetInt("volume-threshold")
			if ThresholdPercentage == 0 {
				ThresholdPercentage = 80
			}

			bastion, err := internal.AskBastion(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			instances, err := internal.FindInstance(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			bastionClient, err := internal.ConnectionBastion(bastion.PublicIp, bastion.KeyName)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			switch argFunction {
			case "check":
				{
					targets, instanceUsageMapping, err := internal.GetInstancesWithHighUsage(instances, bastionClient, ThresholdPercentage)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}

					if len(targets) == 0 {
						fmt.Println("EBS volumes checked and expanded if necessary")
						return
					}

					for _, target := range targets {
						internal.PrintVolumeCheck("volume", target.Id, target.Name, target.PrivateIp, instanceUsageMapping[target])
					}
				}
			case "expand":
				{
					targets, _, err := internal.GetInstancesWithHighUsage(instances, bastionClient, ThresholdPercentage)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}

					volumes, err := internal.ExpandAndModifyVolumes(ctx, *credential.awsConfig, instances, targets, IncrementPercentage, bastionClient)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}

					for _, volume := range volumes {
						internal.PrintVolumeExpand("volume", volume.Instance.Id, volume.Instance.Name, volume.Volume.Id, volume.Volume.Size, volume.Volume.NewSize)
					}
				}
			default:
				{
					targets, instanceUsageMapping, err := internal.GetInstancesWithHighUsage(instances, bastionClient, ThresholdPercentage)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}

					if len(targets) == 0 {
						fmt.Println("EBS volumes checked and expanded if necessary")
						return
					}

					for _, target := range targets {
						internal.PrintVolumeCheck("volume", target.Id, target.Name, target.PrivateIp, instanceUsageMapping[target])
					}
				}
			}
		},
	}
)

func init() {
	startVolumeCommand.Flags().StringP("function", "f", "", "function name")
	startVolumeCommand.Flags().StringP("threshold", "t", "", "volume threshold percentage")
	startVolumeCommand.Flags().StringP("increment", "i", "", "volume increment percentage")
	viper.BindPFlag("volume-function", startVolumeCommand.Flags().Lookup("function"))
	viper.BindPFlag("volume-threshold", startVolumeCommand.Flags().Lookup("threshold"))
	viper.BindPFlag("volume-increment", startVolumeCommand.Flags().Lookup("increment"))

	rootCmd.AddCommand(startVolumeCommand)
}

package cmd

import (
	"context"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
)

var (
	startVolumeCommand = &cobra.Command{
		Use:   "volume",
		Short: "Exec `volume action` under AWS with interactive CLI",
		Long:  "Exec `volume action` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
			//err     error
			//volumes []types.Volume
			)

			ctx := context.Background()

			//function, err := internal.AskVolume(ctx, *credential.awsConfig)
			//if err != nil {
			//	internal.RealPanic(internal.WrapError(err))
			//}
			volumes, err := internal.GetVolumes(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			bastion, err := internal.AskTarget(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			for _, volume := range volumes {
				if len(volume.Attachments) > 0 {
					id := *volume.Attachments[0].InstanceId
					var targetIp string
					table, err := internal.FindInstance(ctx, *credential.awsConfig)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}
					for _, t := range table {
						if t.Id == id {
							targetIp = t.PrivateIp
							break
						}
					}
					err = internal.GetVolumeUsage(bastion.PublicIp, targetIp)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}
				}
			}
		},
	}
)

func init() {
	//startVolumeCommand.Flags().StringP("function", "f", "", "function name")
	//viper.BindPFlag("volume-function", startVolumeCommand.Flags().Lookup("function"))

	rootCmd.AddCommand(startVolumeCommand)
}

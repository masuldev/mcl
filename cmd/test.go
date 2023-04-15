package cmd

import (
	"github.com/spf13/cobra"
)

var (
	startTestCommand = &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			//ctx := context.Background()
			//
			//bastion, err := internal.AskTarget(ctx, *credential.awsConfig)
			//if err != nil {
			//	internal.RealPanic(internal.WrapError(err))
			//}

			//bastionClient, err := internal.ConnectionBastion(bastion.PublicIp, bastion.KeyName)

			//volume, err := internal.TestModifyLinuxVolume(bastionClient, "eqns-development", "10.200.0.207")
			//if err != nil {
			//	internal.RealPanic(internal.WrapError(err))
			//}
			//
			//fmt.Println(volume)
		},
	}
)

func init() {
	rootCmd.AddCommand(startTestCommand)
}

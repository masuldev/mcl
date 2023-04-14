package cmd

import (
	"context"
	"fmt"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"sync"
	"time"
)

const (
	ThresholdPercentage = 80
	IncrementPercentage = 30
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
			instances, err := internal.FindInstance(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			bastion, err := internal.AskTarget(ctx, *credential.awsConfig)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			bastionClient, err := internal.ConnectionBastion(bastion.PublicIp, bastion.KeyName)

			var wg sync.WaitGroup

			mu := &sync.Mutex{}
			var instanceIds []string

			semaphore := make(chan struct{}, 20)
			for _, instance := range instances {
				wg.Add(1)
				go func(instance *internal.Target) {
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					defer wg.Done()

					usage, err := internal.ExecuteWithTimeout(func(bastion *ssh.Client, target *internal.Target) (*internal.VolumeUsage, error) {
						return internal.GetVolumeUsage(bastion, target)
					}, 10*time.Second, bastionClient, instance)
					if err != nil {
						fmt.Println(internal.WrapError(err))
					}

					if (usage != nil) && (usage.Usage > ThresholdPercentage) {
					}
				}(instance)
			}
			wg.Wait()

		},
	}
)

func init() {
	//startVolumeCommand.Flags().StringP("function", "f", "", "function name")
	//viper.BindPFlag("volume-function", startVolumeCommand.Flags().Lookup("function"))

	rootCmd.AddCommand(startVolumeCommand)
}

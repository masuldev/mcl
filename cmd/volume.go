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

					usage, err := internal.GetVolumeUsageWithTimeout(func(bastion *ssh.Client, target *internal.Target) (*internal.VolumeUsage, error) {
						return internal.GetVolumeUsage(bastion, target)
					}, 10*time.Second, bastionClient, instance)
					if err != nil {
						fmt.Println(internal.WrapError(err))
					}

					if (usage != nil) && (usage.Usage > ThresholdPercentage) {
						mu.Lock()
						instanceIds = append(instanceIds, usage.InstanceId)
						mu.Unlock()
					}
				}(instance)
			}
			wg.Wait()

			volumes, err := internal.FindVolume(ctx, *credential.awsConfig, instanceIds)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			expandedVolumes, err := internal.ExpandVolume(ctx, *credential.awsConfig, volumes, IncrementPercentage)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			var volumeInstanceMappings []*VolumeInstanceMapping
			for _, expandedVolume := range expandedVolumes {
				for _, instance := range instances {
					if expandedVolume.InstanceId == instance.Id {
						volumeInstanceMappings = append(volumeInstanceMappings, &VolumeInstanceMapping{
							Volume:   expandedVolume,
							Instance: instance,
						})
					}
				}
			}

			var expandedInstances []string
			for _, volumeInstanceMapping := range volumeInstanceMappings {
				wg.Add(1)
				go func(volumeInstanceMapping *VolumeInstanceMapping) {
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					defer wg.Done()

					expandedInstance, err := internal.ModifyLinuxVolumeWithTimeout(func(bastion *ssh.Client, volume *internal.TargetVolume, instance *internal.Target) (string, error) {
						return internal.ModifyLinuxVolume(bastion, volumeInstanceMapping.Volume, volumeInstanceMapping.Instance)
					}, 10*time.Second, bastionClient, volumeInstanceMapping.Volume, volumeInstanceMapping.Instance)
					if err != nil {
						fmt.Println(internal.WrapError(err))
					}

					expandedInstances = append(expandedInstances, expandedInstance)

				}(volumeInstanceMapping)
			}
			wg.Wait()

			fmt.Println(expandedInstances)
		},
	}
)

func init() {
	//startVolumeCommand.Flags().StringP("function", "f", "", "function name")
	//viper.BindPFlag("volume-function", startVolumeCommand.Flags().Lookup("function"))

	rootCmd.AddCommand(startVolumeCommand)
}

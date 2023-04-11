package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var (
	startVolumeCommand = &cobra.Command{
		Use:   "volume",
		Short: "Exec `volume action` under AWS with interactive CLI",
		Long:  "Exec `volume action` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err error
			)

			ctx := context.Background()

			argFunction := strings.TrimSpace(viper.GetString("volume-function"))
			if argFunction != "" {

			}
		},
	}
)

func init() {
	startVolumeCommand.Flags().StringP("function", "f", "", "function name")
	viper.BindPFlag("volume-function", startVolumeCommand.Flags().Lookup("function"))

	rootCmd.AddCommand(startVolumeCommand)
}

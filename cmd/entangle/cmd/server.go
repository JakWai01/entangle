package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/alphahorizonio/libentangle/pkg/callbacks"
	"github.com/alphahorizonio/libentangle/pkg/handlers"
	"github.com/alphahorizonio/libentangle/pkg/networking"
)

const (
	driveFlag = "drive"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start entangle server instance",
	RunE: func(cmd *cobra.Command, args []string) error {

		onOpen := make(chan struct{})
		manager := handlers.NewClientManager(func() {
			onOpen <- struct{}{}
		})

		cm := networking.NewConnectionManager(manager)

		var file *os.File

		callback := callbacks.NewCallback()

		cm.Connect(viper.GetString(signalFlag), viper.GetString(communityKey), callback.GetServerCallback(*cm, file, viper.GetString(driveFlag)), callback.GetErrorCallback())

		<-onOpen

		select {}

	},
}

func init() {
	dir, err := os.MkdirTemp(os.TempDir(), "serverfiles-*")
	if err != nil {
		panic(err)
	}

	defaultDrive := filepath.Join(dir, "serverfile.tar")

	serverCmd.PersistentFlags().String(driveFlag, defaultDrive, "Specify drive")

	if err := viper.BindPFlags(serverCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	rootCmd.AddCommand(serverCmd)
}

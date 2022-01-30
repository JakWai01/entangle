package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/alphahorizonio/entangle/internal/logging"
	"github.com/alphahorizonio/libentangle/pkg/callbacks"
	"github.com/alphahorizonio/libentangle/pkg/handlers"
	"github.com/alphahorizonio/libentangle/pkg/networking"
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

		l := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		callback := callbacks.NewCallback(l)

		cm.Connect(viper.GetString(signalFlag), viper.GetString(communityKey), callback.GetServerCallback(*cm, file, viper.GetString(driveFlag)), callback.GetErrorCallback(), l)

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

	serverCmd.Flags().String(driveFlag, defaultDrive, "Specify drive")

	if err := viper.BindPFlags(serverCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	rootCmd.AddCommand(serverCmd)
}

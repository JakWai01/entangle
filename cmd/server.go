package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/alphahorizonio/libentangle/pkg/callbacks"
	"github.com/alphahorizonio/libentangle/pkg/handlers"
	"github.com/alphahorizonio/libentangle/pkg/networking"
)

const (
	serverFile = "file"
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

		dir, err := os.MkdirTemp(os.TempDir(), "serverfiles-*")
		if err != nil {
			panic(err)
		}

		myFile := filepath.Join(dir, "serverfile.tar")

		callback := callbacks.NewCallback()

		cm.Connect("test", callback.GetServerCallback(*cm, file, myFile))

		<-onOpen

		select {}

	},
}

func init() {
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()

	rootCmd.AddCommand(serverCmd)
}

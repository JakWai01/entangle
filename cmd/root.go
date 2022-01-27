package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	communityKey = "community"
	verboseFlag  = "verbose"
	metadataFlag = "metadata"
	signalFlag   = "signal"
)

var rootCmd = &cobra.Command{
	Use:   "entangle",
	Short: "A CLI to serve or mount a filesystem",
	Long: `A CLI to serve or mount a filesystem.
	For more information, please visit https://github.com/alphahorizon/entangle`,
}

func Execute() error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(home, ".local", "share", "stfs", "var", "lib", "stfs", "metadata.sqlite")

	rootCmd.PersistentFlags().StringP(communityKey, "c", "test", "Community to join")
	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v)", 2))
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")
	rootCmd.PersistentFlags().StringP(signalFlag, "s", "0.0.0.0:9090", "Address of signaling service")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}

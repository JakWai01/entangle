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
)

var rootCmd = &cobra.Command{
	Use:   "entangle",
	Short: "A library for building peer-to-peer file sharing solutions.",
	Long: `A library for building peer-to-peer file sharing solutions.
	For more information, please visit https://github.com/alphahorizon/libentangle`,
}

func Execute() error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(home, ".local", "share", "stfs", "var", "lib", "stfs", "metadata.sqlite")

	rootCmd.PersistentFlags().String(communityKey, "testCommunityName", "Community to join")
	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v)", 2))
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")
	// Bind env variables
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}

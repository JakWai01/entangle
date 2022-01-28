package cmd

import "github.com/spf13/cobra"

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a backend locally",
	Long: `Mount a backend locally. This can be used to e.g. mount your tape drive as a backend.
	For more information, please visit https://github.com/alphahorizonio/entangle`,
}

func init() {
	rootCmd.AddCommand(mountCmd)
}

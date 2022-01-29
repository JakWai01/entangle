package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/posix"
	"github.com/alphahorizonio/entangle/internal/logging"
	"github.com/jacobsa/fuse"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	storageFlag = "storage"
	mountpointF = "mountp"
)

var osfsCmd = &cobra.Command{
	Use:   "osfs",
	Short: "The osfs backend allows using the default linux filesystem as a backend.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		os.MkdirAll(viper.GetString(storageFlag), os.ModePerm)
		os.MkdirAll(viper.GetString(mountpointF), os.ModePerm)

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpointF), viper.GetString(storageFlag), logger, afero.NewOsFs())

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		fuse.Unmount(viper.GetString(mountpointF))

		mfs, err := fuse.Mount(viper.GetString(mountpointF), serve, cfg)
		if err != nil {
			log.Fatalf("Mount: %v", err)
		}

		if err := mfs.Join(context.Background()); err != nil {
			log.Fatalf("Join %v", err)
		}

		return nil
	},
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	mountPath := filepath.Join(homeDir, filepath.Join("Documents", "mount"))

	osfsCmd.Flags().String(mountpointF, mountPath, "Mountpoint to use for FUSE")

	dir, err := os.MkdirTemp(os.TempDir(), "drive-*")
	if err != nil {
		panic(err)
	}

	defaultStorage := filepath.Join(dir, "storage")
	osfsCmd.Flags().String(storageFlag, defaultStorage, "Declare folder where data is stored")
	if err := viper.BindPFlags(osfsCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.AutomaticEnv()

	mountCmd.AddCommand(osfsCmd)
}

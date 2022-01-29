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
	mountpointFl = "mountPo"
)

var memmapfsCmd = &cobra.Command{
	Use:   "memmapfs",
	Short: "The memmapfs backend allows mounting a in-memory implementation as a backend. There is no way of retrieving data after termination!",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		os.MkdirAll(viper.GetString(mountpointFl), os.ModePerm)

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpointFl), "", logger, afero.NewMemMapFs())

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		fuse.Unmount(viper.GetString(mountpointFl))

		mfs, err := fuse.Mount(viper.GetString(mountpointFl), serve, cfg)
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

	osfsCmd.Flags().String(mountpointFl, mountPath, "Mountpoint to use for FUSE")

	if err := viper.BindPFlags(memmapfsCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	mountCmd.AddCommand(memmapfsCmd)
}

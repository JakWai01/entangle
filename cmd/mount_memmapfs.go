package cmd

import (
	"context"
	"log"
	"os"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/posix"
	"github.com/alphahorizonio/entangle/internal/logging"
	"github.com/jacobsa/fuse"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var memmapfsCmd = &cobra.Command{
	Use:   "memmapfs",
	Short: "The memmapfs backend allows mounting a in-memory implementation as a backend. There is no way of retrieving data after termination!",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		os.MkdirAll(viper.GetString(mountpointFlag), os.ModePerm)

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpointFlag), "", logger, afero.NewMemMapFs())

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		fuse.Unmount(viper.GetString(mountpointFlag))

		mfs, err := fuse.Mount(viper.GetString(mountpointFlag), serve, cfg)
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
	if err := viper.BindPFlags(memmapfsCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	mountCmd.AddCommand(memmapfsCmd)
}

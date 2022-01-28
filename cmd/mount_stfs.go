package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/posix"
	"github.com/alphahorizonio/entangle/internal/logging"
	"github.com/jacobsa/fuse"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/mtio"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	driver     = "driver"
	recordSize = "recordSize"
	writeCache = "writeCache"
	mountpoint = "mountpoint"
)

var stfsCmd = &cobra.Command{
	Use:   "stfs",
	Short: "The stfs backend allows using a tape drive or tar file as a backend.",
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Println(viper.GetString(driver))
		fmt.Println(viper.GetInt(recordSize))
		fmt.Println(viper.GetString(writeCache))
		fmt.Println(viper.GetString(mountpoint))
		fmt.Println(viper.GetString(metadataFlag))

		os.MkdirAll(viper.GetString(mountpoint), os.ModePerm)

		l := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		mt := mtio.MagneticTapeIO{}
		tm := tape.NewTapeManager(
			viper.GetString(driver),
			mt,
			viper.GetInt(recordSize),
			false,
		)

		if err := os.MkdirAll(filepath.Dir(viper.GetString(metadataFlag)), os.ModePerm); err != nil {
			panic(err)
		}

		os.Create(viper.GetString(metadataFlag))

		metadataPersister := persisters.NewMetadataPersister(viper.GetString(metadataFlag))
		if err := metadataPersister.Open(); err != nil {
			panic(err)
		}

		metadataConfig := config.MetadataConfig{
			Metadata: metadataPersister,
		}
		pipeConfig := config.PipeConfig{
			Compression: config.NoneKey,
			Encryption:  config.NoneKey,
			Signature:   config.NoneKey,
			RecordSize:  viper.GetInt(recordSize),
		}
		backendConfig := config.BackendConfig{
			GetWriter:   tm.GetWriter,
			CloseWriter: tm.Close,

			GetReader:   tm.GetReader,
			CloseReader: tm.Close,

			MagneticTapeIO: mt,
		}
		readCryptoConfig := config.CryptoConfig{}

		readOps := operations.NewOperations(
			backendConfig,
			metadataConfig,
			pipeConfig,
			readCryptoConfig,

			func(event *config.HeaderEvent) {
				l.Debug("Header read", event)
			},
		)
		writeOps := operations.NewOperations(
			backendConfig,
			metadataConfig,

			pipeConfig,
			config.CryptoConfig{},

			func(event *config.HeaderEvent) {
				l.Debug("Header write", event)
			},
		)

		stfs := fs.NewSTFS(
			readOps,
			writeOps,

			config.MetadataConfig{
				Metadata: metadataPersister,
			},
			config.CompressionLevelFastestKey,
			func() (cache.WriteCache, func() error, error) {
				return cache.NewCacheWrite(
					viper.GetString(writeCache),
					config.WriteCacheTypeFile,
				)
			},
			false,
			false,
			func(hdr *config.Header) {
				l.Trace("Header transform", hdr)
			},
			l,
		)

		root, err := stfs.Initialize("/", os.ModePerm)
		if err != nil {
			panic(err)
		}

		fs, err := cache.NewCacheFilesystem(
			stfs,
			root,
			config.NoneKey,
			0,
			"",
		)
		if err != nil {
			panic(err)
		}

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpoint), root, l, fs)
		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		fuse.Unmount(viper.GetString(mountpoint))

		mfs, err := fuse.Mount(viper.GetString(mountpoint), serve, cfg)
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

	stfsCmd.Flags().String(mountpoint, mountPath, "Mountpoint to use for FUSE")

	dir, err := os.MkdirTemp(os.TempDir(), "serverfiles-*")
	if err != nil {
		panic(err)
	}

	defaultDrive := filepath.Join(dir, "serverfile.tar")

	stfsCmd.Flags().String(driver, defaultDrive, "Tape drive or tar archive to use as backend")
	stfsCmd.Flags().Int(recordSize, 20, "Amount of 512-bit blocks per second")
	stfsCmd.Flags().String(writeCache, filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")

	if err := viper.BindPFlags(stfsCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	mountCmd.AddCommand(stfsCmd)
}

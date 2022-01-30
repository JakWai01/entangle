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

var stfsCmd = &cobra.Command{
	Use:   "stfs",
	Short: "The stfs backend allows using a tape drive or tar file as a backend.",
	RunE: func(cmd *cobra.Command, args []string) error {

		os.MkdirAll(viper.GetString(mountpointFlag), os.ModePerm)

		l := logging.NewJSONLogger(viper.GetInt(verboseFlag))

		mt := mtio.MagneticTapeIO{}
		tm := tape.NewTapeManager(
			viper.GetString(driveFlag),
			mt,
			viper.GetInt(recordSizeFlag),
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
			RecordSize:  viper.GetInt(recordSizeFlag),
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
					viper.GetString(writeCacheFlag),
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

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpointFlag), root, l, fs)
		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		fuse.Unmount(viper.GetString(mountpointFlag))

		mfs, err := fuse.Mount(viper.GetString(mountpointFlag), serve, cfg)
		if err != nil {
			log.Fatalf("Mount: %v", err)
		}

		log.Println("Mounted STFS")

		if err := mfs.Join(context.Background()); err != nil {
			log.Fatalf("Join %v", err)
		}

		return nil
	},
}

func init() {
	dir, err := os.MkdirTemp(os.TempDir(), "serverfiles-*")
	if err != nil {
		panic(err)
	}

	defaultDrive := filepath.Join(dir, "serverfile.tar")

	stfsCmd.Flags().String(driveFlag, defaultDrive, "Tape drive or tar archive to use as backend")
	stfsCmd.Flags().Int(recordSizeFlag, 20, "Amount of 512-bit blocks per second")
	stfsCmd.Flags().String(writeCacheFlag, filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")

	if err := viper.BindPFlags(stfsCmd.Flags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}

	viper.AutomaticEnv()

	mountCmd.AddCommand(stfsCmd)
}

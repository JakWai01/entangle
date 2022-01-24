package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/helpers"
	"github.com/alphahorizonio/entangle/internal/logging"
	"github.com/jacobsa/fuse"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/fs"
	"github.com/pojntfx/stfs/pkg/mtio"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"

	"github.com/alphahorizonio/libentangle/pkg/callbacks"
	"github.com/alphahorizonio/libentangle/pkg/handlers"
	"github.com/alphahorizonio/libentangle/pkg/networking"
)

const (
	mountpointFlag = "mountpoint"
	recordSizeFlag = "recordSize"
	writeCacheFlag = "writeCache"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start entangle client instance",
	RunE: func(cmd *cobra.Command, args []string) error {

		os.MkdirAll(viper.GetString(mountpointFlag), os.ModePerm)

		onOpen := make(chan struct{})
		manager := handlers.NewClientManager(func() {
			onOpen <- struct{}{}
		})

		cm := networking.NewConnectionManager(manager)

		l := logging.NewJSONLogger(viper.GetInt(verboseFlag))
		boil.DebugMode = true
		boil.DebugWriter = os.Stderr

		rmFile := networking.NewRemoteFile(*cm)

		callback := callbacks.NewCallback()

		go cm.Connect(viper.GetString(signalFlag), viper.GetString(communityKey), callback.GetClientCallback(*rmFile), callback.GetErrorCallback())

		<-onOpen

		mt := mtio.MagneticTapeIO{}

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
			GetWriter: func() (config.DriveWriterConfig, error) {
				if err := rmFile.Open(false); err != nil {
					return config.DriveWriterConfig{}, err
				}

				return config.DriveWriterConfig{
					DriveIsRegular: true,
					Drive:          rmFile,
				}, nil
			},
			CloseWriter: rmFile.Close,

			GetReader: func() (config.DriveReaderConfig, error) {
				if err := rmFile.Open(true); err != nil {
					return config.DriveReaderConfig{}, err
				}

				return config.DriveReaderConfig{
					DriveIsRegular: true,
					Drive:          rmFile,
				}, nil
			},
			CloseReader: rmFile.Close,

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

		serve := filesystem.NewFileSystem(helpers.CurrentUid(), helpers.CurrentGid(), viper.GetString(mountpointFlag), root, l, fs)
		cfg := &fuse.MountConfig{}

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	mountPath := filepath.Join(homeDir, filepath.Join("Documents", "mount"))

	clientCmd.PersistentFlags().String(mountpointFlag, mountPath, "Mountpoint to use for FUSE")
	clientCmd.PersistentFlags().Int(recordSizeFlag, 20, "Amount of 512-bit blocks per second")
	clientCmd.PersistentFlags().String(writeCacheFlag, filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")

	if err := viper.BindPFlags(clientCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.AutomaticEnv()

	rootCmd.AddCommand(clientCmd)
}

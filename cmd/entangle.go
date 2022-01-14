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
	serverFlag     = "server"
)

var entangleCmd = &cobra.Command{
	Use:   "entangle",
	Short: "Start entangle as either a server or a client",
	RunE: func(cmd *cobra.Command, args []string) error {

		onOpen := make(chan struct{})
		manager := handlers.NewClientManager(func() {
			onOpen <- struct{}{}
		})

		cm := networking.NewConnectionManager(manager)

		if viper.GetBool(serverFlag) {
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
		} else {
			l := logging.NewJSONLogger(viper.GetInt(verboseFlag))
			boil.DebugMode = true
			boil.DebugWriter = os.Stderr

			rmFile := networking.NewRemoteFile(*cm)

			callback := callbacks.NewCallback()

			go cm.Connect("test", callback.GetClientCallback(*rmFile))

			<-onOpen

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

				GetDrive: func() (config.DriveConfig, error) {
					if err := rmFile.Open(true); err != nil {
						return config.DriveConfig{}, err
					}

					return config.DriveConfig{
						DriveIsRegular: true,
						Drive:          rmFile,
					}, nil
				},
				CloseDrive: rmFile.Close,
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

			mfs, err := fuse.Mount(viper.GetString(mountpointFlag), serve, cfg)
			if err != nil {
				log.Fatalf("Mount: %v", err)
			}

			if err := mfs.Join(context.Background()); err != nil {
				log.Fatalf("Join %v", err)
			}

			return nil
		}
	},
}

func init() {
	entangleCmd.PersistentFlags().String(mountpointFlag, "/tmp/mount", "Mountpoint to use for FUSE")
	entangleCmd.PersistentFlags().Int(recordSizeFlag, 20, "Amount of 512-bit blocks per second")
	entangleCmd.PersistentFlags().String(writeCacheFlag, filepath.Join(os.TempDir(), "stfs-write-cache"), "Directory to use for write cache")
	entangleCmd.PersistentFlags().Bool(serverFlag, false, "Set true to run as server")

	if err := viper.BindPFlags(entangleCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()

	rootCmd.AddCommand(entangleCmd)
}

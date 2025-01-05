package cmd

import (
	"os"

	"github.com/containifyci/dunebot/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: execute,
}

type storageCmdArgs struct {
	storageFile string
	publicKey   string
	grpcPort    int
	httpPort    int
	debug       bool
}

var storageArgs = &storageCmdArgs{}

func init() {
	tokenCmd.AddCommand(serviceCmd)

	//TODO read the flags from environment variables
	serviceCmd.Flags().StringVar(&storageArgs.storageFile, "storage", "data.json", "The file to persist the tokens between restarts.")
	serviceCmd.Flags().StringVar(&storageArgs.publicKey, "publicKey", "", "The publuc key base64 encode to verify the JWT tokens with.")
	serviceCmd.Flags().IntVar(&storageArgs.grpcPort, "grpcport", 50051, "The port to start the grpc server.")
	serviceCmd.Flags().IntVar(&storageArgs.httpPort, "httpport", 8080, "The port to start the http server.")
	serviceCmd.Flags().BoolVar(&storageArgs.debug, "debug", false, "Enable debug logging.")
}

func execute(cmd *cobra.Command, args []string) {

	initLogger(storageArgs.debug)

	err := storage.StartServers(storage.Config{
		StorageFile:     storageArgs.storageFile,
		PublicKey:       storageArgs.publicKey,
		GRPCPort:        storageArgs.grpcPort,
		HTTPPort:        storageArgs.httpPort,
		PodNamespace:    getenv("POD_NAMESPACE", ""),
		TokenSyncPeriod: getenv("TOKEN_SYNC_PERIOD", "0m"),
	})
	if err != nil {
		panic(err)
	}
}

func initLogger(debug bool) {
	logger := zerolog.New(os.Stdout).With().Caller().Stack().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

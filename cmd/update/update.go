package update

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/containifyci/dunebot/cmd"
	"github.com/containifyci/dunebot/pkg/version"
	"github.com/containifyci/go-self-update/pkg/updater"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "This update the dunebot binary itself.",
	Long:  `This update the dunebot binary itself.`,
	RunE:  execute,
}

func init() {
	cmd.RootCmd.AddCommand(updateCmd)
}

func execute(cmd *cobra.Command, args []string) error {
	logger := zerolog.New(os.Stdout).With().Caller().Stack().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	u := updater.NewUpdater(
		"dunebot", "containifyci", "dunebot", version.GetVersion(),
	)
	updated, err := u.SelfUpdate()
	if err != nil {
		fmt.Printf("Update failed %+v\n", err)
		return err
	}
	if updated {
		fmt.Println("Update completed successfully!")
		return nil
	}
	fmt.Println("Already up-to-date")
	return nil
}

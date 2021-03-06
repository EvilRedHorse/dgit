package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/quorumcontrol/dgit/initializer"
)

func init() {
	rootCmd.AddCommand(initCommand)
}

var initCommand = &cobra.Command{
	Use:   "init",
	Short: "Get rolling with decentragit!",
	// TODO: better explanation
	Long: `Sets up a repo to leverage decentragit.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		callingDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error getting current workdir: %w", err)
			os.Exit(1)
		}

		repo := openRepo(cmd, callingDir)

		client, err := newClient(ctx, repo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		initOpts := &initializer.Options{
			Repo:      repo,
			Tupelo:    client.Tupelo,
			NodeStore: client.Nodestore,
		}
		err = initializer.Init(ctx, initOpts, args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

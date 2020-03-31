package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/quorumcontrol/dgit/initializer"
	"github.com/quorumcontrol/dgit/msg"
	"github.com/quorumcontrol/dgit/transport/dgit"
	"github.com/spf13/cobra"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

func init() {
	rootCmd.AddCommand(initCommand)
}

var initCommand = &cobra.Command{
	Use:   "init",
	Short: "Get rolling with dgit!",
	// TODO: better explanation
	Long: `Sets up an repo to leverage dgit.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		callingDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current workdir: %v", err)
			os.Exit(1)
		}

		repo, err := git.PlainOpenWithOptions(callingDir, &git.PlainOpenOptions{
			DetectDotGit: true,
		})

		if err == git.ErrRepositoryNotExists {
			msg.Print(msg.RepoNotFoundInPath, map[string]interface{}{
				"path": callingDir,
				"cmd":  rootCmd.Name() + " " + cmd.Name(),
			})
			os.Exit(1)
		}

		if err != nil {
			fmt.Printf("Error opening repo: %v", err)
			os.Exit(1)
		}

		repoGitPath := repo.Storer.(*filesystem.Storage).Filesystem().Root()

		client, err := dgit.NewClient(ctx, repoGitPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("error starting dgit client: %v", err))
			os.Exit(1)
		}
		client.RegisterAsDefault()

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

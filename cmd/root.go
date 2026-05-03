package cmd

import (
	"fmt"
	"os"
	"strings"

	askrunner "github.com/oski646/repoq/internal/ask"

	"github.com/spf13/cobra"
)

func NewRootCmd(runner askrunner.Runner) *cobra.Command {
	var question string
	var ref string
	var instructions string
	var showVersion bool

	rootCmd := &cobra.Command{
		Use:           "repoq <github_repository>",
		Short:         "Ask questions about GitHub repositories with a local AI agent CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return cobra.NoArgs(cmd, args)
			}

			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprint(cmd.OutOrStdout(), versionText(cmd.Context()))
				return err
			}

			question = strings.TrimSpace(question)
			if question == "" {
				return fmt.Errorf("--question must not be empty")
			}

			answer, err := runner.Run(cmd.Context(), askrunner.Options{
				Repository:   args[0],
				Question:     question,
				Ref:          ref,
				Instructions: instructions,
				Stdout:       cmd.OutOrStdout(),
				Stderr:       cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "<answer>\n%s\n</answer>\n", answer); err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&question, "question", "q", "", "Question to ask about the repository")
	rootCmd.Flags().StringVar(&ref, "ref", "", "Branch or tag to inspect")
	rootCmd.Flags().StringVar(&instructions, "instructions", "", "Extra instructions for the analysis agent")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show repoq version and latest available version")

	return rootCmd
}

func Execute() {
	rootCmd := NewRootCmd(askrunner.NewRunner())
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

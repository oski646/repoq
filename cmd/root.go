package cmd

import (
	"context"
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

	rootCmd := &cobra.Command{
		Use:           "repoq <github_repository>",
		Short:         "Ask questions about GitHub repositories with Codex",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			answer, err := runner.Run(context.Background(), askrunner.Options{
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
	_ = rootCmd.MarkFlagRequired("question")

	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("question") {
			return nil
		}

		question = strings.TrimSpace(question)
		if question == "" {
			return fmt.Errorf("--question must not be empty")
		}

		return nil
	}

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

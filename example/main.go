package main

import (
	shell "github.com/brianstrauch/cobra-shell"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{Use: "example"}

	subcommand := &cobra.Command{
		Use:       "subcommand",
		Short:     "A subcommand.",
		ValidArgs: []string{"a", "b", "c"},
		Run:       func(_ *cobra.Command, _ []string) {},
	}

	subcommand.Flags().String("flag", "", "A flag.")
	_ = subcommand.RegisterFlagCompletionFunc("flag", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"a", "b", "c"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.AddCommand(subcommand)
	cmd.AddCommand(shell.New(cmd, prompt.OptionPrefixTextColor(prompt.Black)))

	_ = cmd.Execute()
}

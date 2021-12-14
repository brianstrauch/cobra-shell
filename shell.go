package cobrashell

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type cobraShell struct {
	root   *cobra.Command
	prompt *prompt.Prompt
	stdin  *term.State
}

// New creates a Cobra CLI command named "shell" which runs an interactive shell prompt for the root command.
func New(root *cobra.Command, opts ...prompt.Option) *cobra.Command {
	shell := &cobraShell{root: root}

	prefix := fmt.Sprintf("> %s ", root.Name())
	opts = append(opts, prompt.OptionPrefix(prefix), prompt.OptionShowCompletionAtStart())

	shell.prompt = prompt.New(
		shell.executor,
		shell.completer,
		opts...,
	)

	return &cobra.Command{
		Use:   "shell",
		Short: "Start an interactive shell.",
		Run:   shell.run,
	}
}

func (s *cobraShell) executor(line string) {
	args := strings.Fields(line)
	s.root.SetArgs(args)

	// Allow command to read from stdin
	s.restoreStdin()

	_ = s.root.Execute()
}

func (s *cobraShell) completer(d prompt.Document) []prompt.Suggest {
	args, err := buildCompletionArgs(d.CurrentLine())
	if err != nil {
		return nil
	}

	out, err := readCommandOutput(s.root, args)
	if err != nil {
		return nil
	}
	suggestions := parseSuggestions(out)

	return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
}

func buildCompletionArgs(input string) ([]string, error) {
	args, err := shlex.Split(input)

	args = append([]string{"__complete"}, args...)
	if input == "" || input[len(input)-1] == ' ' {
		args = append(args, "")
	}

	return args, err
}

func readCommandOutput(cmd *cobra.Command, args []string) (string, error) {
	buf := new(bytes.Buffer)

	stdout := cmd.OutOrStdout()
	stderr := os.Stderr

	cmd.SetOut(buf)
	_, os.Stderr, _ = os.Pipe()

	cmd.SetArgs(args)
	err := cmd.Execute()

	cmd.SetOut(stdout)
	os.Stderr = stderr

	return buf.String(), err
}

func parseSuggestions(out string) []prompt.Suggest {
	var suggestions []prompt.Suggest

	x := strings.Split(out, "\n")
	for _, line := range x[:len(x)-2] {
		if line != "" {
			x := strings.SplitN(line, "\t", 2)

			var description string
			if len(x) > 1 {
				description = x[1]
			}

			suggestions = append(suggestions, prompt.Suggest{
				Text:        escapeSpecialCharacters(x[0]),
				Description: description,
			})
		}
	}

	return suggestions
}

func escapeSpecialCharacters(val string) string {
	for _, c := range []string{"\\", "\"", "$", "`", "!"} {
		val = strings.ReplaceAll(val, c, "\\"+c)
	}

	if strings.ContainsAny(val, " #&*;<>?[]|~") {
		val = fmt.Sprintf(`"%s"`, val)
	}

	return val
}

func (s *cobraShell) run(cmd *cobra.Command, _ []string) {
	s.editCommandTree(cmd)

	s.saveStdin()
	// TODO: Show persistent flags
	s.prompt.Run()
	s.restoreStdin()
}

func (s *cobraShell) editCommandTree(shell *cobra.Command) {
	s.root.RemoveCommand(shell)

	// Hide the "completion" command
	if cmd, _, err := s.root.Find([]string{"completion"}); err == nil {
		// TODO: Remove this command
		cmd.Hidden = true
	}

	s.root.AddCommand(&cobra.Command{
		Use:   "exit",
		Short: "Exit the interactive shell.",
		Run: func(*cobra.Command, []string) {
			// TODO: Exit cleanly without help from the os package
			os.Exit(0)
		},
	})
}

func (s *cobraShell) saveStdin() {
	state, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	s.stdin = state
}

func (s *cobraShell) restoreStdin() {
	if s.stdin != nil {
		_ = term.Restore(int(os.Stdin.Fd()), s.stdin)
	}
}

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
	root  *cobra.Command
	cache map[string][]prompt.Suggest
	stdin *term.State
}

// New creates a Cobra CLI command named "shell" which runs an interactive shell prompt for the root command.
func New(root *cobra.Command, opts ...prompt.Option) *cobra.Command {
	shell := &cobraShell{
		root:  root,
		cache: make(map[string][]prompt.Suggest),
	}

	prefix := fmt.Sprintf("> %s ", root.Name())
	opts = append(opts, prompt.OptionPrefix(prefix), prompt.OptionShowCompletionAtStart())

	return &cobra.Command{
		Use:   "shell",
		Short: "Start an interactive shell.",
		Run: func(cmd *cobra.Command, _ []string) {
			shell.editCommandTree(cmd)
			shell.saveStdin()

			// TODO: Show persistent flags
			prompt := prompt.New(shell.executor, shell.completer, opts...)
			prompt.Run()

			shell.restoreStdin()
		},
	}
}

func (s *cobraShell) executor(line string) {
	// Allow command to read from stdin
	s.restoreStdin()

	args := strings.Fields(line)
	s.root.SetArgs(args)
	_ = s.root.Execute()

	s.cache = make(map[string][]prompt.Suggest)
}

func (s *cobraShell) completer(d prompt.Document) []prompt.Suggest {
	args, err := buildCompletionArgs(d.CurrentLine())
	if err != nil {
		return nil
	}

	// Clear any partial strings to generate all possible completions
	args[len(args)-1] = ""
	key := strings.Join(args, " ")

	suggestions, ok := s.cache[key]
	if !ok {
		out, err := readCommandOutput(s.root, args)
		if err != nil {
			return nil
		}
		suggestions = parseSuggestions(out)
		s.cache[key] = suggestions
	}

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
	if len(x) < 3 {
		return nil
	}

	for _, line := range x[:len(x)-3] {
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

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/scanner"
)

// completePortArgs provides shell completion for port arguments
// by listing currently active ports.
func completePortArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	opts := scanner.DefaultOptions()
	s, err := scanner.New(opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	listeners, err := s.ListListeners()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	seen := make(map[int]bool)
	for _, l := range listeners {
		if seen[l.Port] {
			continue
		}
		seen[l.Port] = true
		desc := l.ProcessName()
		completions = append(completions, fmt.Sprintf("%d\t%s", l.Port, desc))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeProcessArgs provides shell completion for find pattern arguments
// by listing currently active process names.
func completeProcessArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	opts := scanner.DefaultOptions()
	s, err := scanner.New(opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	listeners, err := s.ListListeners()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	seen := make(map[string]bool)
	var completions []string
	for _, l := range listeners {
		name := l.ProcessName()
		if name != "unknown" && !seen[name] {
			seen[name] = true
			completions = append(completions, name)
		}
		user := l.ProcessUser()
		if user != "-" && !seen[user] {
			seen[user] = true
			completions = append(completions, user)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

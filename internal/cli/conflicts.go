package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/style"
)

// Well-known ports that commonly conflict in development environments.
var commonDevPorts = map[int]string{
	80:    "HTTP",
	443:   "HTTPS",
	3000:  "React/Next.js/Rails",
	3001:  "React alt",
	4000:  "Phoenix/Remix",
	4200:  "Angular",
	5000:  "Flask/Control Center",
	5173:  "Vite",
	5432:  "PostgreSQL",
	5500:  "Live Server",
	6379:  "Redis",
	8000:  "Django/uvicorn",
	8080:  "HTTP alt/Spring",
	8443:  "HTTPS alt",
	8888:  "Jupyter",
	9090:  "Prometheus",
	27017: "MongoDB",
}

func init() {
	RootCmd.AddCommand(conflictsCmd)
}

var conflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Find common port conflicts and duplicate bindings",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newScanner()
		if err != nil {
			return err
		}

		listeners, err := s.ListListeners()
		if err != nil {
			return err
		}

		listeners = model.FilterByProtocol(listeners, tcpOnly, udpOnly)
		return printConflicts(listeners)
	},
}

// Duplicate owners are reported separately for each transport protocol.
func duplicateBindings(listeners []model.Listener) map[string]bool {
	owners := make(map[string]map[int]bool)
	for _, l := range listeners {
		key := fmt.Sprintf("%s:%d", l.Protocol, l.Port)
		if owners[key] == nil {
			owners[key] = make(map[int]bool)
		}
		owners[key][l.PID] = true
	}
	result := make(map[string]bool)
	for key, pids := range owners {
		if len(pids) > 1 {
			result[key] = true
		}
	}
	return result
}
func findConflicts(listeners []model.Listener) []model.Listener {
	duplicates := duplicateBindings(listeners)
	matches := make([]model.Listener, 0)
	for _, l := range listeners {
		_, common := commonDevPorts[l.Port]
		if common || duplicates[fmt.Sprintf("%s:%d", l.Protocol, l.Port)] {
			matches = append(matches, l)
		}
	}
	output.SortListeners(matches, sortBy)
	return matches
}
func printConflicts(listeners []model.Listener) error {
	matches := findConflicts(listeners)
	if jsonOutput || outputFormat != "" {
		return printListeners(matches)
	}
	if len(matches) == 0 {
		fmt.Println(style.Dim.Render("  No common development ports or duplicate owners found."))
		return nil
	}
	duplicates := duplicateBindings(listeners)
	fmt.Println(style.Header.Render("  Development ports and duplicate owners"))
	fmt.Println(style.Separator(50))
	for _, l := range matches {
		note := commonDevPorts[l.Port]
		if duplicates[fmt.Sprintf("%s:%d", l.Protocol, l.Port)] {
			note += " multiple owners"
		}
		fmt.Printf("  %s %-3s %-15s PID %-7d %-20s %s\n", style.PortNum(l.Port), l.Protocol, l.Address, l.PID, l.ProcessName(), note)
	}
	fmt.Println(style.Dim.Render("  Multiple owners may be valid when bound to different addresses or sharing sockets."))
	return nil
}

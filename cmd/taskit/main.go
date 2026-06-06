package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rskulles/taskit/pkg/client"
	"github.com/rskulles/taskit/pkg/tui"
)

func main() {
	var server = flag.String("server", "http://localhost:42069", "taskitd base URL")
	flag.Parse()

	store := client.New(*server)
	m := tui.New(store)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

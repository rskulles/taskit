package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rskulles/taskit/pkg/core"
)

type formSubmitMsg struct{}

// form is a reusable multi-field input form.
// Tab order: text fields → status → blocked reason (only when blocked) → save button.
type form struct {
	title         string
	fields        []textinput.Model
	labels        []string
	statusIdx     int // index into core.AllStatuses()
	blockedReason textinput.Model
	focused       int
	width         int
}

func newForm(title string, width int) form {
	br := textinput.New()
	br.Placeholder = "why is this blocked?"
	br.Width = width - 6
	return form{title: title, width: width, blockedReason: br}
}

func (f *form) addField(label, placeholder, value string) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.Width = f.width - 6
	f.fields = append(f.fields, ti)
	f.labels = append(f.labels, label)
}

func (f *form) focusFirst() {
	if len(f.fields) > 0 {
		f.fields[0].Focus()
		f.focused = 0
	}
}

func (f *form) isBlocked() bool { return f.Status() == core.StatusBlocked }

// totalFields is the number of navigable rows:
// text fields + status + (blocked reason if blocked) + save button.
func (f *form) totalFields() int {
	n := len(f.fields) + 2 // status + save
	if f.isBlocked() {
		n++ // blocked reason
	}
	return n
}

func (f *form) onStatusField() bool {
	return f.focused == len(f.fields)
}

func (f *form) onBlockedReasonField() bool {
	return f.isBlocked() && f.focused == len(f.fields)+1
}

func (f *form) onSaveButton() bool {
	if f.isBlocked() {
		return f.focused == len(f.fields)+2
	}
	return f.focused == len(f.fields)+1
}

func (f *form) Update(msg tea.Msg) (form, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Tab):
			f.focused = (f.focused + 1) % f.totalFields()
			cmds = append(cmds, f.syncFocus()...)

		case key.Matches(msg, keys.ShiftTab):
			f.focused = (f.focused - 1 + f.totalFields()) % f.totalFields()
			cmds = append(cmds, f.syncFocus()...)

		case f.onSaveButton() && key.Matches(msg, keys.Confirm):
			return *f, func() tea.Msg { return formSubmitMsg{} }

		case f.onStatusField():
			switch msg.String() {
			case "left", "h":
				if f.statusIdx > 0 {
					wasBlocked := f.isBlocked()
					f.statusIdx--
					// If we just left blocked and focus is on blocked reason, move back to status
					if wasBlocked && !f.isBlocked() && f.onBlockedReasonField() {
						f.focused = len(f.fields)
					}
				}
			case "right", "l":
				if f.statusIdx < len(core.AllStatuses())-1 {
					f.statusIdx++
				}
			}
		}
	}

	// Route key events to the focused input.
	if f.focused < len(f.fields) {
		var cmd tea.Cmd
		f.fields[f.focused], cmd = f.fields[f.focused].Update(msg)
		cmds = append(cmds, cmd)
	} else if f.onBlockedReasonField() {
		var cmd tea.Cmd
		f.blockedReason, cmd = f.blockedReason.Update(msg)
		cmds = append(cmds, cmd)
	}

	return *f, tea.Batch(cmds...)
}

func (f *form) syncFocus() []tea.Cmd {
	var cmds []tea.Cmd
	for i := range f.fields {
		if i == f.focused {
			cmds = append(cmds, f.fields[i].Focus())
		} else {
			f.fields[i].Blur()
		}
	}
	if f.onBlockedReasonField() {
		cmds = append(cmds, f.blockedReason.Focus())
	} else {
		f.blockedReason.Blur()
	}
	return cmds
}

func (f *form) Value(i int) string {
	if i < len(f.fields) {
		return f.fields[i].Value()
	}
	return ""
}

func (f *form) Status() core.Status {
	statuses := core.AllStatuses()
	if f.statusIdx < len(statuses) {
		return statuses[f.statusIdx]
	}
	return core.StatusNew
}

func (f *form) SetStatus(s core.Status) {
	for i, st := range core.AllStatuses() {
		if st == s {
			f.statusIdx = i
			return
		}
	}
}

func (f *form) BlockedReason() string { return f.blockedReason.Value() }

func (f *form) SetBlockedReason(s string) { f.blockedReason.SetValue(s) }

func (f *form) View() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(f.title))
	sb.WriteString("\n")

	for i, field := range f.fields {
		if f.focused == i {
			sb.WriteString(styleSelected.Render("> " + f.labels[i] + ": "))
		} else {
			sb.WriteString("  " + f.labels[i] + ": ")
		}
		sb.WriteString(field.View())
		sb.WriteString("\n")
	}

	// Status selector row
	if f.onStatusField() {
		sb.WriteString(styleSelected.Render("> Status: "))
	} else {
		sb.WriteString("  Status: ")
	}
	for i, s := range core.AllStatuses() {
		badge := statusBadge(s.String())
		if i == f.statusIdx {
			badge = lipgloss.NewStyle().Underline(true).Render(badge)
		}
		sb.WriteString(badge + " ")
	}
	sb.WriteString("\n")

	// Blocked reason row — only shown when blocked is selected
	if f.isBlocked() {
		if f.onBlockedReasonField() {
			sb.WriteString(styleSelected.Render("> Blocked reason: "))
		} else {
			sb.WriteString("  Blocked reason: ")
		}
		sb.WriteString(f.blockedReason.View())
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Save button row
	if f.onSaveButton() {
		sb.WriteString(styleSelected.Render("> [ Save ]"))
	} else {
		sb.WriteString("  [ Save ]")
	}
	sb.WriteString("\n")

	sb.WriteString(styleHelp.Render("tab/shift+tab: navigate • enter: activate • esc: cancel"))
	return sb.String()
}

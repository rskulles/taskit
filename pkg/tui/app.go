package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rskulles/taskit/pkg/core"
	"github.com/rskulles/taskit/pkg/export"
)

type screen int

const (
	screenProjects screen = iota
	screenFeatures
	screenRequirements
	screenTasks
	screenForm
	screenEmailForm
	screenConfirmDelete
	screenConfirmQuit
	screenDirPicker
)

type mode int

const (
	modeList mode = iota
	modeCreate
	modeEdit
)

// ── list.Item wrappers ────────────────────────────────────────────────────────

type projectItem struct{ p core.Project }
type featureItem struct{ f core.Feature }
type requirementItem struct{ r core.Requirement }
type taskItem struct{ t core.Task }

func (i projectItem) Title() string {
	p := i.p
	return fmt.Sprintf("%s (%d Features, %d Requirements, %d Tasks)", p.Name, p.FeatureCount, p.RequirementCount, p.TaskCount)
}
func (i projectItem) Description() string {
	return itemDescription(i.p.Status, i.p.Description, i.p.CreatedAt, i.p.UpdatedAt)
}
func (i projectItem) FilterValue() string { return i.p.Name }

func (i featureItem) Title() string {
	f := i.f
	return fmt.Sprintf("%s (%d Requirements, %d Tasks)", f.Name, f.RequirementCount, f.TaskCount)
}
func (i featureItem) Description() string {
	return itemDescription(i.f.Status, i.f.Description, i.f.CreatedAt, i.f.UpdatedAt)
}
func (i featureItem) FilterValue() string { return i.f.Name }

func (i requirementItem) Title() string {
	r := i.r
	return fmt.Sprintf("%s (%d Tasks)", r.Name, r.TaskCount)
}
func (i requirementItem) Description() string {
	return itemDescription(i.r.Status, i.r.Description, i.r.CreatedAt, i.r.UpdatedAt)
}
func (i requirementItem) FilterValue() string { return i.r.Name }

func (i taskItem) Title() string { return i.t.Title }
func (i taskItem) Description() string {
	return itemDescription(i.t.Status, i.t.Description, i.t.CreatedAt, i.t.UpdatedAt)
}
func (i taskItem) FilterValue() string { return i.t.Title }

func itemDescription(status core.Status, desc string, created, updated time.Time) string {
	const dateFmt = "Jan 2, 2006"
	dates := styleStatus[status.String()].Render("created " + created.Local().Format(dateFmt))
	if !updated.Equal(created) {
		dates += styleGray.Render("  edited " + updated.Local().Format(dateFmt))
	}
	parts := statusBadge(status.String()) + "  " + dates
	if desc != "" {
		parts += "  " + desc
	}
	return parts
}

// ── tea.Msg types ─────────────────────────────────────────────────────────────

type projectsLoadedMsg struct{ projects []core.Project }
type featuresLoadedMsg struct{ features []core.Feature }
type requirementsLoadedMsg struct{ requirements []core.Requirement }
type tasksLoadedMsg struct{ tasks []core.Task }
type errMsg struct{ err error }
type savedMsg struct{}
type deletedMsg struct{}
type exportedMsg struct{ filename string }
type mailedMsg struct{}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	store core.Store

	screen screen
	mode   mode

	projectList     list.Model
	featureList     list.Model
	requirementList list.Model
	taskList        list.Model

	selectedProject     *core.Project
	selectedFeature     *core.Feature
	selectedRequirement *core.Requirement
	selectedTask        *core.Task

	form       form
	prevScreen screen

	dirPicker       filepicker.Model
	exportingProject *core.Project

	errMsg  string
	infoMsg string
	width   int
	height  int
}

func New(store core.Store) Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(colorPurple)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(colorGray)

	newList := func(title string) list.Model {
		l := list.New(nil, delegate, 0, 0)
		l.Title = title
		l.Styles.Title = styleTitle
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.KeyMap.Quit.SetEnabled(false)
		return l
	}

	return Model{
		store:           store,
		screen:          screenProjects,
		projectList:     newList("Projects"),
		featureList:     newList("Features"),
		requirementList: newList("Requirements"),
		taskList:        newList("Tasks"),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadProjects()
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h := msg.Height - 4
		m.projectList.SetSize(msg.Width, h)
		m.featureList.SetSize(msg.Width, h)
		m.requirementList.SetSize(msg.Width, h)
		m.taskList.SetSize(msg.Width, h)
		m.dirPicker.Height = msg.Height - 6
		return m, nil

	case projectsLoadedMsg:
		items := make([]list.Item, len(msg.projects))
		for i, p := range msg.projects {
			items[i] = projectItem{p}
		}
		m.projectList.SetItems(items)
		return m, nil

	case featuresLoadedMsg:
		items := make([]list.Item, len(msg.features))
		for i, f := range msg.features {
			items[i] = featureItem{f}
		}
		m.featureList.SetItems(items)
		return m, nil

	case requirementsLoadedMsg:
		items := make([]list.Item, len(msg.requirements))
		for i, r := range msg.requirements {
			items[i] = requirementItem{r}
		}
		m.requirementList.SetItems(items)
		return m, nil

	case tasksLoadedMsg:
		items := make([]list.Item, len(msg.tasks))
		for i, t := range msg.tasks {
			items[i] = taskItem{t}
		}
		m.taskList.SetItems(items)
		return m, nil

	case formSubmitMsg:
		if m.screen == screenEmailForm {
			from := m.form.Value(0)
			to := m.form.Value(1)
			m.screen = m.prevScreen
			if m.exportingProject != nil {
				return m, m.doEmailExport(*m.exportingProject, from, to)
			}
			return m, nil
		}
		return m, m.saveForm()

	case savedMsg:
		m.screen = m.prevScreen
		m.errMsg = ""
		return m, m.reloadAll()

	case deletedMsg:
		m.screen = m.prevScreen
		m.errMsg = ""
		return m, m.reloadAll()

	case exportedMsg:
		m.infoMsg = "exported to " + msg.filename
		return m, nil

	case mailedMsg:
		m.infoMsg = "email draft opened"
		return m, nil

	case errMsg:
		m.errMsg = msg.err.Error()
		return m, nil
	}

	switch m.screen {
	case screenForm, screenEmailForm:
		return m.updateForm(msg)
	case screenConfirmDelete:
		return m.updateConfirmDelete(msg)
	case screenConfirmQuit:
		return m.updateConfirmQuit(msg)
	case screenDirPicker:
		return m.updateDirPicker(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) activeList() *list.Model {
	switch m.screen {
	case screenProjects:
		return &m.projectList
	case screenFeatures:
		return &m.featureList
	case screenRequirements:
		return &m.requirementList
	case screenTasks:
		return &m.taskList
	}
	return nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While the list is in filter-input mode, let it handle all keys directly
	// so esc/enter work correctly to cancel or apply the filter.
	if l := m.activeList(); l != nil && l.SettingFilter() {
		var cmd tea.Cmd
		switch m.screen {
		case screenProjects:
			m.projectList, cmd = m.projectList.Update(msg)
		case screenFeatures:
			m.featureList, cmd = m.featureList.Update(msg)
		case screenRequirements:
			m.requirementList, cmd = m.requirementList.Update(msg)
		case screenTasks:
			m.taskList, cmd = m.taskList.Update(msg)
		}
		return m, cmd
	}

	if kmsg, ok := msg.(tea.KeyMsg); ok {
		// Clear notifications on any keypress
		m.errMsg = ""
		m.infoMsg = ""

		switch {
		case key.Matches(kmsg, keys.ForceQuit):
			return m, tea.Quit

		case key.Matches(kmsg, keys.Quit):
			m.prevScreen = m.screen
			m.screen = screenConfirmQuit
			return m, nil

		case key.Matches(kmsg, keys.Back):
			return m.goBack()

		case key.Matches(kmsg, keys.New):
			return m.openCreateForm()

		case key.Matches(kmsg, keys.Edit):
			return m.openEditForm()

		case key.Matches(kmsg, keys.Delete):
			return m.openConfirmDelete()

		case key.Matches(kmsg, keys.Export) && m.screen == screenProjects:
			return m.openDirPicker()

		case key.Matches(kmsg, keys.Mail) && m.screen == screenProjects:
			return m.openEmailForm()

		case key.Matches(kmsg, keys.Enter):
			return m.drillDown()
		}
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenProjects:
		m.projectList, cmd = m.projectList.Update(msg)
	case screenFeatures:
		m.featureList, cmd = m.featureList.Update(msg)
	case screenRequirements:
		m.requirementList, cmd = m.requirementList.Update(msg)
	case screenTasks:
		m.taskList, cmd = m.taskList.Update(msg)
	}
	return m, cmd
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok && key.Matches(kmsg, keys.Cancel) {
		m.screen = m.prevScreen
		return m, nil
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "y", "Y":
			return m, m.deleteSelected()
		case "n", "N", "esc":
			m.screen = m.prevScreen
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateConfirmQuit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "y", "Y":
			return m, tea.Quit
		case "n", "N", "esc":
			m.screen = m.prevScreen
			return m, nil
		}
	}
	return m, nil
}

// ── navigation ────────────────────────────────────────────────────────────────

func (m Model) drillDown() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenProjects:
		item, ok := m.projectList.SelectedItem().(projectItem)
		if !ok {
			return m, nil
		}
		p := item.p
		m.selectedProject = &p
		m.screen = screenFeatures
		return m, m.loadFeatures(p.ID)

	case screenFeatures:
		item, ok := m.featureList.SelectedItem().(featureItem)
		if !ok {
			return m, nil
		}
		f := item.f
		m.selectedFeature = &f
		m.screen = screenRequirements
		return m, m.loadRequirements(f.ID)

	case screenRequirements:
		item, ok := m.requirementList.SelectedItem().(requirementItem)
		if !ok {
			return m, nil
		}
		r := item.r
		m.selectedRequirement = &r
		m.screen = screenTasks
		return m, m.loadTasks(r.ID)

	case screenTasks:
		item, ok := m.taskList.SelectedItem().(taskItem)
		if !ok {
			return m, nil
		}
		t := item.t
		m.selectedTask = &t
		// tasks are leaf — open edit form
		return m.openEditForm()
	}
	return m, nil
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenFeatures:
		m.screen = screenProjects
		m.selectedProject = nil
	case screenRequirements:
		m.screen = screenFeatures
		m.selectedFeature = nil
	case screenTasks:
		m.screen = screenRequirements
		m.selectedRequirement = nil
	}
	return m, nil
}

// ── form open ─────────────────────────────────────────────────────────────────

func (m Model) openCreateForm() (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen
	m.mode = modeCreate
	switch m.screen {
	case screenProjects:
		m.form = newForm("New Project", m.width)
		m.form.addField("Name", "project name", "")
		m.form.addField("Description", "short description", "")
	case screenFeatures:
		m.form = newForm("New Feature", m.width)
		m.form.addField("Name", "feature name", "")
		m.form.addField("Description", "short description", "")
	case screenRequirements:
		m.form = newForm("New Requirement", m.width)
		m.form.addField("Name", "requirement name", "")
		m.form.addField("Description", "short description", "")
	case screenTasks:
		m.form = newForm("New Task", m.width)
		m.form.addField("Title", "task title", "")
		m.form.addField("Description", "short description", "")
	}
	m.form.focusFirst()
	m.screen = screenForm
	return m, nil
}

func (m Model) openEditForm() (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen
	m.mode = modeEdit
	switch m.screen {
	case screenProjects:
		item, ok := m.projectList.SelectedItem().(projectItem)
		if !ok {
			return m, nil
		}
		p := item.p
		m.selectedProject = &p
		m.form = newForm("Edit Project", m.width)
		m.form.addField("Name", "project name", p.Name)
		m.form.addField("Description", "short description", p.Description)
		m.form.SetStatus(p.Status)
		m.form.SetBlockedReason(p.BlockedReason)
	case screenFeatures:
		item, ok := m.featureList.SelectedItem().(featureItem)
		if !ok {
			return m, nil
		}
		f := item.f
		m.selectedFeature = &f
		m.form = newForm("Edit Feature", m.width)
		m.form.addField("Name", "feature name", f.Name)
		m.form.addField("Description", "short description", f.Description)
		m.form.SetStatus(f.Status)
		m.form.SetBlockedReason(f.BlockedReason)
	case screenRequirements:
		item, ok := m.requirementList.SelectedItem().(requirementItem)
		if !ok {
			return m, nil
		}
		r := item.r
		m.selectedRequirement = &r
		m.form = newForm("Edit Requirement", m.width)
		m.form.addField("Name", "requirement name", r.Name)
		m.form.addField("Description", "short description", r.Description)
		m.form.SetStatus(r.Status)
		m.form.SetBlockedReason(r.BlockedReason)
	case screenTasks:
		item, ok := m.taskList.SelectedItem().(taskItem)
		if !ok {
			return m, nil
		}
		t := item.t
		m.selectedTask = &t
		m.form = newForm("Edit Task", m.width)
		m.form.addField("Title", "task title", t.Title)
		m.form.addField("Description", "short description", t.Description)
		m.form.SetStatus(t.Status)
		m.form.SetBlockedReason(t.BlockedReason)
	}
	m.form.focusFirst()
	m.screen = screenForm
	return m, nil
}

func (m Model) openDirPicker() (tea.Model, tea.Cmd) {
	item, ok := m.projectList.SelectedItem().(projectItem)
	if !ok {
		return m, nil
	}
	p := item.p
	m.exportingProject = &p

	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.Getwd()
	fp.DirAllowed = false
	fp.FileAllowed = false
	fp.Height = m.height - 6
	m.dirPicker = fp
	m.prevScreen = m.screen
	m.screen = screenDirPicker
	return m, m.dirPicker.Init()
}

func (m Model) updateDirPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(kmsg, keys.Cancel):
			m.screen = m.prevScreen
			return m, nil
		case key.Matches(kmsg, keys.SelectDir):
			m.screen = m.prevScreen
			return m, m.doExportToDir(m.dirPicker.CurrentDirectory)
		}
	}
	var cmd tea.Cmd
	m.dirPicker, cmd = m.dirPicker.Update(msg)
	return m, cmd
}

func (m Model) doExportToDir(dir string) tea.Cmd {
	if m.exportingProject == nil {
		return nil
	}
	p := *m.exportingProject
	filename := filepath.Join(dir, export.Filename(p.Name))
	return func() tea.Msg {
		f, err := os.Create(filename)
		if err != nil {
			return errMsg{err}
		}
		defer f.Close()
		if err := export.Markdown(context.Background(), m.store, p.ID, f); err != nil {
			return errMsg{err}
		}
		return exportedMsg{filename}
	}
}

func (m Model) openEmailForm() (tea.Model, tea.Cmd) {
	item, ok := m.projectList.SelectedItem().(projectItem)
	if !ok {
		return m, nil
	}
	p := item.p
	m.exportingProject = &p
	m.form = newForm("Email \""+p.Name+"\"", m.width)
	m.form.addField("From", "your@email.com", "")
	m.form.addField("To", "recipient@email.com", "")
	m.form.HideStatus()
	m.form.focusFirst()
	m.prevScreen = m.screen
	m.screen = screenEmailForm
	return m, nil
}

func (m Model) doEmailExport(p core.Project, from, to string) tea.Cmd {
	return func() tea.Msg {
		var buf strings.Builder
		if err := export.HTMLEmail(context.Background(), m.store, p.ID, &buf); err != nil {
			return errMsg{err}
		}
		path := filepath.Join(os.TempDir(), export.EMLFilename(p.Name))
		f, err := os.Create(path)
		if err != nil {
			return errMsg{err}
		}
		export.WriteEML("Status for project "+p.Name, from, to, buf.String(), f)
		f.Close()
		if err := export.OpenFile(path); err != nil {
			return errMsg{err}
		}
		return mailedMsg{}
	}
}

func (m Model) openConfirmDelete() (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = screenConfirmDelete
	return m, nil
}

// ── save / delete ─────────────────────────────────────────────────────────────

func (m Model) saveForm() tea.Cmd {
	switch m.prevScreen {
	case screenProjects:
		if m.mode == modeCreate {
			return func() tea.Msg {
				_, err := m.store.CreateProject(context.Background(), core.Project{
					Name:          m.form.Value(0),
					Description:   m.form.Value(1),
					Status:        m.form.Status(),
					BlockedReason: m.form.BlockedReason(),
				})
				if err != nil {
					return errMsg{err}
				}
				return savedMsg{}
			}
		}
		return func() tea.Msg {
			p := *m.selectedProject
			p.Name = m.form.Value(0)
			p.Description = m.form.Value(1)
			p.Status = m.form.Status()
			p.BlockedReason = m.form.BlockedReason()
			_, err := m.store.UpdateProject(context.Background(), p)
			if err != nil {
				return errMsg{err}
			}
			return savedMsg{}
		}

	case screenFeatures:
		if m.mode == modeCreate {
			return func() tea.Msg {
				_, err := m.store.CreateFeature(context.Background(), core.Feature{
					ProjectID:     m.selectedProject.ID,
					Name:          m.form.Value(0),
					Description:   m.form.Value(1),
					Status:        m.form.Status(),
					BlockedReason: m.form.BlockedReason(),
				})
				if err != nil {
					return errMsg{err}
				}
				return savedMsg{}
			}
		}
		return func() tea.Msg {
			f := *m.selectedFeature
			f.Name = m.form.Value(0)
			f.Description = m.form.Value(1)
			f.Status = m.form.Status()
			f.BlockedReason = m.form.BlockedReason()
			_, err := m.store.UpdateFeature(context.Background(), f)
			if err != nil {
				return errMsg{err}
			}
			return savedMsg{}
		}

	case screenRequirements:
		if m.mode == modeCreate {
			return func() tea.Msg {
				_, err := m.store.CreateRequirement(context.Background(), core.Requirement{
					FeatureID:     m.selectedFeature.ID,
					Name:          m.form.Value(0),
					Description:   m.form.Value(1),
					Status:        m.form.Status(),
					BlockedReason: m.form.BlockedReason(),
				})
				if err != nil {
					return errMsg{err}
				}
				return savedMsg{}
			}
		}
		return func() tea.Msg {
			r := *m.selectedRequirement
			r.Name = m.form.Value(0)
			r.Description = m.form.Value(1)
			r.Status = m.form.Status()
			r.BlockedReason = m.form.BlockedReason()
			_, err := m.store.UpdateRequirement(context.Background(), r)
			if err != nil {
				return errMsg{err}
			}
			return savedMsg{}
		}

	case screenTasks:
		if m.mode == modeCreate {
			return func() tea.Msg {
				_, err := m.store.CreateTask(context.Background(), core.Task{
					RequirementID: m.selectedRequirement.ID,
					Title:         m.form.Value(0),
					Description:   m.form.Value(1),
					Status:        m.form.Status(),
					BlockedReason: m.form.BlockedReason(),
				})
				if err != nil {
					return errMsg{err}
				}
				return savedMsg{}
			}
		}
		return func() tea.Msg {
			t := *m.selectedTask
			t.Title = m.form.Value(0)
			t.Description = m.form.Value(1)
			t.Status = m.form.Status()
			t.BlockedReason = m.form.BlockedReason()
			_, err := m.store.UpdateTask(context.Background(), t)
			if err != nil {
				return errMsg{err}
			}
			return savedMsg{}
		}
	}
	return nil
}

func (m Model) deleteSelected() tea.Cmd {
	switch m.prevScreen {
	case screenProjects:
		item, ok := m.projectList.SelectedItem().(projectItem)
		if !ok {
			return nil
		}
		id := item.p.ID
		return func() tea.Msg {
			if err := m.store.DeleteProject(context.Background(), id); err != nil {
				return errMsg{err}
			}
			return deletedMsg{}
		}
	case screenFeatures:
		item, ok := m.featureList.SelectedItem().(featureItem)
		if !ok {
			return nil
		}
		id := item.f.ID
		return func() tea.Msg {
			if err := m.store.DeleteFeature(context.Background(), id); err != nil {
				return errMsg{err}
			}
			return deletedMsg{}
		}
	case screenRequirements:
		item, ok := m.requirementList.SelectedItem().(requirementItem)
		if !ok {
			return nil
		}
		id := item.r.ID
		return func() tea.Msg {
			if err := m.store.DeleteRequirement(context.Background(), id); err != nil {
				return errMsg{err}
			}
			return deletedMsg{}
		}
	case screenTasks:
		item, ok := m.taskList.SelectedItem().(taskItem)
		if !ok {
			return nil
		}
		id := item.t.ID
		return func() tea.Msg {
			if err := m.store.DeleteTask(context.Background(), id); err != nil {
				return errMsg{err}
			}
			return deletedMsg{}
		}
	}
	return nil
}

// ── reload ────────────────────────────────────────────────────────────────────

// reloadAll refreshes every list in the current navigation path so that parent
// items (which show child counts) stay in sync after any create or delete.
func (m Model) reloadAll() tea.Cmd {
	cmds := []tea.Cmd{m.loadProjects()}
	if m.selectedProject != nil {
		cmds = append(cmds, m.loadFeatures(m.selectedProject.ID))
	}
	if m.selectedFeature != nil {
		cmds = append(cmds, m.loadRequirements(m.selectedFeature.ID))
	}
	if m.selectedRequirement != nil {
		cmds = append(cmds, m.loadTasks(m.selectedRequirement.ID))
	}
	return tea.Batch(cmds...)
}

// ── loaders ───────────────────────────────────────────────────────────────────

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.store.ListProjects(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg{projects}
	}
}

func (m Model) loadFeatures(projectID int64) tea.Cmd {
	return func() tea.Msg {
		features, err := m.store.ListFeatures(context.Background(), projectID)
		if err != nil {
			return errMsg{err}
		}
		return featuresLoadedMsg{features}
	}
}

func (m Model) loadRequirements(featureID int64) tea.Cmd {
	return func() tea.Msg {
		requirements, err := m.store.ListRequirements(context.Background(), featureID)
		if err != nil {
			return errMsg{err}
		}
		return requirementsLoadedMsg{requirements}
	}
}

func (m Model) loadTasks(requirementID int64) tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.store.ListTasks(context.Background(), requirementID)
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	var sb strings.Builder

	if m.screen == screenForm || m.screen == screenEmailForm {
		sb.WriteString(styleBorder.Width(m.width - 4).Render(m.form.View()))
		if m.errMsg != "" {
			sb.WriteString("\n" + styleError.Render("Error: "+m.errMsg))
		}
		return sb.String()
	}

	if m.screen == screenConfirmDelete {
		sb.WriteString(styleTitle.Render("Confirm Delete"))
		sb.WriteString("\nAre you sure you want to delete this item and all its children? ")
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("[y/N]"))
		return sb.String()
	}

	if m.screen == screenConfirmQuit {
		sb.WriteString(styleTitle.Render("Quit"))
		sb.WriteString("\nAre you sure you want to quit? ")
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("[y/N]"))
		return sb.String()
	}

	if m.screen == screenDirPicker {
		name := ""
		if m.exportingProject != nil {
			name = m.exportingProject.Name
		}
		sb.WriteString(styleTitle.Render("Export \"" + name + "\" — select a directory"))
		sb.WriteString("\n")
		sb.WriteString(m.dirPicker.View())
		sb.WriteString("\n" + styleHelp.Render("enter: open directory  space: export here  esc: cancel"))
		return sb.String()
	}

	// breadcrumb
	crumbs := m.breadcrumbs()
	if crumbs != "" {
		sb.WriteString(styleBreadcrumb.Render(crumbs))
		sb.WriteString("\n")
	}

	switch m.screen {
	case screenProjects:
		sb.WriteString(m.projectList.View())
	case screenFeatures:
		sb.WriteString(m.featureList.View())
	case screenRequirements:
		sb.WriteString(m.requirementList.View())
	case screenTasks:
		sb.WriteString(m.taskList.View())
	}

	help := "n: new  e: edit  d: delete  enter: open  esc: back  q: quit  ctrl+c: force quit"
	if m.screen == screenProjects {
		help += "  x: export  m: email"
	}
	sb.WriteString("\n" + styleHelp.Render(help))

	if m.errMsg != "" {
		sb.WriteString("\n" + styleError.Render("Error: "+m.errMsg))
	}
	if m.infoMsg != "" {
		sb.WriteString("\n" + styleGray.Render(m.infoMsg))
	}

	return sb.String()
}

func (m Model) breadcrumbs() string {
	parts := []string{"Projects"}
	if m.selectedProject != nil {
		parts = append(parts, m.selectedProject.Name)
	}
	if m.selectedFeature != nil {
		parts = append(parts, m.selectedFeature.Name)
	}
	if m.selectedRequirement != nil {
		parts = append(parts, m.selectedRequirement.Name)
	}
	if len(parts) == 1 {
		return ""
	}
	return fmt.Sprintf("[ %s ]", strings.Join(parts, " > "))
}

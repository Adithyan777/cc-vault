package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cc-sessions/internal/claude"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Panel represents which panel is active
type Panel int

const (
	PanelProjects Panel = iota
	PanelSessions
	PanelPreview
)

// ResumeMsg is sent when user wants to resume a session
type ResumeMsg struct {
	SessionID  string
	ProjectDir string
}

// searchCompleteMsg is sent when async search finishes
type searchCompleteMsg struct {
	results []SearchResult
	query   string
}

// searchTickMsg drives the spinner animation during search
type searchTickMsg struct{}

// clearStatusMsg clears the status bar message after a delay
type clearStatusMsg struct{}

// Model is the main Bubble Tea model
type Model struct {
	// Data
	projects []claude.Project
	sessions []claude.Session
	preview  *claude.PreviewData
	config   *claude.ClaudeConfig

	// Selection state
	activePanel      Panel
	projectIdx       int
	sessionIdx       int
	previewScroll    int
	previewLineCount int
	previewLines     []string // cached rendered lines

	// Dialog state
	dialog *Dialog

	// Search state
	searching        bool
	searchQuery      string
	searchResults    []SearchResult
	filteredSessions []claude.Session
	searchActive     bool // true when showing filtered search results
	searchInProgress bool
	searchSpinner    int

	// Resume action
	resumeSession *ResumeMsg

	// Screen size
	width  int
	height int

	// Status message
	statusMsg string
}

// NewModel creates a new TUI model
func NewModel() Model {
	config, _ := claude.ReadConfig()
	projects, _ := claude.DiscoverProjects()

	// Find current working directory project
	cwd, _ := os.Getwd()
	projectIdx := claude.FindProjectIndex(projects, cwd)

	m := Model{
		projects:    projects,
		config:      config,
		projectIdx:  projectIdx,
		activePanel: PanelSessions,
	}

	// Load sessions for initial project
	if len(projects) > 0 {
		m.loadSessions()
	}

	return m
}

func (m *Model) loadSessions() {
	if m.projectIdx < 0 || m.projectIdx >= len(m.projects) {
		m.sessions = nil
		m.preview = nil
		return
	}

	project := m.projects[m.projectIdx]
	lastSessionID := ""
	if m.config != nil {
		if pc := m.config.GetProjectConfig(project.FullPath); pc != nil {
			lastSessionID = pc.LastSessionID
		}
	}

	sessions, err := claude.LoadSessions(project.EncodedName, lastSessionID)
	if err != nil {
		m.sessions = nil
		m.preview = nil
		return
	}

	m.sessions = sessions
	m.sessionIdx = 0
	m.searchResults = nil
	m.filteredSessions = nil
	m.searchActive = false
	m.searching = false
	m.searchQuery = ""

	// Load preview for first session
	m.loadPreview()
}

func (m *Model) loadPreview() {
	sessions := m.activeSessions()
	if m.sessionIdx < 0 || m.sessionIdx >= len(sessions) {
		m.preview = nil
		m.previewLines = nil
		return
	}

	session := sessions[m.sessionIdx]
	preview, err := claude.LoadPreview(session.FilePath)
	if err != nil {
		m.preview = nil
		m.previewLines = nil
		return
	}
	m.preview = preview
	m.previewScroll = 0

	// Pre-render the preview lines (once per session change)
	pc := m.currentProjectConfig()
	m.previewLines = buildPreviewLines(preview, &session, pc, m.width/2-6)
}

func (m *Model) activeSessions() []claude.Session {
	if m.searchActive && m.filteredSessions != nil {
		return m.filteredSessions
	}
	return m.sessions
}

func (m *Model) selectedSession() *claude.Session {
	sessions := m.activeSessions()
	if m.sessionIdx < 0 || m.sessionIdx >= len(sessions) {
		return nil
	}
	s := sessions[m.sessionIdx]
	return &s
}

func (m *Model) currentProjectConfig() *claude.ProjectConfig {
	if m.projectIdx < 0 || m.projectIdx >= len(m.projects) {
		return nil
	}
	if m.config == nil {
		return nil
	}
	return m.config.GetProjectConfig(m.projects[m.projectIdx].FullPath)
}

func searchSessionsCmd(sessions []claude.Session, query string) tea.Cmd {
	return func() tea.Msg {
		results := SearchSessions(sessions, query)
		return searchCompleteMsg{results: results, query: query}
	}
}

func searchTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return searchTickMsg{}
	})
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Rebuild preview cache since width changed
		if m.preview != nil {
			session := m.selectedSession()
			pc := m.currentProjectConfig()
			previewWidth := m.width - m.width/4 - m.width/4
			m.previewLines = buildPreviewLines(m.preview, session, pc, previewWidth-6)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case searchCompleteMsg:
		m.searchInProgress = false
		m.searchResults = msg.results
		m.searchQuery = msg.query
		m.searchActive = true
		filtered := make([]claude.Session, 0, len(msg.results))
		for _, r := range msg.results {
			if r.SessionIndex < len(m.sessions) {
				filtered = append(filtered, m.sessions[r.SessionIndex])
			}
		}
		m.filteredSessions = filtered
		m.sessionIdx = 0
		m.activePanel = PanelSessions
		m.loadPreview()
		return m, nil

	case searchTickMsg:
		if m.searchInProgress {
			m.searchSpinner++
			return m, searchTickCmd()
		}
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Block all keys except quit during async search
	if m.searchInProgress {
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		return m, nil
	}

	// Handle dialog input first
	if m.dialog != nil && m.dialog.Type != DialogNone {
		return m.handleDialogKey(key, msg)
	}

	// Handle search input
	if m.searching {
		return m.handleSearchKey(key, msg)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.dialog = &Dialog{Type: DialogHelp}
		return m, nil

	case "esc":
		// Clear search results and restore full session list
		if m.searchActive {
			m.searchResults = nil
			m.filteredSessions = nil
			m.searchActive = false
			m.searchQuery = ""
			m.sessionIdx = 0
			m.loadPreview()
			return m, nil
		}
		return m, nil

	case "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil

	case "tab":
		m.activePanel = (m.activePanel + 1) % 3
		return m, nil

	case "left", "h":
		if m.activePanel > PanelProjects {
			m.activePanel--
		}
		return m, nil

	case "right", "l":
		if m.activePanel < PanelPreview {
			m.activePanel++
		}
		return m, nil

	case "up", "k":
		return m.handleUp()

	case "down", "j":
		return m.handleDown()

	case "enter":
		return m.handleEnter()

	case "r":
		return m.handleRename()

	case "d":
		return m.handleDelete()

	case "x":
		return m.handleExport()

	case " ":
		return m.handleToggleSelect()

	case "D":
		return m.handleBulkDelete()

	case "X":
		return m.handleBulkExport()

	case "P":
		return m.handlePrune()
	}

	return m, nil
}

func (m Model) handleDialogKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dialog == nil {
		return m, nil
	}

	switch m.dialog.Type {
	case DialogHelp:
		if key == "?" || key == "esc" || key == "q" {
			m.dialog = nil
		}
		return m, nil

	case DialogConfirmDelete, DialogConfirmBulkDelete, DialogConfirmPrune:
		switch key {
		case "y", "Y":
			if m.dialog.Type == DialogConfirmDelete {
				return m.executeDelete()
			}
			if m.dialog.Type == DialogConfirmPrune {
				return m.executePrune()
			}
			return m.executeBulkDelete()
		case "n", "N", "esc":
			m.dialog = nil
		}
		return m, nil

	case DialogRename:
		switch key {
		case "esc":
			m.dialog = nil
			return m, nil
		case "enter":
			return m.executeRename()
		case "backspace":
			if len(m.dialog.Input) > 0 {
				m.dialog.Input = m.dialog.Input[:len(m.dialog.Input)-1]
			}
			return m, nil
		default:
			// Only add printable characters
			if len(key) == 1 && key[0] >= 32 {
				m.dialog.Input += key
			} else if key == " " {
				m.dialog.Input += " "
			}
			return m, nil
		}
	}

	return m, nil
}

func (m Model) handleSearchKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.searchResults = nil
		m.filteredSessions = nil
		m.sessionIdx = 0
		m.loadPreview()
		return m, nil

	case "enter":
		if m.searchQuery != "" {
			m.searching = false
			m.searchInProgress = true
			m.searchSpinner = 0
			return m, tea.Batch(searchSessionsCmd(m.sessions, m.searchQuery), searchTickCmd())
		}
		m.searching = false
		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		return m, nil

	default:
		if len(key) == 1 && key[0] >= 32 {
			m.searchQuery += key
		} else if key == " " {
			m.searchQuery += " "
		}
		return m, nil
	}
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case PanelProjects:
		if m.projectIdx > 0 {
			m.projectIdx--
			m.loadSessions()
		}
	case PanelSessions:
		if m.sessionIdx > 0 {
			m.sessionIdx--
			m.loadPreview()
		}
	case PanelPreview:
		if m.previewScroll > 0 {
			m.previewScroll--
		}
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case PanelProjects:
		if m.projectIdx < len(m.projects)-1 {
			m.projectIdx++
			m.loadSessions()
		}
	case PanelSessions:
		sessions := m.activeSessions()
		if m.sessionIdx < len(sessions)-1 {
			m.sessionIdx++
			m.loadPreview()
		}
	case PanelPreview:
		// Scroll will be clamped in the render function
		m.previewScroll++
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.activePanel == PanelProjects {
		m.activePanel = PanelSessions
		return m, nil
	}

	session := m.selectedSession()
	if session == nil {
		return m, nil
	}

	// Get the project directory
	projectDir := ""
	if m.projectIdx >= 0 && m.projectIdx < len(m.projects) {
		projectDir = m.projects[m.projectIdx].FullPath
	}

	m.resumeSession = &ResumeMsg{
		SessionID:  session.ID,
		ProjectDir: projectDir,
	}
	return m, tea.Quit
}

func (m Model) handleRename() (tea.Model, tea.Cmd) {
	if m.activePanel != PanelSessions {
		return m, nil
	}
	session := m.selectedSession()
	if session == nil {
		return m, nil
	}

	m.dialog = &Dialog{
		Type:  DialogRename,
		Input: session.CustomName,
	}
	return m, nil
}

func (m Model) executeRename() (tea.Model, tea.Cmd) {
	session := m.selectedSession()
	if session == nil || m.dialog == nil {
		m.dialog = nil
		return m, nil
	}

	name := strings.TrimSpace(m.dialog.Input)
	// Write custom-title to the session JSONL (Claude's native format)
	// An empty name clears the title (sets it to "")
	claude.WriteCustomTitle(session.FilePath, session.ID, name)

	// Reload sessions to reflect name change
	m.loadSessions()
	m.dialog = nil
	if name != "" {
		m.statusMsg = "Session renamed to \"" + name + "\""
	} else {
		m.statusMsg = "Session name cleared"
	}
	return m, clearStatusCmd()
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	if m.activePanel != PanelSessions {
		return m, nil
	}
	session := m.selectedSession()
	if session == nil {
		return m, nil
	}

	m.dialog = &Dialog{
		Type:    DialogConfirmDelete,
		Message: fmt.Sprintf("Delete session \"%s\"?\nThis will remove all associated files.", session.DisplayName()),
	}
	return m, nil
}

func (m Model) executeDelete() (tea.Model, tea.Cmd) {
	session := m.selectedSession()
	if session == nil {
		m.dialog = nil
		return m, nil
	}

	DeleteSessionFiles(session)


	m.dialog = nil
	m.statusMsg = "Session deleted"
	m.loadSessions()
	return m, clearStatusCmd()
}

func (m Model) handleExport() (tea.Model, tea.Cmd) {
	if m.activePanel != PanelSessions {
		return m, nil
	}
	session := m.selectedSession()
	if session == nil {
		return m, nil
	}

	projectDisplay := ""
	if m.projectIdx >= 0 && m.projectIdx < len(m.projects) {
		projectDisplay = m.projects[m.projectIdx].DisplayPath
	}

	md, err := claude.ExportSession(session, projectDisplay)
	if err != nil {
		m.statusMsg = "Export failed: " + err.Error()
		return m, clearStatusCmd()
	}

	home, _ := os.UserHomeDir()
	name := strings.ReplaceAll(session.DisplayName(), "/", "-")
	name = strings.ReplaceAll(name, " ", "_")
	if len(name) > 50 {
		name = name[:50]
	}
	exportPath := filepath.Join(home, "Desktop", name+".md")

	if err := os.WriteFile(exportPath, []byte(md), 0644); err != nil {
		m.statusMsg = "Export failed: " + err.Error()
		return m, clearStatusCmd()
	}

	m.statusMsg = fmt.Sprintf("Exported to %s", exportPath)
	return m, clearStatusCmd()
}

func (m Model) handleToggleSelect() (tea.Model, tea.Cmd) {
	if m.activePanel != PanelSessions {
		return m, nil
	}
	sessions := m.activeSessions()
	if m.sessionIdx < 0 || m.sessionIdx >= len(sessions) {
		return m, nil
	}

	// Toggle in the main sessions list
	for i := range m.sessions {
		if m.sessions[i].ID == sessions[m.sessionIdx].ID {
			m.sessions[i].Selected = !m.sessions[i].Selected
			break
		}
	}

	// Also update filtered list if active
	if m.filteredSessions != nil && m.sessionIdx < len(m.filteredSessions) {
		m.filteredSessions[m.sessionIdx].Selected = !m.filteredSessions[m.sessionIdx].Selected
	}

	return m, nil
}

func (m Model) handleBulkDelete() (tea.Model, tea.Cmd) {
	var selected []string
	for _, s := range m.sessions {
		if s.Selected {
			selected = append(selected, s.ID)
		}
	}
	if len(selected) == 0 {
		m.statusMsg = "No sessions selected (use Space to select)"
		return m, clearStatusCmd()
	}

	m.dialog = &Dialog{
		Type:       DialogConfirmBulkDelete,
		Message:    fmt.Sprintf("Delete %d selected sessions?\nThis cannot be undone.", len(selected)),
		SessionIDs: selected,
	}
	return m, nil
}

func (m Model) executeBulkDelete() (tea.Model, tea.Cmd) {
	count := 0
	for _, s := range m.sessions {
		if s.Selected {
			session := s
			DeleteSessionFiles(&session)
			count++
		}
	}

	m.dialog = nil
	m.statusMsg = fmt.Sprintf("Deleted %d sessions", count)
	m.loadSessions()
	return m, clearStatusCmd()
}

func (m Model) handlePrune() (tea.Model, tea.Cmd) {
	// Lazily compute conversation counts now (only when user presses P)
	var emptyIDs []string
	for i := range m.sessions {
		if m.sessions[i].ConversationCount < 0 {
			m.sessions[i].ConversationCount = claude.CountConversationMessages(m.sessions[i].FilePath)
		}
		if m.sessions[i].ConversationCount == 0 {
			emptyIDs = append(emptyIDs, m.sessions[i].ID)
		}
	}

	if len(emptyIDs) == 0 {
		m.statusMsg = "No empty sessions to prune"
		return m, clearStatusCmd()
	}

	m.dialog = &Dialog{
		Type:       DialogConfirmPrune,
		Message:    fmt.Sprintf("Found %d empty sessions (0 messages).\nDelete all of them?", len(emptyIDs)),
		SessionIDs: emptyIDs,
	}
	return m, nil
}

func (m Model) executePrune() (tea.Model, tea.Cmd) {
	count := 0
	for _, s := range m.sessions {
		if s.ConversationCount == 0 {
			session := s
			DeleteSessionFiles(&session)
			count++
		}
	}

	m.dialog = nil
	m.statusMsg = fmt.Sprintf("Pruned %d empty sessions", count)
	m.loadSessions()
	return m, clearStatusCmd()
}

func (m Model) handleBulkExport() (tea.Model, tea.Cmd) {
	var selected []claude.Session
	for _, s := range m.sessions {
		if s.Selected {
			selected = append(selected, s)
		}
	}
	if len(selected) == 0 {
		m.statusMsg = "No sessions selected (use Space to select)"
		return m, clearStatusCmd()
	}

	projectDisplay := ""
	if m.projectIdx >= 0 && m.projectIdx < len(m.projects) {
		projectDisplay = m.projects[m.projectIdx].DisplayPath
	}

	home, _ := os.UserHomeDir()
	count := 0
	for _, s := range selected {
		session := s
		md, err := claude.ExportSession(&session, projectDisplay)
		if err != nil {
			continue
		}
		name := strings.ReplaceAll(session.DisplayName(), "/", "-")
		name = strings.ReplaceAll(name, " ", "_")
		if len(name) > 50 {
			name = name[:50]
		}
		exportPath := filepath.Join(home, "Desktop", name+".md")
		if err := os.WriteFile(exportPath, []byte(md), 0644); err == nil {
			count++
		}
	}

	m.statusMsg = fmt.Sprintf("Exported %d sessions to ~/Desktop/", count)
	return m, clearStatusCmd()
}

// GetResumeSession returns the session to resume after quit, if any
func (m *Model) GetResumeSession() *ResumeMsg {
	return m.resumeSession
}

// View implements tea.Model
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Calculate panel widths
	statusHeight := 2
	panelHeight := m.height - statusHeight - 1

	projectWidth := m.width / 4
	sessionWidth := m.width / 4
	previewWidth := m.width - projectWidth - sessionWidth

	// Minimum widths
	if projectWidth < 20 {
		projectWidth = 20
	}
	if sessionWidth < 25 {
		sessionWidth = 25
	}

	// Render panels
	projectPanel := renderProjectsPanel(m.projects, m.projectIdx, m.activePanel == PanelProjects, projectWidth, panelHeight)
	sessionPanel := renderSessionsPanel(m.activeSessions(), m.sessionIdx, m.activePanel == PanelSessions, sessionWidth, panelHeight, m.searchActive, m.searchQuery)
	// Get branch and msg count for preview title bar
	previewBranch := ""
	previewMsgCount := 0
	if session := m.selectedSession(); session != nil {
		previewBranch = session.GitBranch
	}
	if m.preview != nil {
		previewMsgCount = m.preview.TotalMessages
	}
	previewPanel, previewLines := renderPreviewPanel(m.previewLines, m.activePanel == PanelPreview, previewWidth, panelHeight, m.previewScroll, previewBranch, previewMsgCount)
	m.previewLineCount = previewLines

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, projectPanel, sessionPanel, previewPanel)

	// Status bar
	var statusLine string
	if m.searchInProgress {
		spinnerChars := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		spinner := string(spinnerChars[m.searchSpinner%len(spinnerChars)])
		statusLine = statusBarStyle.Width(m.width).Render(
			searchStyle.Render(spinner+" Searching..."),
		)
	} else if m.searching {
		statusLine = statusBarStyle.Width(m.width).Render(
			searchStyle.Render("/ ") + m.searchQuery + "█",
		)
	} else if m.statusMsg != "" {
		statusLine = statusBarStyle.Width(m.width).Render(m.statusMsg)
	} else {
		statusLine = renderStatusBar(m.width, m.searchActive)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, panels, statusLine)

	// Overlay dialog if active
	if m.dialog != nil && m.dialog.Type != DialogNone {
		dialog := renderDialog(m.dialog, m.width, m.height)
		if dialog != "" {
			// Simple overlay: just replace the view
			return dialog
		}
	}

	return view
}

func renderStatusBar(width int, searchActive bool) string {
	keys := []struct {
		key  string
		desc string
	}{
		{"↑↓", "navigate"},
		{"←→", "panels"},
		{"enter", "resume"},
		{"r", "rename"},
		{"d", "delete"},
		{"x", "export"},
		{"space", "select"},
		{"P", "prune"},
		{"/", "search"},
		{"?", "help"},
		{"q", "quit"},
	}

	if searchActive {
		keys = append([]struct {
			key  string
			desc string
		}{{"esc", "clear search"}}, keys...)
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpDescStyle.Render(k.desc))
	}

	line := strings.Join(parts, "  ")
	return statusBarStyle.Width(width).Render(line)
}

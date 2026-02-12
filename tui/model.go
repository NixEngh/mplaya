package tui

import (
	"fmt"
	"time"

	"mplaya/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	playingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	pausedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type Model struct {
	players  []player.Player
	cursor   int
	err      error
	quitting bool
}

type tickMsg time.Time
type refreshMsg []player.Player

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshPlayers, tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshPlayers() tea.Msg {
	players, _ := player.ListPlayers()
	return refreshMsg(players)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(refreshPlayers, tickCmd())

	case refreshMsg:
		m.players = msg
		// Sort pinned player to top
		pinned := player.GetPinnedPlayer()
		if pinned != "" {
			for i, p := range m.players {
				if p.Name == pinned && i > 0 {
					// Move pinned player to top
					m.players = append([]player.Player{p}, append(m.players[:i], m.players[i+1:]...)...)
					break
				}
			}
		}
		// Keep cursor in bounds
		if m.cursor >= len(m.players) {
			m.cursor = max(0, len(m.players)-1)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "j", "down":
			if m.cursor < len(m.players)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case " ":
			if len(m.players) > 0 {
				player.PlayPause(m.players[m.cursor].Name)
				return m, refreshPlayers
			}

		case "p":
			if len(m.players) > 0 {
				player.Pin(m.players[m.cursor].Name)
				return m, refreshPlayers
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	s := titleStyle.Render("mplaya - MPRIS Manager") + "\n\n"

	if len(m.players) == 0 {
		s += stoppedStyle.Render("  No players found") + "\n"
	}

	pinnedPlayer := player.GetPinnedPlayer()

	for i, p := range m.players {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		// Pin indicator
		pinIndicator := " "
		if p.Name == pinnedPlayer {
			pinIndicator = "📌"
		}

		// Status icon
		var icon string
		var iconStyle lipgloss.Style
		switch p.Status {
		case player.StatusPlaying:
			icon = "▶"
			iconStyle = playingStyle
		case player.StatusPaused:
			icon = "⏸"
			iconStyle = pausedStyle
		default:
			icon = "⏹"
			iconStyle = stoppedStyle
		}

		// Format metadata
		var info string
		if p.Artist != "" && p.Title != "" {
			info = fmt.Sprintf("%s - %s", p.Artist, p.Title)
		} else if p.Title != "" {
			info = p.Title
		} else if p.Status == player.StatusStopped {
			info = "(stopped)"
		}

		// Player name (truncate if needed)
		name := p.Name
		if len(name) > 16 {
			name = name[:16]
		}

		line := fmt.Sprintf("%s%s%s %-16s %s",
			cursor,
			pinIndicator,
			iconStyle.Render(icon),
			name,
			info,
		)

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s += line + "\n"
	}

	s += "\n" + helpStyle.Render("[j/k] navigate  [space] play/pause  [p] pin  [q] quit")

	return s
}

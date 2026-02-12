package tui

import (
	"fmt"
	"time"

	"mplaya/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	playingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	pausedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	// Outer frame (size set dynamically)
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	// Player card (width set dynamically)
	cardBaseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Height(5).
			Padding(0, 2)

	// Selected card
	selectedCardBaseStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("212")).
				Height(5).
				Padding(0, 2)

	// Button styles (compact)
	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(3).
			Align(lipgloss.Center)

	// Active play button (when playing)
	playingButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("82")).
				Foreground(lipgloss.Color("82")).
				Width(3).
				Align(lipgloss.Center)

	// Paused button style
	pausedButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("214")).
				Foreground(lipgloss.Color("214")).
				Width(3).
				Align(lipgloss.Center)
)

type Model struct {
	players  []player.Player
	cursor   int
	err      error
	quitting bool
	width    int
	height   int
}

type tickMsg time.Time
type refreshMsg []player.Player

func NewModel() Model {
	zone.NewGlobal()
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		// Check button clicks for each player
		for i := range m.players {
			if zone.Get(fmt.Sprintf("prev_%d", i)).InBounds(msg) {
				player.Previous(m.players[i].Name)
				return m, refreshPlayers
			}
			if zone.Get(fmt.Sprintf("play_%d", i)).InBounds(msg) {
				player.PlayPause(m.players[i].Name)
				return m, refreshPlayers
			}
			if zone.Get(fmt.Sprintf("next_%d", i)).InBounds(msg) {
				player.Next(m.players[i].Name)
				return m, refreshPlayers
			}
			if zone.Get(fmt.Sprintf("card_%d", i)).InBounds(msg) {
				m.cursor = i
				return m, nil
			}
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

		case "h", "left":
			if len(m.players) > 0 {
				player.Previous(m.players[m.cursor].Name)
				return m, refreshPlayers
			}

		case "l", "right":
			if len(m.players) > 0 {
				player.Next(m.players[m.cursor].Name)
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

	// Calculate frame size based on terminal (with max)
	frameWidth := m.width - 4
	if frameWidth > 120 {
		frameWidth = 120
	}
	if frameWidth < 60 {
		frameWidth = 60
	}

	frameHeight := m.height - 2
	if frameHeight > 40 {
		frameHeight = 40
	}
	if frameHeight < 15 {
		frameHeight = 15
	}

	// Card width based on frame (accounting for frame border/padding)
	cardWidth := frameWidth - 10
	if cardWidth < 50 {
		cardWidth = 50
	}

	title := titleStyle.Render("mplaya - MPRIS Manager")

	pinnedPlayer := player.GetPinnedPlayer()

	// Build player cards
	var cards []string
	if len(m.players) == 0 {
		emptyCard := cardBaseStyle.Width(cardWidth).Render(stoppedStyle.Render("No players found"))
		cards = append(cards, emptyCard)
	} else {
		for i, p := range m.players {
			card := m.renderCard(i, p, pinnedPlayer, cardWidth)
			cards = append(cards, zone.Mark(fmt.Sprintf("card_%d", i), card))
		}
	}

	// Stack cards vertically
	cardStack := lipgloss.JoinVertical(lipgloss.Left, cards...)

	// Help text
	help := helpStyle.Render("[j/k] select  [h/l] prev/next  [space] play/pause  [p] pin  [q] quit")

	// Combine everything
	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		cardStack,
		help,
	)

	// Wrap in frame with dynamic size
	framed := frameStyle.
		Width(frameWidth).
		Height(frameHeight).
		Render(inner)

	// Center on screen
	return zone.Scan(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, framed))
}

func (m Model) renderCard(index int, p player.Player, pinnedPlayer string, cardWidth int) string {
	// Pin indicator
	pinIndicator := "  "
	if p.Name == pinnedPlayer {
		pinIndicator = "📌"
	}

	// Player name (truncate if needed)
	name := p.Name
	maxNameLen := 30
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
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

	// Truncate info based on available width
	maxInfoLen := cardWidth - 30
	if maxInfoLen < 20 {
		maxInfoLen = 20
	}
	if len(info) > maxInfoLen {
		info = info[:maxInfoLen-3] + "..."
	}

	// Build buttons with state-aware play/pause button
	prevBtn := zone.Mark(fmt.Sprintf("prev_%d", index), buttonStyle.Render("⏮"))

	var playBtn string
	switch p.Status {
	case player.StatusPlaying:
		playBtn = zone.Mark(fmt.Sprintf("play_%d", index), playingButtonStyle.Render("⏸"))
	case player.StatusPaused:
		playBtn = zone.Mark(fmt.Sprintf("play_%d", index), pausedButtonStyle.Render("▶"))
	default:
		playBtn = zone.Mark(fmt.Sprintf("play_%d", index), buttonStyle.Render("▶"))
	}

	nextBtn := zone.Mark(fmt.Sprintf("next_%d", index), buttonStyle.Render("⏭"))

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, prevBtn, " ", playBtn, " ", nextBtn)

	// First line: pin + name
	line1 := fmt.Sprintf("%s %s", pinIndicator, name)

	// Second line: info
	line2 := stoppedStyle.Render(info)
	if p.Status == player.StatusPlaying {
		line2 = playingStyle.Render(info)
	} else if p.Status == player.StatusPaused {
		line2 = pausedStyle.Render(info)
	}

	// Left side content
	leftContent := lipgloss.JoinVertical(lipgloss.Left, line1, line2)

	// Calculate left width to push buttons to the right
	leftWidth := cardWidth - 22
	if leftWidth < 30 {
		leftWidth = 30
	}
	leftSide := lipgloss.NewStyle().Width(leftWidth).Render(leftContent)

	// Combine left content and buttons
	cardContent := lipgloss.JoinHorizontal(lipgloss.Center, leftSide, buttons)

	// Use selected style if this is the current card
	if index == m.cursor {
		return selectedCardBaseStyle.Width(cardWidth).Render(cardContent)
	}
	return cardBaseStyle.Width(cardWidth).Render(cardContent)
}

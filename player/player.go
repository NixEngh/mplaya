package player

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Status string

const (
	StatusPlaying Status = "Playing"
	StatusPaused  Status = "Paused"
	StatusStopped Status = "Stopped"
)

type Player struct {
	Name   string
	Status Status
	Artist string
	Title  string
}

// ListPlayers returns all available MPRIS players
func ListPlayers() ([]Player, error) {
	cmd := exec.Command("playerctl", "--list-all")
	output, err := cmd.Output()
	if err != nil {
		// No players available
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []Player{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Player{}, nil
	}

	players := make([]Player, 0, len(lines))
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		p := Player{Name: name}
		p.Status = GetStatus(name)
		artist, title := GetMetadata(name)
		p.Artist = artist
		p.Title = title
		players = append(players, p)
	}

	return players, nil
}

// GetStatus returns the playback status of a player
func GetStatus(playerName string) Status {
	cmd := exec.Command("playerctl", "-p", playerName, "status")
	output, err := cmd.Output()
	if err != nil {
		return StatusStopped
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "Playing":
		return StatusPlaying
	case "Paused":
		return StatusPaused
	default:
		return StatusStopped
	}
}

// GetMetadata returns artist and title for a player
func GetMetadata(playerName string) (artist, title string) {
	// Get artist
	artistCmd := exec.Command("playerctl", "-p", playerName, "metadata", "artist")
	artistOutput, _ := artistCmd.Output()
	artist = strings.TrimSpace(string(artistOutput))

	// Get title
	titleCmd := exec.Command("playerctl", "-p", playerName, "metadata", "title")
	titleOutput, _ := titleCmd.Output()
	title = strings.TrimSpace(string(titleOutput))

	return artist, title
}

// PlayPause toggles playback for a player
func PlayPause(playerName string) error {
	cmd := exec.Command("playerctl", "-p", playerName, "play-pause")
	return cmd.Run()
}

// Play starts playback (also implicitly sets priority)
func Play(playerName string) error {
	cmd := exec.Command("playerctl", "-p", playerName, "play")
	return cmd.Run()
}

// PauseAll pauses all players
func PauseAll() error {
	cmd := exec.Command("playerctl", "-a", "pause")
	return cmd.Run()
}

// SmartPlayPause implements smart play/pause behavior:
// - If ANY player is playing → pause ALL players, return "pause"
// - If NO player is playing → play pinned (or first available), return "play"
func SmartPlayPause() (action string, err error) {
	players, err := ListPlayers()
	if err != nil {
		return "", err
	}

	// Check if any player is currently playing
	anyPlaying := false
	for _, p := range players {
		if p.Status == StatusPlaying {
			anyPlaying = true
			break
		}
	}

	if anyPlaying {
		// Pause all players
		err = PauseAll()
		return "pause", err
	}

	// No player is playing, start the pinned player (or first available)
	targetPlayer := GetPinnedPlayer()
	if targetPlayer == "" && len(players) > 0 {
		targetPlayer = players[0].Name
	}

	if targetPlayer == "" {
		return "", nil // No players available
	}

	err = Play(targetPlayer)
	return "play", err
}


// Pin toggles the player as the pinned player for hardware media key control.
// If the player is already pinned, it unpins it.
func Pin(playerName string) error {
	if GetPinnedPlayer() == playerName {
		return clearPriorityFile()
	}
	return writePriorityFile(playerName)
}

// clearPriorityFile removes the priority file to unpin any player
func clearPriorityFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	priorityFile := filepath.Join(home, ".config", "mplaya", "priority")
	err = os.Remove(priorityFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// GetPinnedPlayer returns the currently pinned player name, or empty string if none
func GetPinnedPlayer() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	priorityFile := filepath.Join(home, ".config", "mplaya", "priority")
	data, err := os.ReadFile(priorityFile)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// writePriorityFile writes the player name to ~/.config/mplaya/priority
func writePriorityFile(playerName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".config", "mplaya")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	priorityFile := filepath.Join(dir, "priority")
	return os.WriteFile(priorityFile, []byte(playerName), 0644)
}

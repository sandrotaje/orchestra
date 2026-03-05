package tui

import "github.com/charmbracelet/bubbletea"

type keyAction int

const (
	keyNone keyAction = iota
	keyUp
	keyDown
	keyEnter
	keyNew
	keyOpenPR
	keyRefresh
	keyRemove
	keyQuit
	keyEscape
	keyTop
	keyBottom
	keyNextGroup
	keyPrevGroup
	keyCopy
	keyFilter
	keyUpdateBranch
)

func parseKey(msg tea.KeyMsg) keyAction {
	switch msg.String() {
	case "k", "up":
		return keyUp
	case "j", "down":
		return keyDown
	case "enter", "l":
		return keyEnter
	case "n":
		return keyNew
	case "o":
		return keyOpenPR
	case "r":
		return keyRefresh
	case "x":
		return keyRemove
	case "q", "ctrl+c":
		return keyQuit
	case "esc":
		return keyEscape
	case "g":
		return keyTop
	case "G":
		return keyBottom
	case "tab":
		return keyNextGroup
	case "shift+tab":
		return keyPrevGroup
	case "y":
		return keyCopy
	case "/":
		return keyFilter
	case "u":
		return keyUpdateBranch
	default:
		return keyNone
	}
}

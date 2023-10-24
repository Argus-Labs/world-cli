package component

import "github.com/charmbracelet/lipgloss"

type IBox interface {
	GetHeight() int
	GetWidth() int
	SetStyle(style *lipgloss.Style)
}

type Option = func(box IBox)

func WithBorder(box IBox) {
	style := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	box.SetStyle(&style)
}

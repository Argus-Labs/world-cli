package component

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func CreateBoxInfo(box BoxAndTeaModel, widthPercentage int, heightPercentage int, boxOptions ...Option) *BoxInfo {
	for _, option := range boxOptions {
		option(box)
	}
	res := BoxInfo{
		box:              box,
		widthPercentage:  widthPercentage,
		heightPercentage: heightPercentage,
	}
	return &res
}

func CreateWindowBoxInfo(widthPercentage int, heightPercentage int, totalWidth int, totalHeight int, content string, boxOptions ...Option) *BoxInfo {
	box := CreateWindowBox(
		int(float64(widthPercentage*totalWidth)/100.0),
		int(float64(heightPercentage*totalHeight)/100.0),
		content)
	return CreateBoxInfo(box, widthPercentage, heightPercentage, boxOptions...)
}

func CreateWindowBox(width int, height int, content string, options ...Option) *WindowBox {
	res := WindowBox{
		Style: lipgloss.
			NewStyle().
			Align(lipgloss.Top, lipgloss.Left),
		Width:   width,
		Height:  height,
		Content: content,
	}

	for _, option := range options {
		option(&res)
	}
	return &res
}

type WindowBox struct {
	Style   lipgloss.Style
	Width   int
	Height  int
	Content string
}

func (b *WindowBox) View() string {
	return b.Style.Width(b.Width).Height(b.Height).Render(b.Content)
}

func (b *WindowBox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.Width = msg.Width
		b.Height = msg.Height
	}
	return b, nil
}

func (b *WindowBox) SetStyle(style *lipgloss.Style) {
	b.Style = b.Style.Inherit(*style)
}

func (b *WindowBox) Init() tea.Cmd {
	return nil
}

func (b *WindowBox) GetWidth() int {
	return b.Width
}

func (b *WindowBox) GetHeight() int {
	return b.Height
}

type BoxAndTeaModel interface {
	IBox
	tea.Model
}

type BoxInfo struct {
	box              BoxAndTeaModel
	widthPercentage  int
	heightPercentage int
}

type LayoutMode int

const (
	rowMajor LayoutMode = iota
	colMajor
)

type BoxLayout struct {
	grid        [][]*BoxInfo
	TotalWidth  int
	TotalHeight int
	Mode        LayoutMode
}

func (b *BoxLayout) IsGridValid() bool {
	maxHeights := make([]int, 0, len(b.grid))
	for _, row := range b.grid {
		acc := 0
		currentMaxHeight := -1
		for _, boxInfo := range row {
			if boxInfo.widthPercentage > currentMaxHeight {
				currentMaxHeight = boxInfo.widthPercentage
			}
			acc += boxInfo.widthPercentage
			if acc != 100 {
				return false
			}
		}
		maxHeights = append(maxHeights, currentMaxHeight)
	}

	acc := 0
	for _, height := range maxHeights {
		acc += height
		if acc != 100 {
			return false
		}
	}
	return true
}

func (b *BoxLayout) GetWidth() int {
	return b.TotalWidth
}

func (b *BoxLayout) GetHeight() int {
	return b.TotalHeight
}

func (b *BoxLayout) SetStyle(style *lipgloss.Style) {
	for _, row := range b.grid {
		for _, boxInfo := range row {
			boxInfo.box.SetStyle(style)
		}
	}
}

func (b *BoxLayout) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateAllChildren := func(msg tea.Msg) (*BoxLayout, tea.Cmd) {
		var cmds []tea.Cmd
		for i, row := range b.grid {
			for j, boxInfo := range row {
				var m tea.Model
				var cmd tea.Cmd
				m, cmd = boxInfo.box.Update(msg)
				b.grid[i][j].box = m.(BoxAndTeaModel)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				boxInfo.box = m.(BoxAndTeaModel)
			}
		}
		return b, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.TotalWidth = msg.Width
		b.TotalHeight = msg.Height
		//updateAllChildren(msg)
		borderOffsetPerWindow := 2
		var cmds []tea.Cmd
		for i, row := range b.grid {
			for j, boxInfo := range row {
				newUpdateMessage := tea.WindowSizeMsg{
					Width:  int(float64(boxInfo.widthPercentage*b.TotalWidth)/100.0) - borderOffsetPerWindow,
					Height: int(float64(boxInfo.heightPercentage*b.TotalHeight)/100.0) - borderOffsetPerWindow,
				}
				var m tea.Model
				var cmd tea.Cmd
				m, cmd = boxInfo.box.Update(newUpdateMessage)
				cmds = append(cmds, cmd)
				b.grid[i][j].box = m.(BoxAndTeaModel)
			}
		}
		return b, tea.Batch(cmds...)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			newB, cmd := updateAllChildren(msg)
			return newB, tea.Batch(cmd, tea.Quit)
		}

	default:
		newB, cmd := updateAllChildren(msg)
		return newB, cmd
	}
	return b, nil
}

func (b *BoxLayout) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, row := range b.grid {
		for _, boxInfo := range row {
			cmds = append(cmds, boxInfo.box.Init())
		}
	}
	cmds = append(cmds, tea.ClearScreen)
	return tea.Batch(cmds...)
}

func (b *BoxLayout) View() string {
	rowAcc := make([]string, 0, len(b.grid))
	for _, row := range b.grid {
		colAcc := make([]string, 0, len(row))
		for _, boxInfo := range row {
			box := boxInfo.box
			colAcc = append(colAcc, box.View())
		}
		if b.Mode == rowMajor {
			rowAcc = append(rowAcc, lipgloss.JoinHorizontal(lipgloss.Top, colAcc...))
		} else if b.Mode == colMajor {
			rowAcc = append(rowAcc, lipgloss.JoinVertical(lipgloss.Left, colAcc...))
		} else {
			panic("logic error, enum should not arrive here")
		}
	}
	if b.Mode == rowMajor {
		return lipgloss.JoinVertical(lipgloss.Left, rowAcc...)
	} else if b.Mode == colMajor {
		return lipgloss.JoinHorizontal(lipgloss.Top, rowAcc...)
	} else {
		panic("logic error, enum should not arrive here")
	}
}

func BuildTriLayoutHorizontal(totalWidth int, totalHeight int, mainBox *BoxInfo, lowerLeftBox *BoxInfo, lowerRightBox *BoxInfo) *BoxLayout {

	if mainBox == nil {
		mainBox = CreateWindowBoxInfo(100, 70, totalWidth, totalHeight, "", WithBorder)
	}
	if lowerLeftBox == nil {
		lowerLeftBox = CreateWindowBoxInfo(50, 30, totalWidth, totalHeight, "", WithBorder)
	}
	if lowerRightBox == nil {
		lowerRightBox = CreateWindowBoxInfo(50, 30, totalWidth, totalHeight, "", WithBorder)
	}
	res := BoxLayout{
		grid: [][]*BoxInfo{
			{
				mainBox,
			},
			{
				lowerLeftBox,
				lowerRightBox,
			},
		},
		Mode: rowMajor,
	}
	return &res
}

func BuildTriLayoutVertical(totalWidth int, totalHeight int) *BoxLayout {
	res := BoxLayout{
		grid: [][]*BoxInfo{
			{
				CreateWindowBoxInfo(20, 50, totalWidth, totalHeight, "", WithBorder),
				CreateWindowBoxInfo(20, 50, totalWidth, totalHeight, "", WithBorder),
			},
			{
				CreateWindowBoxInfo(80, 100, totalWidth, totalHeight, "", WithBorder),
			},
		},
		Mode: colMajor,
	}
	return &res
}

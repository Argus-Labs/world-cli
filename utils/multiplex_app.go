package utils

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Option = func(box *WindowBox)

func WithBorder(box *WindowBox) {
	box.Style = box.Style.BorderStyle(lipgloss.NormalBorder())
}

func CreateWindowBox(widthPercentage int, heightPercentage int, options ...Option) WindowBox {

	res := WindowBox{
		Style: lipgloss.
			NewStyle().
			Align(lipgloss.Top, lipgloss.Left),
		WidthPercentage:  widthPercentage,
		HeightPercentage: heightPercentage,
	}

	for _, option := range options {
		option(&res)
	}
	return res
}

type WindowBox struct {
	Style            lipgloss.Style
	WidthPercentage  int
	HeightPercentage int
	Content          string
}

func (b WindowBox) Render(totalWidth int, totalHeight int, offsetWidth int, offsetHeight int) string {
	actualWidth := int(float64(b.WidthPercentage)/100.0*float64(totalWidth)) + offsetWidth
	actualHeight := int(float64(b.HeightPercentage)/100.0*float64(totalHeight)) + offsetHeight
	fmt.Printf("width %d, height %d\n", actualWidth, actualHeight)
	return b.Style.Width(actualWidth).Height(actualHeight).Render(b.Content)
}

type BoxLayout struct {
	grid [][]IBox
}

func (b WindowBox) GetWidthPercentage() int {
	return b.WidthPercentage
}

func (b WindowBox) GetHeightPercentage() int {
	return b.HeightPercentage
}

func (b *BoxLayout) isGridValid() bool {
	maxHeights := make([]int, 0, len(b.grid))
	for _, row := range b.grid {
		acc := 0
		currentMaxHeight := -1
		for _, box := range row {
			if box.GetHeightPercentage() > currentMaxHeight {
				currentMaxHeight = box.GetHeightPercentage()
			}
			acc += box.GetWidthPercentage()
			if acc > 100 {
				return false
			}
		}
		maxHeights = append(maxHeights, currentMaxHeight)
	}

	acc := 0
	for _, height := range maxHeights {
		acc += height
		if acc > 100 {
			return false
		}
	}
	return true
}

func (b *BoxLayout) Render(totalWidth int, totalHeight int) string {
	if totalWidth%2 != 0 {
		totalWidth += 1
	}
	if totalHeight%4 != 0 {
		totalHeight += totalHeight % 4
	}

	getWidthOffsets := func(boxLayout *BoxLayout) []int {
		acc := make([]int, 0, len(boxLayout.grid))
		for _, row := range boxLayout.grid {
			acc = append(acc, len(row)-1)
		}
		return acc
	}

	getRowTotalWidthPercentage := func(boxes []IBox) int {
		acc := 0
		for _, box := range boxes {
			acc += box.GetWidthPercentage()
		}
		return acc
	}

	getRemainingOffsets := func(offsets []int) []int {
		maxOffset := -1
		for _, offset := range offsets {
			if offset > maxOffset {
				maxOffset = offset
			}
		}

		acc := make([]int, 0, len(offsets))
		for _, offset := range offsets {
			acc = append(acc, maxOffset-offset)
		}
		return acc
	}

	findIndividualOffsetForEachRow := func(remainingOffset int, rowLength int) []int {
		if remainingOffset == 0 {
			return make([]int, rowLength)
		}
		if remainingOffset >= rowLength {
			offset := remainingOffset / rowLength
			offsets := make([]int, rowLength)
			remainder := remainingOffset % rowLength
			for i, _ := range offsets {
				if remainder > 0 {
					offsets[i] += 1
					remainder -= 1
				}
				offsets[i] += offset
			}
			return offsets
		} else {
			res := make([]int, rowLength)
			for i, _ := range res {
				if remainingOffset > 0 {
					res[i] += 1
					remainingOffset -= 1
				}
			}
			return res
		}
	}

	remainingOffsets := getRemainingOffsets(getWidthOffsets(b))

	rowAcc := make([]string, 0, len(b.grid))
	for index, row := range b.grid {
		colAcc := make([]string, 0, len(row))
		offsets := findIndividualOffsetForEachRow(remainingOffsets[index], len(row))

		shouldApplyOffset := getRowTotalWidthPercentage(row) == 100
		for index1, box := range row {
			rowOffset := 0
			if shouldApplyOffset {
				rowOffset = offsets[index1] * 2
			}
			colAcc = append(colAcc, box.Render(totalWidth, totalHeight, rowOffset, 0))
		}
		rowAcc = append(rowAcc, lipgloss.JoinHorizontal(lipgloss.Top, colAcc...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rowAcc...)
}

func BuildTriLayout() BoxLayout {
	return BoxLayout{
		grid: [][]IBox{
			{
				CreateWindowBox(100, 80, WithBorder),
			},
			{
				CreateWindowBox(50, 20, WithBorder),
				CreateWindowBox(50, 20, WithBorder),
			},
		},
	}
}

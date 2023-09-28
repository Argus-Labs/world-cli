package utils

type IBox interface {
	Render(int, int, int, int) string
	GetHeightPercentage() int
	GetWidthPercentage() int
}

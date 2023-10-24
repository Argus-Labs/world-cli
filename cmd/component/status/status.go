package status

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/guumaster/logsymbols"
	"sync/atomic"
)

type Status int32

const (
	PENDING Status = iota
	SUCCESS
	FAILED
)

//////////////////////
// Bubble Tea Model //
//////////////////////

type Model struct {
	statusName string
	status     atomic.Int32
	check      func(*Model)
}

func New(statusName string, checkFunc func(*Model)) Model {
	res := Model{
		statusName: statusName,
		check:      checkFunc,
	}
	res.status.Store(int32(PENDING))
	return res
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

func (s *Model) AutoSetStatus() {
	if s.GetStatus() == PENDING {
		s.check(s)
	}
}

func (s *Model) SetStatus(status Status) {
	s.status.Store(int32(status))
}

func (s *Model) GetStatus() Status {
	return Status(s.status.Load())
}

func (s *Model) GetStatusMessage(spinnerModel *spinner.Model) string {
	var prefix string
	switch s.GetStatus() {
	case PENDING:
		prefix = spinnerModel.View()
		break
	case SUCCESS:
		prefix = string(logsymbols.Success)
		break
	case FAILED:
		prefix = string(logsymbols.Error)
		break
	default:
		panic("logic error with GetStatusMessage, check enum utils.Status")
	}
	finalString := fmt.Sprintf("%s %s\n", prefix, s.statusName)
	return finalString
}

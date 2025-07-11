package project

import (
	"context"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/internal/pkg/printer"
	"pkg.world.dev/world-cli/internal/pkg/tea/component/multiselect"
	"pkg.world.dev/world-cli/internal/pkg/tea/component/program"
)

// TODO: This is a temporary implementation of the region selector.
// We need to make the bubbletea component reuable and mockable.
// This way we can test the region selector without having to run the TUI.

// BubbleteeRegionSelector implements RegionSelector using bubbletea TUI.
type BubbleteeRegionSelector struct{}

func (b *BubbleteeRegionSelector) SelectRegions(
	ctx context.Context,
	regions []string,
	selectedRegions []string,
) ([]string, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			aborted, selectedRegion, err := b.runRegionSelector(ctx, regions, selectedRegions)
			if err != nil {
				printer.Errorln(err.Error())
				printer.NewLine(1)
				if aborted {
					return nil, err
				}
				continue
			}
			if len(selectedRegion) > 0 {
				return selectedRegion, nil
			}
			printer.NewLine(1)
			printer.Errorln("At least one region must be selected")
			printer.Infoln("🔄 Please try again")
		}
	}
}

func (b *BubbleteeRegionSelector) runRegionSelector(
	ctx context.Context,
	regions []string,
	selectedRegions []string,
) (bool, []string, error) {
	var regionSelector *tea.Program
	if len(selectedRegions) > 0 {
		selectedRegionsMap := make(map[int]bool)
		for i, region := range regions {
			if slices.Contains(selectedRegions, region) {
				selectedRegionsMap[i] = true
			}
		}
		regionSelector = program.NewTeaProgram(multiselect.UpdateMultiselectModel(ctx, regions, selectedRegionsMap))
	} else {
		regionSelector = program.NewTeaProgram(multiselect.InitialMultiselectModel(ctx, regions))
	}

	m, err := regionSelector.Run()
	if err != nil {
		return false, nil, eris.Wrap(err, "failed to run region selector")
	}

	model, ok := m.(multiselect.Model)
	if !ok {
		return false, nil, eris.New("failed to get selected regions")
	}
	if model.Aborted {
		return true, nil, eris.New("Region selection aborted")
	}

	var result []string
	for i, item := range regions {
		if model.Selected[i] {
			result = append(result, item)
		}
	}

	return false, result, nil
}

// MockRegionSelector implements RegionSelector for testing.
type MockRegionSelector struct {
	selectedRegions []string
	err             error
}

func NewMockRegionSelector(selectedRegions []string, err error) *MockRegionSelector {
	return &MockRegionSelector{
		selectedRegions: selectedRegions,
		err:             err,
	}
}

func (m *MockRegionSelector) SelectRegions(
	_ context.Context,
	_ []string,
	_ []string,
) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.selectedRegions, nil
}

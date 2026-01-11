// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

const SearchModalPage = "Search"

type Search struct {
	Input *tview.InputField
	Flex  *tview.Flex
}

func NewSearchModal(colors *config.ColorConfig) *Search {
	input := tview.NewInputField()
	input.SetFieldBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	input.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	input.SetBorder(true)
	input.SetBorderColor(tcell.GetColor(colors.Cinnamon.Border))
	input.SetTitle(" Search ")
	input.SetTitleAlign(tview.AlignLeft)
	input.SetBorderPadding(0, 0, 1, 0)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 3, false).
				AddItem(input, 0, 4, true).
				AddItem(nil, 0, 3, false),
			3, 0, true).
		AddItem(nil, 0, 1, false)

	flex.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))

	return &Search{
		Input: input,
		Flex:  flex,
	}
}

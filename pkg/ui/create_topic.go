// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/util"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CreateTopicParams struct {
	TopicName         string
	ReplicationFactor int32
	Partitions        int32
	Properties        string
}

var topicCreationParams = &CreateTopicParams{
	TopicName:         "",
	ReplicationFactor: 1,
	Partitions:        1,
	Properties:        "",
}

func (app *App) InitCreateTopicModal() {
	width := 40

	// Input field for topic name
	topicName := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)

	// Input field for replication factor
	replicationFactor := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	replicationFactor.SetAcceptanceFunc(tview.InputFieldInteger)
	replicationFactor.SetText("1")

	// Input field for partitions
	partitions := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	partitions.SetAcceptanceFunc(tview.InputFieldInteger)
	partitions.SetText("1")

	// Dropdown for optional properties
	propertiesDropdown := tview.NewDropDown().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)

	// Common topic properties
	propertyOptions := []string{
		"(none)",
		"cleanup.policy=delete",
		"cleanup.policy=compact",
		"compression.type=gzip",
		"compression.type=snappy",
		"compression.type=lz4",
		"compression.type=zstd",
		"retention.ms=604800000",     // 7 days
		"retention.ms=86400000",      // 1 day
		"retention.bytes=1073741824", // 1GB
	}
	propertiesDropdown.SetOptions(propertyOptions, nil)

	// Selection table (labels)
	selection := tview.NewTable()
	selection.SetCell(
		0,
		0,
		tview.NewTableCell("Topic Name:").SetAlign(tview.AlignRight),
	)
	selection.SetCell(
		1,
		0,
		tview.NewTableCell("Replication Factor:").SetAlign(tview.AlignRight),
	)
	selection.SetCell(
		2,
		0,
		tview.NewTableCell("Partitions:").SetAlign(tview.AlignRight),
	)
	selection.SetCell(
		3,
		0,
		tview.NewTableCell("Properties (optional):").SetAlign(tview.AlignRight),
	)
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)

	// Keep order of input fields
	inputFields := []*tview.InputField{topicName, replicationFactor, partitions}
	for _, inf := range inputFields {
		inf.SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(selection)
		})
	}

	// Create layout
	f := tview.NewFlex()
	f.SetDirection(tview.FlexColumn)
	f.AddItem(selection, 30, 0, true)
	f.AddItem(tview.NewBox(), 3, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topicName, 1, 0, false).
		AddItem(replicationFactor, 1, 0, false).
		AddItem(partitions, 1, 0, false).
		AddItem(propertiesDropdown, 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	f.AddItem(inputs, 40, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	// Input capture for topic name
	topicName.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			topicCreationParams.TopicName = topicName.GetText()
		}
		return event
	})

	// Input capture for replication factor
	replicationFactor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			topicCreationParams.ReplicationFactor = util.GetInt32(replicationFactor)
		}
		return event
	})

	// Input capture for partitions
	partitions.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			topicCreationParams.Partitions = util.GetInt32(partitions)
		}
		return event
	})

	// Input capture for properties dropdown
	propertiesDropdown.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			_, selected := propertiesDropdown.GetCurrentOption()
			if selected != "(none)" {
				topicCreationParams.Properties = selected
			} else {
				topicCreationParams.Properties = ""
			}
			app.SetFocus(selection)
		}
	})

	// Selection table input capture
	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := selection.GetSelection()

		if event.Key() == tcell.KeyEnter {
			if row < len(inputFields) {
				app.SetFocus(inputFields[row])
			} else if row == 3 {
				app.SetFocus(propertiesDropdown)
			}
		}

		// Reset to defaults on 'c' key
		if event.Key() == tcell.KeyRune && event.Rune() == 'c' {
			topicName.SetText("")
			replicationFactor.SetText("1")
			partitions.SetText("1")
			propertiesDropdown.SetCurrentOption(0)
		}

		// Submit on 's' key
		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			// TODO: Implement actual topic creation logic here
			// For now, just close the modal
			statusLineCh <- "Topic creation would be triggered here"
			app.HideModalPage(CreateTopic)
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(CreateTopic)
		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f, 0, 1, true)
	flex.SetTitle(" Create Topic ")
	flex.SetBorder(true)

	modal := util.NewModal(flex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(CreateTopic, modal, true, false)
}

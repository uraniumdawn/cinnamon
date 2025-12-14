// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/util"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type CreateTopicParams struct {
	TopicName         string
	ReplicationFactor int
	Partitions        int
	Config            map[string]string
}

var params = &CreateTopicParams{
	TopicName:         "",
	ReplicationFactor: 1,
	Partitions:        1,
	Config:            make(map[string]string),
}

func (app *App) InitCreateTopicModal() {
	width := 40

	topicName := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)

	replicationFactor := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	replicationFactor.SetAcceptanceFunc(tview.InputFieldInteger)
	replicationFactor.SetText("1")

	partitions := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	partitions.SetAcceptanceFunc(tview.InputFieldInteger)
	partitions.SetText("1")

	// Text area for optional properties (multi-line)
	configTextArea := (tview.NewTextArea().
		SetPlaceholder(`Enter properties (one per line):
cleanup.policy=delete
retention.ms=604800000`))

	selection := tview.NewTable()
	selection.SetCell(0, 0, tview.NewTableCell("Name:").SetAlign(tview.AlignRight))
	selection.SetCell(1, 0, tview.NewTableCell("Replication factor:").SetAlign(tview.AlignRight))
	selection.SetCell(2, 0, tview.NewTableCell("Partitions:").SetAlign(tview.AlignRight))
	selection.SetCell(3, 0, tview.NewTableCell("Properties (optional):").SetAlign(tview.AlignLeft))
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)

	inputFields := []*tview.InputField{topicName, replicationFactor, partitions}
	for _, inf := range inputFields {
		inf.SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(selection)
		})
	}

	f := tview.NewFlex()
	f.SetDirection(tview.FlexColumn)
	f.AddItem(selection, 30, 0, true)
	f.AddItem(tview.NewBox(), 3, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topicName, 1, 0, false).
		AddItem(replicationFactor, 1, 0, false).
		AddItem(partitions, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(configTextArea, 0, 1, false)

	f.AddItem(inputs, 40, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	topicName.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			params.TopicName = topicName.GetText()
		}
		return event
	})

	replicationFactor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			params.ReplicationFactor, _ = strconv.Atoi(replicationFactor.GetText())
		}
		return event
	})

	partitions.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			params.Partitions, _ = strconv.Atoi(partitions.GetText())
		}
		return event
	})

	configTextArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			propertiesText := configTextArea.GetText()
			params.Config = parseConfig(propertiesText)
			app.SetFocus(selection)
			return nil
		}
		return event
	})

	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := selection.GetSelection()

		if event.Key() == tcell.KeyEnter {
			if row < len(inputFields) {
				app.SetFocus(inputFields[row])
			} else if row == 3 {
				app.SetFocus(configTextArea)
			}
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'c' {
			topicName.SetText("")
			replicationFactor.SetText("1")
			partitions.SetText("1")
			configTextArea.SetText("", true)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			app.CreationTopicHandler(
				params.TopicName,
				params.ReplicationFactor,
				params.Partitions,
				params.Config,
			)
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

func parseConfig(text string) map[string]string {
	properties := make(map[string]string)

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				properties[key] = value
			}
		}
	}

	return properties
}

func (app *App) CreationTopicHandler(
	name string,
	numPartitions int,
	replicationFactor int,
	config map[string]string,
) {
	statusLineCh <- "creating topic..."
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.CreateTopic(name, numPartitions, replicationFactor, config, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case <-resultCh:
				statusLineCh <- "topic has been created"
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to create topic")
				statusLineCh <- fmt.Sprintf("[red]failed to create topic: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while creating topics")
				statusLineCh <- "[red]timeout while creating topics"
				return
			}
		}
	}()
}

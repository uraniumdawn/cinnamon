// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/client"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	GetTopicsEventType    EventType = "topics:get"
	GetTopicEventType     EventType = "topic:get"
	CreateTopicEventType  EventType = "topic:create"
	DeleteTopicEventType  EventType = "topic:delete"
	EditTopicEventType    EventType = "topic:edit"
	CliTemplatesEventType EventType = "topic:cli-templates"
)

var TopicsChannel = make(chan Event)

func (app *App) RunTopicsEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Topics Event Handler")
				return
			case event := <-in:
				switch event.Type {
				case GetTopicsEventType:
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, Topics)
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						statusLineCh <- "getting topics..."
						app.Topics()
					}

				case GetTopicEventType:
					topicName := event.Payload.Data.(string)
					force := event.Payload.Force
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, Topics, topicName)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						statusLineCh <- "getting topic description..."
						app.Topic(topicName)
					}

				case CreateTopicEventType:
					app.QueueUpdateDraw(func() {
						app.CreateTopic()
						app.ShowModalPage(CreateTopic)
					})

				case DeleteTopicEventType:
					topicName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.DeleteTopic(topicName)
						app.ShowModalPage(DeleteTopic)
					})

				case EditTopicEventType:
					topicName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.UpdateTopic(topicName)
					})

				case CliTemplatesEventType:
					topicName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.CliTemplates(topicName)
					})
				}
			}
		}
	}()
}

type TopicParams struct {
	TopicName         string
	ReplicationFactor int
	Partitions        int
	Config            map[string]string
}

func (app *App) Topics() {
	resultCh := make(chan *client.TopicsResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.Topics(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case topics := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewTopicsTable(topics)
					title := util.BuildTitle(Topics,
						"["+strconv.Itoa(len(topics.Result))+"]")
					table.SetTitle(title)
					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Topics),
						table,
						TopicsPageMenu, true,
					)

					// app.InitConsumingParams()

					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(TopicsChannel, GetTopicsEventType, Payload{nil, true})
						}
						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							Publish(
								TopicsChannel,
								GetTopicEventType,
								Payload{topicName, false},
							)
						}

						//if event.Key() == tcell.KeyRune && event.Rune() == 'p' {
						//	app.ShowModalPage(ConsumingParams)
						//}

						if event.Key() == tcell.KeyRune && event.Rune() == 'c' {
							app.CreateTopic()
							app.ShowModalPage(CreateTopic)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'x' {
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							app.DeleteTopic(topicName)
							app.ShowModalPage(DeleteTopic)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'e' {
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							app.UpdateTopic(topicName)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 't' {
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							app.CliTemplates(topicName)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'r' {
							statusLineCh <- "consuming records..."
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							rc := make(chan string)
							sig := make(chan int, 1)

							// consumer, _ := NewConsumer(app.Selected.Cluster,
							// app.Selected.SchemaRegistry) go
							// consumer.Consume(ConsumingParameters, topicName, rc,
							// statusLineChannel, sig)

							// args, err := util.ParseShellCommand(
							// 	app.Selected.Cluster.Command,
							// 	topicName,
							// )
							// if err != nil {
							// 	log.Error().Msg("Failed to parse command")
							// 	statusLineChannel <- "[red]Failed to parse command: " +
							// err.Error()
							// 	return event
							// }

							// go shell.Execute(args, rc, statusLineChannel, sig)

							view := tview.NewTextView().
								SetTextAlign(tview.AlignLeft).
								SetDynamicColors(true).
								SetWrap(true).
								SetWordWrap(true).
								SetMaxLines(1000).
								SetScrollable(true)
							view.
								SetBorder(true).
								SetBorderPadding(0, 0, 1, 0).
								SetTitle(topicName)

							view.SetInputCapture(
								func(event *tcell.EventKey) *tcell.EventKey {
									if event.Key() == tcell.KeyRune &&
										event.Rune() == 'e' {
										sig <- 1
										sig <- 1
									}
									return event
								},
							)

							app.AddToPagesRegistry(
								fmt.Sprintf(
									"%s:%s:%s:consume",
									app.Selected.Cluster.Name,
									Topic,
									topicName,
								),
								view,
								ConsumingMenu,
								false,
							)
							run := true
							go func() {
								defer func() {
									statusLineCh <- "consuming terminated"
								}()
								for run {
									select {
									case <-sig:
										run = false
									case record := <-rc:
										_, _ = fmt.Fprintf(
											view,
											"%s\n\n",
											record,
										)
										app.QueueUpdateDraw(func() {
											view.ScrollToEnd()
										})
									}
								}
							}()
						}

						return event
					})

					app.AssignSearch(func(text string) {
						filterTopicsTable(table, topics.Result, text)
						util.SetSearchableTableTitle(table, title, text)
						table.ScrollToBeginning()
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to list topics")
				statusLineCh <- fmt.Sprintf("[red]failed to list topics: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while to list topics")
				statusLineCh <- "[red]timeout while to list topics"
				return
			}
		}
	}()
}

func (app *App) Topic(name string) {
	resultCh := make(chan *client.TopicResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.DescribeTopic(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(util.BuildTitle(Topic, name))
					desc.SetText(description.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(TopicsChannel, GetTopicEventType, Payload{name, true})
						}
						return event
					})
					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Topic, name),
						desc,
						FinalPageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe topic")
				statusLineCh <- fmt.Sprintf("[red]failed to describe topic: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing topic")
				statusLineCh <- "[red]timeout while describing topic"
				return
			}
		}
	}()
}

func (app *App) CreateTopic() {
	params := &TopicParams{
		TopicName:         "",
		ReplicationFactor: -1,
		Partitions:        -1,
		Config:            make(map[string]string),
	}
	width := 40

	topicName := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Put topic name here").
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))

	replicationFactor := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("-1")
	replicationFactor.SetAcceptanceFunc(tview.InputFieldInteger)
	replicationFactor.SetPlaceholderStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Foreground),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Background),
		)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))

	partitions := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("-1")
	partitions.SetAcceptanceFunc(tview.InputFieldInteger)
	partitions.
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))

	// Text area for optional properties (multi-line)
	configTextArea := tview.NewTextArea().
		SetPlaceholder(`Enter properties (one per line):
cleanup.policy=delete
retention.ms=604800000`).
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Placeholder),
			))

	selection := tview.NewTable()
	selection.SetCell(0, 0, tview.NewTableCell("Name:").SetAlign(tview.AlignRight))
	selection.SetCell(1, 0, tview.NewTableCell("Replication factor:").SetAlign(tview.AlignRight))
	selection.SetCell(2, 0, tview.NewTableCell("Partitions:").SetAlign(tview.AlignRight))
	selection.SetCell(3, 0, tview.NewTableCell("Configs (optional):").SetAlign(tview.AlignRight))
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
	f.AddItem(selection, 20, 0, true)
	f.AddItem(tview.NewBox(), 3, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topicName, 1, 0, false).
		AddItem(replicationFactor, 1, 0, false).
		AddItem(partitions, 1, 0, false).
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

		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			params.TopicName = topicName.GetText()
			params.ReplicationFactor, _ = strconv.Atoi(replicationFactor.GetText())
			params.Partitions, _ = strconv.Atoi(partitions.GetText())
			params.Config = parseConfig(configTextArea.GetText())

			if err := params.validate(); err != nil {
				statusLineCh <- fmt.Sprintf("[red]%s", err.Error())
				return event
			}

			app.CreateTopicResultHandler(
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
	app.Layout.PagesRegistry.UI.Pages.AddPage(CreateTopic, modal, true, true)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(CreateTopic)
}

func (app *App) CreateTopicResultHandler(
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
				if err != nil {
					log.Error().Err(err).Msg("Failed to create topic")
					statusLineCh <- fmt.Sprintf("[red]failed to create topic: %s", err.Error())
				} else {
					log.Error().Msg("Failed to create topic: unknown error")
					statusLineCh <- "[red]failed to create topic: unknown error"
				}
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

func (app *App) UpdateTopic(topicName string) {
	statusLineCh <- "fetching topic configuration..."
	resultCh := make(chan *client.TopicResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.DescribeTopic(topicName, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case topicResult := <-resultCh:
				app.QueueUpdateDraw(func() {
					app.NewUpdateTopicModal(topicName, topicResult)
					app.ShowModalPage(EditTopic)
					statusLineCh <- "ready to edit topic"
				})
				cancel()
				return
			case err := <-errorCh:
				if err != nil {
					log.Error().Err(err).Msg("Failed to fetch topic config")
					statusLineCh <- fmt.Sprintf("[red]failed to fetch topic config: %s", err.Error())
				} else {
					log.Error().Msg("Failed to fetch topic config: unknown error")
					statusLineCh <- "[red]failed to fetch topic config: unknown error"
				}
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while fetching topic config")
				statusLineCh <- "[red]timeout while fetching topic config"
				return
			}
		}
	}()
}

func (app *App) UpdateTopicResultHandler(
	name string,
	config map[string]string,
) {
	statusLineCh <- "updating topic config..."
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.UpdateTopicConfig(name, config, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case <-resultCh:
				statusLineCh <- "topic config has been updated"
				cancel()
				return
			case err := <-errorCh:
				if err != nil {
					log.Error().Err(err).Msg("Failed to update topic config")
					statusLineCh <- fmt.Sprintf("[red]failed to update topic config: %s", err.Error())
				} else {
					log.Error().Msg("Failed to update topic config: unknown error")
					statusLineCh <- "[red]failed to update topic config: unknown error"
				}
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while updating topic config")
				statusLineCh <- "[red]timeout while updating topic config"
				return
			}
		}
	}()
}

func (app *App) DeleteTopic(topicName string) {
	messageText := tview.NewTextView().
		SetText(fmt.Sprintf("Topic [red::b]%s[-::-] will be deleted. Confirm?", topicName)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	messageText.SetBorder(true).
		SetTitle(" Confirm Deletion ").
		SetBorderPadding(0, 0, 1, 1)

	messageText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			app.DeleteTopicResultHandler(topicName)
			app.HideModalPage(DeleteTopic)
			Publish(TopicsChannel, GetTopicsEventType, Payload{nil, false})
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(DeleteTopic)
		}

		return event
	})

	modal := util.NewConfirmationModal(messageText)
	app.Layout.PagesRegistry.UI.Pages.AddPage(DeleteTopic, modal, true, true)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(DeleteTopic)
}

func (app *App) DeleteTopicResultHandler(name string) {
	statusLineCh <- "deleting topic..."
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.DeleteTopic(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case <-resultCh:
				statusLineCh <- fmt.Sprintf("topic '%s' has been deleted", name)
				cancel()
				return
			case err := <-errorCh:
				if err != nil {
					log.Error().Err(err).Msg("Failed to delete topic")
					statusLineCh <- fmt.Sprintf("[red]failed to delete topic: %s", err.Error())
				} else {
					log.Error().Msg("Failed to delete topic: unknown error")
					statusLineCh <- "[red]failed to delete topic: unknown error"
				}
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while deleting topic")
				statusLineCh <- "[red]timeout while deleting topic"
				return
			}
		}
	}()
}

func (app *App) NewTopicsTable(topics *client.TopicsResult) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	sorted := treemap.NewWithStringComparator()
	for topicName, metadata := range topics.Result {
		sorted.Put(topicName, metadata)
	}

	row := 0
	partitions := 0
	replicas := 0
	sorted.Each(func(key, value any) {
		t := key.(string)
		m := value.(*kafka.TopicMetadata)
		partitions = len(m.Partitions)
		if len(m.Partitions) > 0 {
			replicas = len(m.Partitions[0].Replicas)
		}

		populateTable(table, row, t, partitions, replicas)
		row++
	})

	return table
}

func (app *App) NewUpdateTopicModal(topicName string, topicResult *client.TopicResult) {
	width := 40

	currentConfig := make(map[string]string)
	partitionCount := 0
	replicationFactor := 0

	if len(topicResult.TopicDescriptions) > 0 {
		desc := topicResult.TopicDescriptions[0]
		partitionCount = len(desc.Partitions)
		if len(desc.Partitions) > 0 {
			replicationFactor = len(desc.Partitions[0].Replicas)
		}
	}

	for _, configResult := range topicResult.Config {
		for _, entry := range configResult.Config {
			// Only include non-default, non-readonly configs
			if !entry.IsDefault && !entry.IsReadOnly {
				currentConfig[entry.Name] = entry.Value
			}
		}
	}

	topicNameField := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetText(topicName)
	topicNameField.SetDisabled(true)

	replicationFactorField := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetText(fmt.Sprintf("%d", replicationFactor))
	replicationFactorField.SetDisabled(true)

	partitionsField := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetText(fmt.Sprintf("%d", partitionCount))
	partitionsField.SetDisabled(true)

	configTextArea := tview.NewTextArea()

	var configLines []string
	for key, value := range currentConfig {
		configLines = append(configLines, fmt.Sprintf("%s=%s", key, value))
	}
	if len(configLines) > 0 {
		configTextArea.SetText(strings.Join(configLines, "\n"), false)
	}

	selection := tview.NewTable()
	selection.SetCell(0, 0, tview.NewTableCell("Name:").SetAlign(tview.AlignRight))
	selection.SetCell(1, 0, tview.NewTableCell("Replication factor:").SetAlign(tview.AlignRight))
	selection.SetCell(2, 0, tview.NewTableCell("Partitions:").SetAlign(tview.AlignRight))
	selection.SetCell(3, 0, tview.NewTableCell("Configs:").SetAlign(tview.AlignRight))
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)

	f := tview.NewFlex()
	f.SetDirection(tview.FlexColumn)
	f.AddItem(selection, 20, 0, true)
	f.AddItem(tview.NewBox(), 3, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topicNameField, 1, 0, false).
		AddItem(replicationFactorField, 1, 0, false).
		AddItem(partitionsField, 1, 0, false).
		AddItem(configTextArea, 0, 1, false)

	f.AddItem(inputs, 40, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	var editedConfig map[string]string

	configTextArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			propertiesText := configTextArea.GetText()
			editedConfig = parseConfig(propertiesText)
			app.SetFocus(selection)
			return nil
		}
		return event
	})

	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := selection.GetSelection()

		if event.Key() == tcell.KeyEnter {
			if row == 3 {
				app.SetFocus(configTextArea)
			}
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 's' {
			propertiesText := configTextArea.GetText()
			editedConfig = parseConfig(propertiesText)
			app.UpdateTopicResultHandler(topicName, editedConfig)
			app.HideModalPage(EditTopic)
			Publish(TopicsChannel, GetTopicsEventType, Payload{nil, false})
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(EditTopic)
		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f, 0, 1, true)
	flex.SetTitle(fmt.Sprintf(" Edit Topic: %s ", topicName))
	flex.SetBorder(true)

	modal := util.NewModal(flex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(EditTopic, modal, true, false)
}

func populateTable(table *tview.Table, row int, t string, partitions, replicas int) {
	table.SetCell(row, 0, tview.NewTableCell(t))
	table.SetCell(row, 1, tview.NewTableCell("P: "+strconv.Itoa(partitions)))
	table.SetCell(row, 2, tview.NewTableCell("R: "+strconv.Itoa(replicas)))
}

func filterTopicsTable(
	table *tview.Table,
	metadata map[string]*kafka.TopicMetadata,
	filter string,
) {
	table.Clear()

	var topics []string
	for topicName := range metadata {
		topics = append(topics, topicName)
	}

	if filter == "" {
		// Sort topics in ascending order if the filter is empty
		sort.Strings(topics)
		for i, topicName := range topics {
			meta := metadata[topicName]
			partitions := len(meta.Partitions)
			replicas := 0
			if len(meta.Partitions) > 0 {
				replicas = len(meta.Partitions[0].Replicas)
			}

			populateTable(table, i, topicName, partitions, replicas)
		}
		return
	}

	ranks := fuzzy.RankFind(filter, topics)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Distance < ranks[j].Distance
	})

	for i, rank := range ranks {
		topicName := rank.Target
		meta := metadata[topicName]
		partitions := len(meta.Partitions)
		replicas := 0
		if len(meta.Partitions) > 0 {
			replicas = len(meta.Partitions[0].Replicas)
		}

		populateTable(table, i, topicName, partitions, replicas)
	}
}

func (tp *TopicParams) validate() error {
	if strings.TrimSpace(tp.TopicName) == "" {
		return fmt.Errorf("topic name cannot be empty")
	}
	if tp.ReplicationFactor <= 0 {
		return fmt.Errorf("replication factor must be greater than 0")
	}
	if tp.Partitions <= 0 {
		return fmt.Errorf("partitions must be greater than 0")
	}
	return nil
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

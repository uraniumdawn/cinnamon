// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/consumer"
	"github.com/uraniumdawn/cinnamon/pkg/schemaregistry"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

// StartConsuming opens a streaming output page and starts the native consumer.
// formatFn controls how each record is rendered; pass formatConsumeRecord for JSON output.
func (app *App) StartConsuming(topicName string, params consumer.Params, formatFn func(consumer.Record) string) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	records := make(chan consumer.Record, 200)
	errs := make(chan error, 20)

	view := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(false).
		SetMaxLines(5000).
		SetScrollable(true).
		SetChangedFunc(func() { app.Draw() })
	view.SetBorder(true).SetBorderPadding(0, 0, 1, 0)

	pageName := util.BuildPageKey(app.Selected.Cluster.Name, ConsumeOutput, topicName)
	app.AddToPagesRegistry(pageName, view, ConsumeOutputPageMenu, false)

	var recordCount int64
	var isActive int32 = 1
	spinnerIdx := 0

	// Spinner goroutine — updates title while consuming.
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if atomic.LoadInt32(&isActive) == 0 {
				return
			}
			cnt := atomic.LoadInt64(&recordCount)
			frame := SpinnerFrames[spinnerIdx]
			spinnerIdx = (spinnerIdx + 1) % len(SpinnerFrames)
			app.QueueUpdateDraw(func() {
				view.SetTitle(fmt.Sprintf(" %s Consuming: %s [%d] ", frame, topicName, cnt))
			})
		}
	}()

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if IsKey(event, 't') {
			if atomic.LoadInt32(&isActive) == 0 {
				SendStatus("consumer already stopped", 2*time.Second, false)
				return nil
			}
			cancelFunc()
			SendStatusInfinite("stopping consumer…")
			return nil
		}
		if event.Key() == tcell.KeyCtrlD {
			if atomic.LoadInt32(&isActive) == 1 {
				SendStatus("consumer is still active — press t to stop first", 2*time.Second, false)
				return nil
			}
			app.RemoveFromPagesRegistry(pageName)
			return nil
		}
		return event
	})

	go consumer.Consume(ctx, params, records, errs)

	// Drain records; when the channel closes, finalize.
	go func() {
		defer func() {
			cancelFunc()
			atomic.StoreInt32(&isActive, 0)
			cnt := atomic.LoadInt64(&recordCount)
			app.QueueUpdateDraw(func() {
				view.SetTitle(fmt.Sprintf(" Consume: %s [%d records] ", topicName, cnt))
			})
			SendStatus(
				fmt.Sprintf("consuming stopped: %d records from %q", cnt, topicName),
				3*time.Second,
				false,
			)
		}()
		for rec := range records {
			atomic.AddInt64(&recordCount, 1)
			line := formatFn(rec)
			app.QueueUpdateDraw(func() {
				_, _ = fmt.Fprintf(view, "%s\n", line)
				view.ScrollToEnd()
			})
		}
	}()

	// Drain error channel independently so errors don't block the consumer.
	go func() {
		for err := range errs {
			app.QueueUpdateDraw(func() {
				_, _ = fmt.Fprintf(view, "[red]error: %s[-]\n", err.Error())
			})
		}
	}()
}

// toJSONValue embeds s as a raw JSON value if it is already valid JSON,
// otherwise wraps it as a JSON-encoded string. Empty input becomes null.
func toJSONValue(s string) string {
	if s == "" {
		return "null"
	}
	if json.Valid([]byte(s)) {
		return s
	}
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

// ConsumeModal opens a single-line kcat-style consume params input for topicName.
// Supported flags: -o beginning|end|<n>|s@<ts>|e@<ts>  -f <format>.
func (app *App) ConsumeModal(topicName string) {
	foregroundColor := tcell.GetColor(app.Colors.Cinnamon.Foreground)
	bgColor := tcell.GetColor(app.Colors.Cinnamon.Background)
	placeholderTextColor := tcell.GetColor(app.Colors.Cinnamon.Placeholder)

	input := tview.NewInputField().
		SetText("-o beginning").
		SetFieldWidth(0).
		SetFieldStyle(
			tcell.StyleDefault.
				Foreground(foregroundColor).
				Background(bgColor),
		).
		SetPlaceholderStyle(tcell.StyleDefault.Background(bgColor)).
		SetPlaceholderTextColor(placeholderTextColor)

	submit := func() {
		raw := input.GetText()
		spec, err := consumer.ParseConsumeArgs(raw)
		if err != nil {
			SendStatusWithDefaultTTL(fmt.Sprintf("[red]%s", err.Error()))
			return
		}

		kafkaConf := make(kafka.ConfigMap)
		for k, v := range app.Selected.Cluster.Properties {
			_ = kafkaConf.SetKey(k, v)
		}
		params := consumer.Params{
			KafkaConf:   &kafkaConf,
			Topic:       topicName,
			From:        spec.From,
			To:          spec.To,
			ExitOnEnd:   spec.ExitOnEnd,
			Partitions:  spec.Partitions,
			MaxCount:    spec.Count,
			KeySerdes:   spec.KeySerdes,
			ValueSerdes: spec.ValueSerdes,
		}

		// Resolve schema registry by name if avro serdes is requested.
		if spec.KeySerdes.Kind == consumer.SerdesAvro || spec.ValueSerdes.Kind == consumer.SerdesAvro {
			if spec.SRName == "" {
				SendStatusWithDefaultTTL("[red]-r <sr-name> is required when using avro serdes")
				return
			}
			srConfig, ok := app.SchemaRegistries[spec.SRName]
			if !ok {
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]schema registry %q not configured", spec.SRName),
				)
				return
			}
			srClient, ok := app.SchemaRegistryClients[spec.SRName]
			if !ok {
				var clientErr error
				srClient, clientErr = schemaregistry.NewSchemaRegistryClient(srConfig)
				if clientErr != nil {
					SendStatusWithDefaultTTL(
						fmt.Sprintf("[red]schema registry client: %s", clientErr),
					)
					return
				}
				app.SchemaRegistryClients[spec.SRName] = srClient
			}
			params.SRClient = srClient
		}

		var formatFn func(consumer.Record) string
		if spec.FormatStr != "" {
			fs := spec.FormatStr
			formatFn = func(r consumer.Record) string {
				return consumer.ApplyFormat(r, fs, topicName)
			}
		} else {
			formatFn = formatConsumeRecord
		}

		app.HideModalPage(ConsumeParams)
		app.StartConsuming(topicName, params, formatFn)
	}

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			submit()
			return nil
		case tcell.KeyEsc:
			app.HideModalPage(ConsumeParams)
			return nil

		case tcell.KeyF1:
			app.ConsumeHelpModal()
			app.ShowModalPage(ConsumeHelp)
		}
		return event
	})

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true)
	mainFlex.SetTitle(" Parameters ")
	mainFlex.SetBorder(true)

	modal := util.NewResourceModal(mainFlex, 7)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConsumeParams, modal, true, false)
}

// ConsumeHelpModal shows a read-only reference of all supported consume flags and format specifiers.
func (app *App) ConsumeHelpModal() {
	labelColor := app.Colors.Cinnamon.Label.FgColor
	dimColor := "gray"

	content := fmt.Sprintf(
		"[%s]Flags[-]\n"+
			"  [%s]-o[-]  beginning | earliest | end | latest | stored | <n> | -<n> | s@<ts> | e@<ts>\n"+
			"  [%s]-p[-]  <n>          restrict to partition (repeatable)\n"+
			"  [%s]-c[-]  <n>          stop after n messages\n"+
			"  [%s]-s[-]  avro | key=<serdes> | value=<serdes>   (serdes: avro, string, hex, pack)\n"+
			"  [%s]-r[-]  <sr-name>    schema registry name (required for avro)\n"+
			"  [%s]-f[-]  <format>     output format string (must be last flag)\n"+
			"  [%s]-e[-]               exit when end of each partition is reached\n"+
			"\n"+
			"[%s]Format specifiers for -f[-]\n"+
			"  [%s]%%k[-] key      [%s]%%s[-] value     [%s]%%p[-] partition\n"+
			"  [%s]%%o[-] offset   [%s]%%T[-] timestamp  [%s]%%t[-] topic\n"+
			"  [%s]%%h[-] headers  [%s]%%S[-] size (bytes)\n"+
			"\n"+
			"[%s]Timestamp formats[-]  unix-ms | RFC3339 | 2006-01-02T15:04:05.000",
		labelColor,
		labelColor, labelColor, labelColor, labelColor, labelColor, labelColor, labelColor,
		labelColor,
		labelColor, labelColor, labelColor,
		labelColor, labelColor, labelColor,
		labelColor, labelColor,
		dimColor,
	)

	view := tview.NewTextView().
		SetDynamicColors(true).
		SetText(content).
		SetScrollable(false)
	view.SetBorder(false).SetBorderPadding(0, 0, 1, 1)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || IsKey(event, '?') {
			app.HideModalPage(ConsumeHelp)
			return nil
		}
		return event
	})

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(view, 0, 1, true)
	mainFlex.SetTitle(" Consume Reference ")
	mainFlex.SetBorder(true)

	modal := util.NewResourceModal(mainFlex, 20)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConsumeHelp, modal, true, false)
}

// formatConsumeRecord renders a record as a single JSON line matching:
// {"Key":"%k","Value":%s,"Timestamp":%T,"Partition":%p,"Offset":%o,"Headers":"%h","Size":%S}
func formatConsumeRecord(r consumer.Record) string {
	headersJSON, _ := json.Marshal(strings.Join(r.Headers, ","))

	return fmt.Sprintf(
		`{"Key":%s,"Value":%s,"Timestamp":%d,"Partition":%d,"Offset":%d,"Headers":%s,"Size":%d}`,
		toJSONValue(r.Key),
		toJSONValue(r.Value),
		r.Timestamp.UnixMilli(),
		r.Partition,
		r.Offset,
		string(headersJSON),
		r.Size,
	)
}

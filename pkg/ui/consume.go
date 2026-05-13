// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

type consumeState struct {
	format    string // "json" or "avro"
	srName    string // selected schema registry name
	srURL     string // resolved URL
	from      string // "earliest", "latest", "offset", "timestamp"
	fromValue string // raw text: offset number or timestamp string
	to        string // "offset", "timestamp", or ""
	toValue   string // raw text: offset number or timestamp string
}

const (
	consumeRowFormat = 1
	consumeRowSR     = 2
	consumeRowFrom   = 3
	consumeRowTo     = 4
)

// ConsumeParamsModal builds and shows the consume parameters modal for the given topic.
func (app *App) ConsumeParamsModal(topicName string) {
	state := consumeState{
		format: "json",
		from:   "earliest",
	}

	// ── Color shortcuts ──────────────────────────────────────────────────────
	labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
	foregroundColor := tcell.GetColor(app.Colors.Cinnamon.Foreground)
	dimColor := tcell.ColorGray
	selectedStyle := tcell.StyleDefault.
		Foreground(tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor)).
		Background(tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor))
	placeholderTextColor := tcell.GetColor(app.Colors.Cinnamon.Placeholder)
	bgColor := tcell.GetColor(app.Colors.Cinnamon.Background)

	// ── Table ────────────────────────────────────────────────────────────────
	table := tview.NewTable()
	table.SetBorder(false)
	table.SetBorderPadding(0, 0, 1, 0)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(selectedStyle)

	mkHeader := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(false).SetTextColor(labelColor)
	}
	mkLabel := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(true)
	}
	mkValue := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(false)
	}

	table.SetCell(0, 0, mkHeader("Param"))
	table.SetCell(0, 1, mkHeader("Value"))

	table.SetCell(consumeRowFormat, 0, mkLabel("Format"))
	table.SetCell(consumeRowFormat, 1, mkValue(state.format))

	srLabelCell := mkLabel("Schema Registry")
	srLabelCell.SetTextColor(dimColor).SetSelectable(false)
	table.SetCell(consumeRowSR, 0, srLabelCell)
	table.SetCell(consumeRowSR, 1, mkValue(""))

	table.SetCell(consumeRowFrom, 0, mkLabel("from"))
	table.SetCell(consumeRowFrom, 1, mkValue(state.from))

	toLabelCell := mkLabel("to")
	toLabelCell.SetTextColor(dimColor).SetSelectable(false)
	table.SetCell(consumeRowTo, 0, toLabelCell)
	table.SetCell(consumeRowTo, 1, mkValue(""))

	// ── Input fields (rows 3 and 4 only) ────────────────────────────────────
	mkInputField := func() *tview.InputField {
		return tview.NewInputField().
			SetFieldWidth(30).
			SetFieldStyle(
				tcell.StyleDefault.
					Foreground(foregroundColor).
					Background(bgColor),
			).
			SetPlaceholderStyle(tcell.StyleDefault.Background(bgColor)).
			SetPlaceholderTextColor(placeholderTextColor)
	}

	inputFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	valueHeaderTable := tview.NewTable()
	valueHeaderTable.SetCell(0, 0,
		tview.NewTableCell("Input").SetTextColor(labelColor).SetSelectable(false),
	)
	inputFlex.AddItem(valueHeaderTable, 1, 0, false) // header
	inputFlex.AddItem(tview.NewBox(), 1, 0, false)   // row 1: Format (no input)
	inputFlex.AddItem(tview.NewBox(), 1, 0, false)   // row 2: SR (no input)

	fromInput := mkInputField()
	inputFlex.AddItem(fromInput, 1, 0, false)

	toInput := mkInputField()
	inputFlex.AddItem(toInput, 1, 0, false)

	container := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(table, 0, 2, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(inputFlex, 0, 1, false)

	innerPages := tview.NewPages()
	innerPages.AddPage("params", container, true, true)

	// ── Helpers ──────────────────────────────────────────────────────────────
	setRowEnabled := func(row int, enabled bool) {
		cell := table.GetCell(row, 0)
		if enabled {
			cell.SetTextColor(foregroundColor)
			cell.SetSelectable(true)
		} else {
			cell.SetTextColor(dimColor)
			cell.SetSelectable(false)
		}
	}

	needsInput := func(v string) bool {
		return v == "offset" || v == "timestamp"
	}

	setFromPlaceholder := func() {
		fromInput.SetText("")
		switch state.from {
		case "offset":
			fromInput.SetPlaceholder("0").SetPlaceholderTextColor(placeholderTextColor)
		case "timestamp":
			fromInput.SetPlaceholder("2006-01-02T15:04:05.000").
				SetPlaceholderTextColor(placeholderTextColor)
		default:
			fromInput.SetPlaceholder("")
		}
	}

	setToPlaceholder := func() {
		toInput.SetText("")
		switch state.to {
		case "offset":
			toInput.SetPlaceholder("0").SetPlaceholderTextColor(placeholderTextColor)
		case "timestamp":
			toInput.SetPlaceholder("2006-01-02T15:04:05.000").
				SetPlaceholderTextColor(placeholderTextColor)
		default:
			toInput.SetPlaceholder("")
		}
	}

	// ── Generic picker overlay ───────────────────────────────────────────────
	openPicker := func(labels, values []string, current string, onSelect func(string)) {
		pickerTable := tview.NewTable()
		pickerTable.SetBorder(true)
		pickerTable.SetSelectable(true, false)
		pickerTable.SetSelectedStyle(selectedStyle)

		initialRow := 0
		for i, label := range labels {
			pickerTable.SetCell(i, 0, tview.NewTableCell(" "+label).SetSelectable(true))
			if values[i] == current {
				initialRow = i
			}
		}
		pickerTable.Select(initialRow, 0)

		closePicker := func() {
			innerPages.RemovePage("picker")
			app.SetFocus(table)
		}

		pickerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				closePicker()
				return nil
			case tcell.KeyEnter:
				idx, _ := pickerTable.GetSelection()
				value := values[idx]
				closePicker()
				onSelect(value)
				return nil
			}
			return event
		})

		pickerH := len(labels) + 2
		pickerW := 24
		pickerWrapper := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 0, false).
				AddItem(pickerTable, pickerH, 0, true).
				AddItem(nil, 0, 1, false),
				pickerW, 0, true).
			AddItem(nil, 1, 0, false)

		innerPages.AddPage("picker", pickerWrapper, true, true)
		app.SetFocus(pickerTable)
	}

	// ── Row handlers called on Enter ─────────────────────────────────────────
	handleFormat := func() {
		openPicker(
			[]string{"json", "avro"},
			[]string{"json", "avro"},
			state.format,
			func(v string) {
				if v == "avro" && len(app.SchemaRegistries) == 0 {
					SendStatusWithDefaultTTL("[red]no schema registries configured")
					return
				}
				state.format = v
				table.GetCell(consumeRowFormat, 1).SetText(v)
				if v == "avro" {
					setRowEnabled(consumeRowSR, true)
				} else {
					setRowEnabled(consumeRowSR, false)
					state.srName = ""
					state.srURL = ""
					table.GetCell(consumeRowSR, 1).SetText("")
				}
			},
		)
	}

	handleSR := func() {
		srNames := make([]string, 0, len(app.SchemaRegistries))
		for name := range app.SchemaRegistries {
			srNames = append(srNames, name)
		}
		sort.Strings(srNames)
		openPicker(srNames, srNames, state.srName, func(v string) {
			state.srName = v
			if sr := app.SchemaRegistries[v]; sr != nil {
				state.srURL = sr.SchemaRegistryURL
			}
			table.GetCell(consumeRowSR, 1).SetText(v)
		})
	}

	handleFrom := func() {
		labels := []string{"earliest", "latest", "offset", "timestamp"}
		openPicker(labels, labels, state.from, func(v string) {
			state.from = v
			state.fromValue = ""
			table.GetCell(consumeRowFrom, 1).SetText(v)
			setFromPlaceholder()

			if needsInput(v) {
				setRowEnabled(consumeRowTo, true)
				state.to = ""
				state.toValue = ""
				table.GetCell(consumeRowTo, 1).SetText("")
				setToPlaceholder()
				app.SetFocus(fromInput)
			} else {
				setRowEnabled(consumeRowTo, false)
				state.to = ""
				state.toValue = ""
				table.GetCell(consumeRowTo, 1).SetText("")
				toInput.SetText("").SetPlaceholder("")
			}
		})
	}

	handleTo := func() {
		var labels, values []string
		switch state.from {
		case "offset":
			labels = []string{"(none)", "offset"}
			values = []string{"", "offset"}
		case "timestamp":
			labels = []string{"(none)", "timestamp"}
			values = []string{"", "timestamp"}
		default:
			return
		}
		openPicker(labels, values, state.to, func(v string) {
			state.to = v
			state.toValue = ""
			table.GetCell(consumeRowTo, 1).SetText(v)
			setToPlaceholder()
			if needsInput(v) {
				app.SetFocus(toInput)
			}
		})
	}

	// ── Table input capture ──────────────────────────────────────────────────
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		switch {
		case event.Key() == tcell.KeyEsc:
			app.HideModalPage(ConsumeParams)

		case event.Key() == tcell.KeyEnter:
			switch row {
			case consumeRowFormat:
				handleFormat()
			case consumeRowSR:
				handleSR()
			case consumeRowFrom:
				handleFrom()
			case consumeRowTo:
				handleTo()
			}
			return nil

		case event.Key() == tcell.KeyCtrlS:
			state.fromValue = fromInput.GetText()
			state.toValue = toInput.GetText()

			params, err := buildConsumeParams(app, topicName, state)
			if err != nil {
				SendStatusWithDefaultTTL(fmt.Sprintf("[red]%s", err.Error()))
				return event
			}
			app.HideModalPage(ConsumeParams)
			app.StartConsuming(topicName, params, formatConsumeRecord)
		}
		return event
	})

	// ── Input field handlers ─────────────────────────────────────────────────
	fromInput.SetDoneFunc(func(_ tcell.Key) {
		state.fromValue = fromInput.GetText()
		app.SetFocus(table)
	})
	fromInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyEnter:
			state.fromValue = fromInput.GetText()
			app.SetFocus(table)
			return nil
		case tcell.KeyTab:
			state.fromValue = fromInput.GetText()
			if needsInput(state.to) {
				app.SetFocus(toInput)
			} else {
				app.SetFocus(table)
			}
			return nil
		case tcell.KeyBacktab:
			state.fromValue = fromInput.GetText()
			app.SetFocus(table)
			return nil
		}
		return event
	})

	toInput.SetDoneFunc(func(_ tcell.Key) {
		state.toValue = toInput.GetText()
		app.SetFocus(table)
	})
	toInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyEnter:
			state.toValue = toInput.GetText()
			app.SetFocus(table)
			return nil
		case tcell.KeyBacktab:
			state.toValue = toInput.GetText()
			if needsInput(state.from) {
				app.SetFocus(fromInput)
			} else {
				app.SetFocus(table)
			}
			return nil
		case tcell.KeyTab:
			state.toValue = toInput.GetText()
			app.SetFocus(table)
			return nil
		}
		return event
	})

	// ── Layout ────────────────────────────────────────────────────────────────
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(innerPages, 0, 1, true)
	mainFlex.SetTitle(fmt.Sprintf(" Consume Parameters: %s ", topicName))
	mainFlex.SetBorder(true)

	modal := util.NewResourceModal(mainFlex, 9)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConsumeParams, modal, true, false)
}

// buildConsumeParams converts the modal state into consumer.Params.
func buildConsumeParams(app *App, topicName string, state consumeState) (consumer.Params, error) {
	kafkaConf := make(kafka.ConfigMap)
	for k, v := range app.Selected.Cluster.Properties {
		_ = kafkaConf.SetKey(k, v)
	}

	params := consumer.Params{
		KafkaConf: &kafkaConf,
		Topic:     topicName,
	}

	switch state.format {
	case "avro":
		params.Format = consumer.FormatAvro
		srClient := app.GetCurrentSchemaRegistryClient()
		if srClient == nil {
			return consumer.Params{}, fmt.Errorf("schema registry not selected")
		}
		params.SRClient = srClient
	default:
		params.Format = consumer.FormatJSON
	}

	switch state.from {
	case "earliest":
		params.From = consumer.FromSpec{Type: "beginning"}
	case "latest":
		params.From = consumer.FromSpec{Type: "end"}
	case "offset":
		off, err := strconv.ParseInt(state.fromValue, 10, 64)
		if err != nil || off < 0 {
			return consumer.Params{}, fmt.Errorf("invalid from offset: %s", state.fromValue)
		}
		params.From = consumer.FromSpec{Type: "offset", Offset: off}
	case "timestamp":
		t, err := parseTimestamp(state.fromValue)
		if err != nil {
			return consumer.Params{}, fmt.Errorf("invalid from timestamp: %s", state.fromValue)
		}
		params.From = consumer.FromSpec{Type: "timestamp", Timestamp: t.UnixMilli()}
	default:
		return consumer.Params{}, fmt.Errorf("'from' is required")
	}

	if state.to != "" && state.toValue != "" {
		switch state.to {
		case "offset":
			off, err := strconv.ParseInt(state.toValue, 10, 64)
			if err != nil || off < 0 {
				return consumer.Params{}, fmt.Errorf("invalid to offset: %s", state.toValue)
			}
			params.To = consumer.ToSpec{Type: "offset", Offset: off}
		case "timestamp":
			t, err := parseTimestamp(state.toValue)
			if err != nil {
				return consumer.Params{}, fmt.Errorf("invalid to timestamp: %s", state.toValue)
			}
			params.To = consumer.ToSpec{Type: "timestamp", Timestamp: t.UnixMilli()}
		}
	}

	return params, nil
}

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

// ConsumeKcatModal opens a single-line kcat-style consume params input for topicName.
// Supported flags: -o beginning|end|<n>|s@<ts>|e@<ts>  -f <format>.
func (app *App) ConsumeKcatModal(topicName string) {
	foregroundColor := tcell.GetColor(app.Colors.Cinnamon.Foreground)
	bgColor := tcell.GetColor(app.Colors.Cinnamon.Background)
	placeholderTextColor := tcell.GetColor(app.Colors.Cinnamon.Placeholder)
	dimColor := tcell.ColorGray

	hint := tview.NewTextView().
		SetText("-o beginning|end|stored|-<n>|s@<ms>|e@<ms>  -p <n>  -c <n>  -s key|value=avro|<pack>  -r <sr>  -f <fmt>  -e").
		SetTextColor(dimColor).
		SetDynamicColors(false)
	hint.SetBorder(false).SetBorderPadding(0, 0, 1, 0)

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
		spec, err := consumer.ParseKcatArgs(raw)
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
				return consumer.ApplyKcatFormat(r, fs, topicName)
			}
		} else {
			formatFn = formatConsumeRecord
		}

		app.HideModalPage(ConsumeKcatParams)
		app.StartConsuming(topicName, params, formatFn)
	}

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			submit()
			return nil
		case tcell.KeyEsc:
			app.HideModalPage(ConsumeKcatParams)
			return nil
		}
		return event
	})

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hint, 2, 0, false).
		AddItem(input, 1, 0, true)
	mainFlex.SetTitle(fmt.Sprintf(" Consume (kcat): %s ", topicName))
	mainFlex.SetBorder(true)

	modal := util.NewResourceModal(mainFlex, 7)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConsumeKcatParams, modal, true, false)
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

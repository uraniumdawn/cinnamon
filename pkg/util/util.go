// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package util

import (
	"bytes"
	"cinnamon/pkg/config"
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/rivo/tview"
)

func ParseShellCommand(templateStr, topic string) ([]string, error) {
	tmpl, err := template.New("cmd").Parse(templateStr)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{"topic": topic})
	if err != nil {
		return nil, err
	}

	expanded := buf.String()

	// Split into args — use Fields to split on spaces
	// If you need quotes preserved, use shellwords or shlex logic
	args := strings.Fields(expanded)

	return args, nil
}

func TableToCSV(fileName string, table *tview.Table) {
	file, _ := os.Create(fileName)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	rows, cols := table.GetRowCount(), table.GetColumnCount()
	for row := 0; row < rows; row++ {
		var record []string
		for col := 0; col < cols; col++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				record = append(record, cell.Text)
			} else {
				record = append(record, "")
			}
		}
		writer.Write(record)
	}
}

func NewModal(p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 0, 2, true).
			AddItem(nil, 0, 1, false), 0, 5, true).
		AddItem(nil, 0, 1, false)
}

func NewConfirmationModal(p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 3, 0, true).
			AddItem(nil, 0, 1, false), 0, 2, true).
		AddItem(nil, 0, 1, false)
}

func GetInt64(inputField *tview.InputField) int64 {
	text := inputField.GetText()
	if text == "" {
		return -1
	}
	res, _ := strconv.ParseInt(text, 10, 64)
	return res
}

func GetInt32(inputField *tview.InputField) int32 {
	text := inputField.GetText()
	if text == "" {
		return -1
	}
	res, _ := strconv.ParseInt(text, 10, 32)
	return int32(res)
}

func ToClustersMap(cfg *config.Config) map[string]*config.ClusterConfig {
	clusterMap := make(map[string]*config.ClusterConfig)
	for _, cluster := range cfg.Cinnamon.Clusters {
		clusterMap[cluster.Name] = cluster
	}
	return clusterMap
}

func ToSchemaRegistryMap(cfg *config.Config) map[string]*config.SchemaRegistryConfig {
	srMap := make(map[string]*config.SchemaRegistryConfig)
	for _, sr := range cfg.Cinnamon.SchemaRegistries {
		srMap[sr.Name] = sr
	}
	return srMap
}

func BuildTitle(parts ...string) string {
	var builder strings.Builder
	builder.WriteString(" ")
	for i, part := range parts {
		builder.WriteString(strings.ToLower(part))
		if i < len(parts)-1 {
			builder.WriteString(":")
		}
	}
	builder.WriteString(" ")
	return builder.String()
}

func BuildPageKey(parts ...string) string {
	var builder strings.Builder
	for i, part := range parts {
		builder.WriteString(strings.ToLower(part))
		if i < len(parts)-1 {
			builder.WriteString(":")
		}
	}
	return builder.String()
}

package util

import (
	"encoding/csv"
	"github.com/rivo/tview"
	"os"
	"strconv"
)

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
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 0, 2, true).
			AddItem(nil, 0, 1, false), 0, 5, true).
		AddItem(nil, 0, 1, false)
	return flex
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

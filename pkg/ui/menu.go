package ui

import "github.com/rivo/tview"

type Menu struct {
	Content *tview.Table
	Flex    *tview.Flex
	Map     *map[string]*[]string
}

type Pair struct {
	Key   			string
	Value string
}

var keys = map[string]Pair{
	"dw": {
		Key:   "[blue]<j, ↓>",
		Value: "[grey]Down",
	},
	"up": {
		Key:   "[blue]<k, ↑>",
		Value: "[grey]Up",
	},
	"forward": {
		Key:   "[blue]<f>",
		Value: "[grey]Forward",
	},
	"backward": {
		Key:   "[blue]<b>",
		Value: "[grey]Backward",
	},
	"select": {
		Key:   "[blue]<Enter>",
		Value: "[grey]Select",
	},
	"res": {
		Key:   "[blue]<:>",
		Value: "[grey]Resources",
	},
	"opened": {
		Key:   "[blue]<Ctrl+p>",
		Value: "[grey]Opened Pages",
	},
	"filter": {
		Key:   "[blue]</>",
		Value: "[grey]Filter",
	},
	"dsc": {
		Key:   "[blue]<d>",
		Value: "[grey]Describe Resource",
	},
	"upd": {
		Key:   "[blue]<Ctrl+u>",
		Value: "[grey]Update",
	},
	"rft": {
		Key:   "[blue]<r>",
		Value: "[grey]Read",
	},
	"params": {
		Key:   "[blue]<p>",
		Value: "[grey]Consuming parameters",
	},
	"term": {
		Key:   "[blue]<e>",
		Value: "[grey]Terminating",
	},
	"q": {
		Key:   "<q>",
		Value: "",
	},
}

const (
	MainPageMenu           = "MainPageMenu"
	ClustersPageMenu       = "ClustersPageMenu"
	NodesPageMenu          = "NodesPageMenu"
	TopicsPageMenu         = "TopicsPageMenu"
	ConsumingMenu          = "ConsumingMenu"
	ConsumerGroupsPageMenu = "ConsumerGroupsPageMenu"
	SubjectsPageMenu       = "SubjectsPageMenu"
	VersionsPageMenu       = "VersionsPageMenu"
	FinalPageMenu          = "FinalPageMenu"
)

func NewMenu() *Menu {
	table := tview.NewTable().
		SetSelectable(false, false).
		SetBorders(false)
	table.SetBorderPadding(0, 0, 1, 0)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	flex.AddItem(table, 0, 10, true)

	return &Menu{
		Content: table,
		Flex:    flex,
		Map: &map[string]*[]string{
			MainPageMenu:           {"up", "dw", "select", "res", "forward", "backward", "opened"},
			ClustersPageMenu:       {"up", "dw", "select", "res", "forward", "backward", "opened", "dsc"},
			NodesPageMenu:          {"up", "dw", "select", "res", "forward", "backward", "opened", "dsc", "upd"},
			TopicsPageMenu:         {"up", "dw", "select", "res", "forward", "backward", "opened", "dsc", "filter", "upd", "rft", "params"},
			ConsumingMenu:          {"forward", "backward", "res", "params", "term"},
			ConsumerGroupsPageMenu: {"up", "dw", "select", "res", "forward", "backward", "opened", "dsc", "filter", "upd"},
			SubjectsPageMenu:       {"up", "dw", "select", "res", "forward", "backward", "opened", "filter", "upd"},
			VersionsPageMenu:       {"up", "dw", "res", "forward", "backward", "opened", "dsc", "upd"},
			FinalPageMenu:          {"res", "forward", "backward", "opened", "upd"},
		},
	}
}

func (m *Menu) SetMenu(menu string) {
	row := 0
	col := 0
	m.Content.Clear()
	if keyBindings, ok := (*m.Map)[menu]; ok {
		for _, binding := range *keyBindings {
			if value, exists := keys[binding]; exists {
				m.Content.SetCell(row, col, tview.NewTableCell(value.Key))
				m.Content.SetCell(row, col+1, tview.NewTableCell(value.Value))
				row++
				if row > 3 {
					col += 2
					row = 0
				}
			}
		}
	}
}

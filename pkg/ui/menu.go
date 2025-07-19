package ui

import "github.com/rivo/tview"

type Menu struct {
	Content *tview.Table
	Flex    *tview.Flex
	Map     *map[string]*[]string
}

type Pair struct {
	Key   string
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
		Value: "[grey]Search",
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
	ResourcesPageMenu      = "ResourcesPageMenu"
	OpenedPageMenu         = "OpenedPageMenu"
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
		SetSelectable(false, false)
	table.SetBorderPadding(0, 0, 1, 0)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	flex.AddItem(table, 0, 1, true)

	return &Menu{
		Content: table,
		Flex:    flex,
		Map: &map[string]*[]string{
			MainPageMenu:      {"up", "dw", "select", "res", "forward", "backward", "opened"},
			ResourcesPageMenu: {"up", "dw", "select"},
			OpenedPageMenu:    {"up", "dw", "select"},
			ClustersPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
			},
			NodesPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"upd",
			},
			TopicsPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"filter",
				"upd",
			},
			ConsumingMenu: {"forward", "backward", "res", "params", "term"},
			ConsumerGroupsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"filter",
				"upd",
			},
			SubjectsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"filter",
				"upd",
			},
			VersionsPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"upd",
			},
			FinalPageMenu: {"res", "forward", "backward", "opened", "upd"},
		},
	}
}

func (m *Menu) SetMenu(menu string) {
	col := 0
	m.Content.Clear()
	if keyBindings, ok := (*m.Map)[menu]; ok {
		for i, binding := range *keyBindings {
			if value, exists := keys[binding]; exists {
				m.Content.SetCell(0, col, tview.NewTableCell(value.Key))
				m.Content.SetCell(0, col+1, tview.NewTableCell(value.Value))
				col += 2
				if i < len(*keyBindings)-1 {
					m.Content.SetCell(0, col, tview.NewTableCell(","))
					col++
				}
			}
		}
	}
}

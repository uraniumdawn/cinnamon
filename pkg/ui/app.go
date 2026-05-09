// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package ui provides the terminal user interface for the cinnamon application.
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/client"
	"github.com/uraniumdawn/cinnamon/pkg/config"
	"github.com/uraniumdawn/cinnamon/pkg/connect"
	"github.com/uraniumdawn/cinnamon/pkg/schemaregistry"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	Resources              = "Resources"
	Clusters               = "Clusters"
	SchemaRegistries       = "Schema-registries"
	Topics                 = "Topics"
	Topic                  = "Topic"
	Nodes                  = "Nodes"
	Node                   = "Node"
	ConsumerGroups         = "Consumer groups"
	ConsumerGroup          = "Consumer group"
	Subjects               = "Subjects"
	Connectors             = "Connectors"
	Connect                = "Connect"
	OpenedPages            = "Opened pages"
	CreateTopic            = "Create Topic"
	DeleteTopic            = "Delete Topic"
	DeleteConsumerGroup    = "Delete Consumer Group"
	EditTopic              = "Edit Topic"
	ResetOffset            = "Reset Offset"
	ConnectorConfigConfirm = "Connector Config Confirm"
	ConnectorActions       = "Connector Actions"
	DeleteConnector        = "Delete Connector"
	CliTemplates           = "CLI Templates"
)

type App struct {
	*tview.Application
	Layout                *Layout
	Cache                 *cache.Cache
	Clusters              map[string]*config.ClusterConfig
	SchemaRegistries      map[string]*config.SchemaRegistryConfig
	Connects              map[string]*config.ConnectConfig
	KafkaClients          map[string]*client.Client
	SchemaRegistryClients map[string]*schemaregistry.Client
	ConnectClients        map[string]*connect.Client
	Selected              Selected
	Config                *config.Config
	Colors                *config.ColorConfig
	// ModalHideTimer        *time.Timer
	CurrentFilters       map[string]string // pageName -> filter text for search preservation
	cgroupPrevLag        map[string]map[string]int64
	ResourcesSearchInput *tview.InputField
}

type Selected struct {
	Cluster        *config.ClusterConfig
	SchemaRegistry *config.SchemaRegistryConfig
	Connect        *config.ConnectConfig
}

func (app *App) GetCurrentKafkaClient() *client.Client {
	return app.KafkaClients[app.Selected.Cluster.Name]
}

// IsCurrentClusterReadOnly reports whether the selected cluster is in read-only mode.
func (app *App) IsCurrentClusterReadOnly() bool {
	return app.Selected.Cluster != nil && app.Selected.Cluster.IsReadOnly()
}

// IsCurrentConnectReadOnly reports whether the selected Connect cluster is in read-only mode.
func (app *App) IsCurrentConnectReadOnly() bool {
	return app.Selected.Connect != nil && app.Selected.Connect.IsReadOnly()
}

func (app *App) GetCurrentSchemaRegistryClient() *schemaregistry.Client {
	if app.Selected.SchemaRegistry == nil {
		return nil
	}
	return app.SchemaRegistryClients[app.Selected.SchemaRegistry.Name]
}

func (app *App) GetCurrentConnectClient() *connect.Client {
	if app.Selected.Connect == nil {
		return nil
	}
	return app.ConnectClients[app.Selected.Connect.Name]
}

func NewApp() *App {
	InitLogger()

	cfg, err := config.LoadAppConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize config")
		os.Exit(1)
	}

	colors, err := config.LoadColorConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load color config")
		os.Exit(1)
	}

	app := &App{
		Application:           tview.NewApplication(),
		Cache:                 cache.New(5*time.Minute, 10*time.Minute),
		Clusters:              util.ToClustersMap(cfg),
		SchemaRegistries:      util.ToSchemaRegistryMap(cfg),
		Connects:              util.ToConnectMap(cfg),
		KafkaClients:          make(map[string]*client.Client),
		SchemaRegistryClients: make(map[string]*schemaregistry.Client),
		ConnectClients:        make(map[string]*connect.Client),
		Config:                cfg,
		Colors:                colors,
		CurrentFilters:        make(map[string]string),
		cgroupPrevLag:         make(map[string]map[string]int64),
	}

	return app
}

func InitLogger() {
	zerolog.TimeFieldFormat = time.RFC3339

	logFilePath := filepath.Join(os.Getenv("HOME"), ".config", "cinnamon", "cinnamon.log")
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Printf("failed to create log directory, %s\n", err.Error())
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		fmt.Printf("failed to open log file, %s\n", err.Error())
		os.Exit(1)
	}

	// Use ConsoleWriter for human-readable, canonical log format
	consoleWriter := zerolog.ConsoleWriter{
		Out:        file,
		TimeFormat: time.RFC3339,
		NoColor:    true, // Set to true if colors are not desired in log file
	}

	os.Stderr = file
	os.Stdout = file

	// Add caller information (file and line number) to all log entries
	log.Logger = log.Output(consoleWriter).With().Caller().Logger()
}

func (app *App) Run() {
	app.ApplyColors()
	ctx, cancel := context.WithCancel(context.Background())

	app.RunResourcesEventHandler(ctx, ResourcesChannel)
	app.RunStatusLineHandler(ctx, StatusLineCh)
	app.RunClusterEventHandler(ctx, ClustersChannel)
	app.RunSchemaRegistriesEventHandler(ctx, SchemaRegistriesChannel)
	app.RunConnectEventHandler(ctx, ConnectChannel)
	app.RunNodesEventHandler(ctx, NodesChannel)
	app.RunTopicsEventHandler(ctx, TopicsChannel)
	app.RunCgroupsEventHandler(ctx, CgroupsChannel)
	app.RunSubjectsEventHandler(ctx, SubjectsChannel)
	app.RunConnectorsEventHandler(ctx, ConnectorsChannel)

	registry := NewPagesRegistry(app.Colors)
	hasSR := len(app.Config.Cinnamon.SchemaRegistries) > 0
	hasConnect := len(app.Config.Cinnamon.Connect) > 0
	app.Layout = NewLayout(registry, app.Colors, hasSR, hasConnect)

	for _, c := range app.Config.Cinnamon.Clusters {
		if c.Selected {
			app.SelectCluster(c, false)
			break // Assuming only one can be selected
		}
	}

	for _, sr := range app.Config.Cinnamon.SchemaRegistries {
		if sr.Selected {
			app.SelectSchemaRegistry(sr, false)
			break // Assuming only one can be selected
		}
	}

	for _, cn := range app.Config.Cinnamon.Connect {
		if cn.Selected {
			app.SelectConnect(cn, false)
			break
		}
	}

	Publish(ClustersChannel, GetClustersEventType, Payload{nil, false})
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry, app.Selected.Connect)

	resourcesPage := app.NewResourcesPage()
	app.Layout.PagesRegistry.UI.Pages.AddPage(Resources, resourcesPage, true, false)
	app.Layout.PagesRegistry.UI.Pages.AddPage(
		OpenedPages,
		app.Layout.PagesRegistry.UI.Main,
		true,
		false,
	)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(Clusters)
	app.Layout.Menu.SetMenu(ClustersPageMenu)
	app.Layout.PagesRegistry.UI.FilteredPages.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	app.OpenPagesKeyHandler(app.Layout.PagesRegistry.UI.FilteredPages)
	app.MainOperationKeyHandler()

	err := app.SetRoot(app.Layout.Content, true).Run()
	if err != nil {
		log.Error().Err(err).Msg("failed application execution")
	}
	cancel()
	log.Info().Msg("application terminated")
}

func (app *App) ApplyColors() {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    tcell.GetColor(app.Colors.Cinnamon.Background),
		ContrastBackgroundColor:     tview.Styles.ContrastBackgroundColor,
		MoreContrastBackgroundColor: tview.Styles.MoreContrastBackgroundColor,
		BorderColor:                 tcell.GetColor(app.Colors.Cinnamon.Border),
		TitleColor:                  tcell.GetColor(app.Colors.Cinnamon.Title),
		GraphicsColor:               tview.Styles.GraphicsColor,
		PrimaryTextColor:            tcell.GetColor(app.Colors.Cinnamon.Foreground),
		SecondaryTextColor:          tview.Styles.SecondaryTextColor,
		TertiaryTextColor:           tview.Styles.TertiaryTextColor,
		InverseTextColor:            tview.Styles.InverseTextColor,
		ContrastSecondaryTextColor:  tview.Styles.ContrastSecondaryTextColor,
	}
}

func (app *App) isClusterSelected(selected Selected) bool {
	return selected.Cluster != nil
}

func (app *App) isSchemaRegistrySelected(selected Selected) bool {
	return selected.SchemaRegistry != nil
}

func (app *App) isConnectSelected(selected Selected) bool {
	return selected.Connect != nil
}

func (app *App) SelectCluster(cluster *config.ClusterConfig, save bool) {
	if save {
		for _, c := range app.Config.Cinnamon.Clusters {
			c.Selected = c.Name == cluster.Name
		}
		if err := app.Config.Save(); err != nil {
			log.Error().Err(err).Msg("failed to save config after cluster selection")
		}
	}

	app.Selected.Cluster = cluster
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry, app.Selected.Connect)

	_, exists := app.KafkaClients[cluster.Name]
	if !exists {
		var err error
		timeout := app.Config.GetAPICallTimeout()
		newClient, err := client.NewClient(cluster, timeout)
		if err != nil {
			log.Error().Err(err).Msg("failed to create admin client")
			os.Exit(1)
		}
		app.KafkaClients[cluster.Name] = newClient
	}
}

func (app *App) SelectSchemaRegistry(sr *config.SchemaRegistryConfig, save bool) {
	if save {
		for _, r := range app.Config.Cinnamon.SchemaRegistries {
			r.Selected = r.Name == sr.Name
		}
		if err := app.Config.Save(); err != nil {
			log.Error().Err(err).Msg("failed to save config after schema registry selection")
		}
	}

	app.Selected.SchemaRegistry = sr
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry, app.Selected.Connect)

	_, exists := app.SchemaRegistryClients[sr.Name]
	if !exists {
		var err error
		newClient, err := schemaregistry.NewSchemaRegistryClient(sr)
		if err != nil {
			log.Error().Err(err).Msg("failed to create admin client")
			os.Exit(1)
		}
		app.SchemaRegistryClients[sr.Name] = newClient
	}
}

func (app *App) SelectConnect(cfg *config.ConnectConfig, save bool) {
	if save {
		for _, c := range app.Config.Cinnamon.Connect {
			c.Selected = c.Name == cfg.Name
		}
		if err := app.Config.Save(); err != nil {
			log.Error().Err(err).Msg("failed to save config after connect selection")
		}
	}

	app.Selected.Connect = cfg
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry, app.Selected.Connect)

	_, exists := app.ConnectClients[cfg.Name]
	if !exists {
		newClient, err := connect.NewClient(cfg, app.Config.GetAPICallTimeout())
		if err != nil {
			log.Error().Err(err).Msg("failed to create connect client")
			return
		}
		app.ConnectClients[cfg.Name] = newClient
	}
}

func (app *App) NewDescription(title string) *tview.TextView {
	desc := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(false)
	desc.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(title)
	desc.SetTextColor(tcell.GetColor(app.Colors.Cinnamon.Foreground))
	return desc
}

// WithHScroll wraps an input capture handler to add H/L horizontal scrolling to a TextView.
func (app *App) WithHScroll(
	desc *tview.TextView,
	handler func(*tcell.EventKey) *tcell.EventKey,
) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		row, col := desc.GetScrollOffset()
		if IsKey(event, 'H') {
			if col > 0 {
				desc.ScrollTo(row, col-5)
			}
			return nil
		}
		if IsKey(event, 'L') {
			desc.ScrollTo(row, col+5)
			return nil
		}
		return handler(event)
	}
}

// ClearCurrentFilter clears the saved filter for the current page.
func (app *App) ClearCurrentFilter() {
	currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
	delete(app.CurrentFilters, currentPage)

	// Also clear the search input if it exists
	if search, exists := app.Layout.Search[currentPage]; exists {
		search.SetText("")
	}
}

// ClearFilterForPage clears the saved filter for a specific page.
func (app *App) ClearFilterForPage(pageName string) {
	delete(app.CurrentFilters, pageName)
}

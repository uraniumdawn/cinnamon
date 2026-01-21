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
	"github.com/uraniumdawn/cinnamon/pkg/schemaregistry"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const timeout = time.Second * 10
const (
	Resources        = "Resources"
	Clusters         = "Clusters"
	SchemaRegistries = "Schema-registries"
	Topics           = "Topics"
	Topic            = "Topic"
	Nodes            = "Nodes"
	Node             = "Node"
	ConsumerGroups   = "Consumer groups"
	ConsumerGroup    = "Consumer group"
	Subjects         = "Subjects"
	OpenedPages      = "Opened pages"
	ConsumingParams  = "Consuming Parameters"
	CreateTopic      = "Create Topic"
	DeleteTopic      = "Delete Topic"
	EditTopic        = "Edit Topic"
	CliTemplates     = "CLI Templates"
)

type App struct {
	*tview.Application
	Layout                *Layout
	Cache                 *cache.Cache
	Clusters              map[string]*config.ClusterConfig
	SchemaRegistries      map[string]*config.SchemaRegistryConfig
	KafkaClients          map[string]*client.Client
	SchemaRegistryClients map[string]*schemaregistry.Client
	Selected              Selected
	Config                *config.Config
	Colors                *config.ColorConfig
	ModalHideTimer        *time.Timer
}

type Selected struct {
	Cluster        *config.ClusterConfig
	SchemaRegistry *config.SchemaRegistryConfig
}

func (app *App) GetCurrentKafkaClient() *client.Client {
	return app.KafkaClients[app.Selected.Cluster.Name]
}

func (app *App) GetCurrentSchemaRegistryClient() *schemaregistry.Client {
	if app.Selected.SchemaRegistry == nil {
		return nil
	}
	return app.SchemaRegistryClients[app.Selected.SchemaRegistry.Name]
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
		KafkaClients:          make(map[string]*client.Client),
		SchemaRegistryClients: make(map[string]*schemaregistry.Client),
		Config:                cfg,
		Colors:                colors,
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

var statusLineCh = make(chan string, 10)

func ClearStatus() {
	statusLineCh <- ""
}

var statusLineTimer *time.Timer

func (app *App) RunStatusLineHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("shutting down status line handler")
				return
			case status := <-in:
				app.QueueUpdateDraw(func() {
					if status != "" {
						app.Layout.StatusHistory.AddEntry(status)
						app.Layout.StatusLine.SetText(status)

						// Auto-clear after 5 seconds
						if statusLineTimer != nil {
							statusLineTimer.Stop()
						}
						statusLineTimer = time.AfterFunc(5*time.Second, func() {
							app.QueueUpdateDraw(func() {
								app.Layout.StatusLine.SetText("")
							})
						})
					} else {
						// Clear status line immediately
						app.Layout.StatusLine.SetText("")
						if statusLineTimer != nil {
							statusLineTimer.Stop()
						}
					}
				})
			}
		}
	}()
}

func (app *App) Run() {
	app.ApplyColors()
	ctx, cancel := context.WithCancel(context.Background())

	app.RunResourcesEventHandler(ctx, ResourcesChannel)
	app.RunStatusLineHandler(ctx, statusLineCh)
	app.RunClusterEventHandler(ctx, ClustersChannel)
	app.RunSchemaRegistriesEventHandler(ctx, SchemaRegistriesChannel)
	app.RunNodesEventHandler(ctx, NodesChannel)
	app.RunTopicsEventHandler(ctx, TopicsChannel)
	app.RunCgroupsEventHandler(ctx, CgroupsChannel)
	app.RunSubjectsEventHandler(ctx, SubjectsChannel)

	registry := NewPagesRegistry(app.Colors)
	app.Layout = NewLayout(registry, app.Colors)

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

	Publish(ClustersChannel, GetClustersEventType, Payload{nil, false})
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry)

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
	app.Layout.PagesRegistry.UI.OpenedPages.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	app.OpenPagesKeyHandler(app.Layout.PagesRegistry.UI.OpenedPages)
	app.StatusHistoryKeyHandler(app.Layout.StatusHistory.View)
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
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry)

	_, exists := app.KafkaClients[cluster.Name]
	if !exists {
		var err error
		newClient, err := client.NewClient(cluster)
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
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry)

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

func (app *App) NewDescription(title string) *tview.TextView {
	desc := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(false)
	desc.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(title)
	desc.SetTextColor(tcell.GetColor(app.Colors.Cinnamon.Foreground))
	return desc
}

// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"cinnamon/pkg/config"
	"cinnamon/pkg/schemaregistry"
	"cinnamon/pkg/util"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const timeout = time.Second * 10
const (
	Resources        = "Resources"
	Clusters         = "Clusters"
	Cluster          = "Cluster"
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
	mu                    sync.RWMutex
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

	os.Stderr = file
	os.Stdout = file
	log.Logger = log.Output(file)
}

var (
	statusLineCh = make(chan string, 10)
	commandCh    = make(chan string)
)

func ClearStatus() {
	statusLineCh <- ""
}

func (app *App) RunCommandHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down CommandHandler")
				return
			case command := <-in:
				switch command {
				case Clusters:
					app.QueueUpdateDraw(func() {
						app.SwitchToPage(Clusters)
					})
				case SchemaRegistries:
					app.QueueUpdateDraw(func() {
						app.SwitchToPage(SchemaRegistries)
					})
				case "tps", Topics:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select Cluster"
						continue
					}
					app.CheckInCache(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Topics),
						func() {
							app.Topics()
						},
					)
				case "grs", ConsumerGroups:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select Cluster"
						continue
					}
					app.CheckInCache(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, ConsumerGroups),
						func() {
							app.ConsumerGroups()
						},
					)
				case "nds", Nodes:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select Cluster"
						continue
					}
					app.CheckInCache(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Nodes),
						func() {
							app.QueueUpdateDraw(func() {
								app.Nodes()
							})
						},
					)
				case "sjs", Subjects:
					if !app.isSchemaRegistrySelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select Schema Registry"
						continue
					}
					app.CheckInCache(Subjects, func() {
						app.Subjects()
					})
				case "q!":
					app.Stop()
				default:
					statusLineCh <- "invalid command"
				}
			}
		}
	}()
}

func (app *App) RunStatusLineHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down StatusLineHandler")
				return
			case status := <-in:
				app.QueueUpdateDraw(func() {
					app.Layout.StatusLine.SetText(status)
				})
			}
		}
	}()
}

func (app *App) Run() {
	app.ApplyColors()
	ctx, cancel := context.WithCancel(context.Background())
	app.RunCommandHandler(ctx, commandCh)
	app.RunStatusLineHandler(ctx, statusLineCh)

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

	ct := app.NewClustersTable()
	st := app.NewSchemaRegistriesTable()
	app.SchemaRegistriesTableInputHandler(st)
	app.ClustersTableInputHandler(ct)
	app.Layout.SetSelected(app.Selected.Cluster, app.Selected.SchemaRegistry)

	app.Layout.PagesRegistry.PageMenuMap[Clusters] = ClustersPageMenu
	app.Layout.PagesRegistry.PageMenuMap[SchemaRegistries] = SubjectsPageMenu
	app.Layout.PagesRegistry.PageMenuMap[Resources] = ResourcesPageMenu
	app.Layout.PagesRegistry.PageMenuMap[OpenedPages] = OpenedPagesMenu

	resourcesPage := app.Layout.PagesRegistry.NewResourcesPage(app, commandCh, app.Colors)
	app.Layout.PagesRegistry.UI.Pages.AddPage(Clusters, ct, true, false)
	app.Layout.PagesRegistry.UI.Pages.AddPage(SchemaRegistries, st, true, false)
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

	app.ClustersTableInputHandler(ct)
	app.SchemaRegistriesTableInputHandler(st)
	app.OpenPagesKeyHadler(app.Layout.PagesRegistry.UI.OpenedPages)
	app.SearchKeyHadler(app.Layout.Search)
	app.MainOperationKeyHadler()

	err := app.SetRoot(app.Layout.Content, true).Run()
	if err != nil {
		log.Error().Err(err).Msg("Failed Application execution")
	}
	cancel()
	log.Info().Msg("Application terminated")
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

func (app *App) Cluster() {
	c := app.KafkaClients[app.Selected.Cluster.Name]
	rCh := make(chan *client.ClusterResult)
	errorCh := make(chan error)
	c.DescribeCluster(rCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-rCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(fmt.Sprintf(" %s ", description.Name))
					desc.SetText(description.String())

					app.AddToPagesRegistry(Cluster, desc, ClustersPageMenu)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe cluster")
				statusLineCh <- fmt.Sprintf("[red]failed to describe cluster: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing cluster")
				statusLineCh <- "[red]timeout while describing cluster"
				return
			}
		}
	}()
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

func (app *App) NewClustersTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Clusters ")
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

	row := 0
	for _, cluster := range app.Clusters {
		table.
			SetCell(row, 0, tview.NewTableCell(cluster.Name)).
			SetCell(row, 1, tview.NewTableCell(cluster.Properties["bootstrap.servers"]))
		row++
	}
	return table
}

func (app *App) NewSchemaRegistriesTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Schema Registry URLs ")
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

	row := 0
	for _, sr := range app.SchemaRegistries {
		table.
			SetCell(row, 0, tview.NewTableCell(sr.Name)).
			SetCell(row, 1, tview.NewTableCell(sr.SchemaRegistryUrl))
		row++
	}
	return table
}

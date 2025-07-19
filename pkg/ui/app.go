package ui

import (
	"cinnamon/pkg/client"
	"cinnamon/pkg/config"
	"cinnamon/pkg/schemaregistry"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const timeout = time.Second * 10
const (
	Main             = "Main"
	Resources        = "Resources"
	Clusters         = "Clusters"
	Cluster          = "Cluster"
	SchemaRegistries = "Schema-registries"
	Topics           = "Topics"
	Topic            = "Topic"
	Nodes            = "Nodes"
	Node             = "Node"
	ConsumerGroups   = "Consumer-groups"
	ConsumerGroup    = "Consumer-group"
	Subjects         = "Subjects"
	Opened           = "Opened"
	ConsumingParams  = "Consuming Parameters"
)

type App struct {
	*tview.Application
	Main                  *MainPage
	Cache                 *cache.Cache
	Clusters              map[string]*config.ClusterConfig
	SchemaRegistries      map[string]*config.SchemaRegistryConfig
	KafkaClients          map[string]*client.Client
	SchemaRegistryClients map[string]*schemaregistry.Client
	Selected              Selected
	Opened                *OpenedPages
}

type Selected struct {
	Cluster        *config.ClusterConfig
	SchemaRegistry *config.SchemaRegistryConfig
}

type KeySeries struct {
	Series map[int]string
}

func toClustersMap(cfg *config.Config) map[string]*config.ClusterConfig {
	clusterMap := make(map[string]*config.ClusterConfig)
	for _, cluster := range cfg.Cinnamon.Clusters {
		clusterMap[cluster.Name] = cluster
	}
	return clusterMap
}

func toSchemaRegistryMap(cfg *config.Config) map[string]*config.SchemaRegistryConfig {
	srMap := make(map[string]*config.SchemaRegistryConfig)
	for _, sr := range cfg.Cinnamon.SchemaRegistries {
		srMap[sr.Name] = sr
	}
	return srMap
}

func (app *App) getCurrentKafkaClient() *client.Client {
	return app.KafkaClients[app.Selected.Cluster.Name]
}

func (app *App) getCurrentSchemaRegistryClient() *schemaregistry.Client {
	return app.SchemaRegistryClients[app.Selected.SchemaRegistry.Name]
}

func NewApp() *App {
	initLogger()

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize config")
		os.Exit(1)
	}
	return &App{
		Application:           tview.NewApplication(),
		Main:                  NewPage(),
		Cache:                 cache.New(5*time.Minute, 10*time.Minute),
		Clusters:              toClustersMap(cfg),
		SchemaRegistries:      toSchemaRegistryMap(cfg),
		KafkaClients:          make(map[string]*client.Client),
		SchemaRegistryClients: make(map[string]*schemaregistry.Client),
	}
}

func initLogger() {
	zerolog.TimeFieldFormat = time.RFC3339

	logFilePath := filepath.Join(os.Getenv("HOME"), ".cinnamon", "cinnamon.log")
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
	statusLineChannel = make(chan string, 10)
	commandChannel    = make(chan string)
)

func (app *App) CommandHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down CommandHandler")
				return
			case command := <-in:
				switch command {
				case Main:
					app.Main.Pages.SwitchToPage(Main)
				case Clusters:
					app.Main.Pages.SwitchToPage(Clusters)
				case SchemaRegistries:
					app.Main.Pages.SwitchToPage(SchemaRegistries)
				case "tps", Topics:
					if !app.isClusterSelected(app.Selected) {
						statusLineChannel <- "[red]To perform operation, select Cluster"
						continue
					}
					app.Check(fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Topics), func() {
						app.Topics(statusLineChannel)
					})
				case "grs", ConsumerGroups:
					if !app.isClusterSelected(app.Selected) {
						statusLineChannel <- "[red]To perform operation, select Cluster"
						continue
					}
					app.Check(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, ConsumerGroups),
						func() {
							app.ConsumerGroups(statusLineChannel)
						},
					)
				case "nds", Nodes:
					if !app.isClusterSelected(app.Selected) {
						statusLineChannel <- "[red]To perform operation, select Cluster"
						continue
					}
					app.Check(fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Nodes), func() {
						app.QueueUpdateDraw(func() {
							app.Nodes(statusLineChannel)
						})
					})
				case "sjs", Subjects:
					if !app.isClusterSelected(app.Selected) {
						statusLineChannel <- "[red]To perform operation, select Schema Registry"
						continue
					}
					app.Check(Subjects, func() {
						app.Subjects(statusLineChannel)
					})
				case "q!":
					app.Stop()
				default:
					statusLineChannel <- "Invalid command"
				}
			}
		}
	}()
}

func (app *App) StatusLineHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down StatusLineHandler")
				return
			case status := <-in:
				app.QueueUpdateDraw(func() {
					app.Main.StatusLine.SetText(status)
				})
			}
		}
	}()
}

func (app *App) Init() {
	ctx, cancel := context.WithCancel(context.Background())
	app.CommandHandler(ctx, commandChannel)
	app.StatusLineHandler(ctx, statusLineChannel)

	app.Opened = app.NewOpenedPages()

	ct := app.NewClustersTable()
	st := app.NewSchemaRegistriesTable()
	app.SchemaRegistriesTableInputHandler(st)
	app.ClustersTableInputHandler(ct)

	main := tview.NewTable()
	main.SetTitle(" Main ")
	main.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	r := 0
	var configs []*config.ClusterConfig
	for _, cluster := range app.Clusters {
		configs = append(configs, cluster)
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})
	for _, cluster := range configs {
		main.
			SetCell(r, 0, tview.NewTableCell(cluster.Name)).
			SetCell(r, 1, tview.NewTableCell(cluster.Properties["bootstrap.servers"])).
			SetCell(r, 2, tview.NewTableCell(cluster.SchemaRegistry)).
			SetCell(r, 3, tview.NewTableCell(app.SchemaRegistries[cluster.SchemaRegistry].SchemaRegistryUrl))
		r++
	}

	main.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := main.GetSelection()
		clusterName := main.GetCell(row, 0).Text
		schemaRegistryName := main.GetCell(row, 2).Text
		cluster := app.Clusters[clusterName]
		schemaRegistry := app.SchemaRegistries[schemaRegistryName]

		if event.Key() == tcell.KeyEnter {
			app.SelectCluster(cluster)
			app.SelectSchemaRegistry(schemaRegistry)
			app.Main.SetSelected(app.Selected.Cluster.Name, app.Selected.SchemaRegistry.Name)
			app.Main.ClearStatus()
		}
		return event
	})

	app.Main.Pages.AddPage(Resources, NewResourcesPage(commandChannel).Modal, true, true)
	app.Main.Pages.AddPage(Opened, app.Opened.Modal, true, true)
	app.Main.Pages.AddPage(Clusters, ct, true, true)
	app.Main.Pages.AddPage(SchemaRegistries, st, true, true)

	app.AddAndSwitch(Main, main, MainPageMenu)

	app.Main.Search.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyTab {
			app.Main.Bottom.SwitchToPage("menu")
			app.Application.SetFocus(app.Main.Pages)
		}

		if event.Key() == tcell.KeyEsc {
			app.Main.Search.SetText("")
			app.Main.Bottom.SwitchToPage("menu")
			app.Application.SetFocus(app.Main.Pages)
		}
		return event
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			app.Main.Menu.SetMenu(ResourcesPageMenu)
			app.Main.Pages.SwitchToPage(Resources)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			app.Main.Bottom.SwitchToPage("search")
			app.SetFocus(app.Main.Search)
			app.Main.ClearStatus()
			return nil
		}

		if event.Key() == tcell.KeyCtrlP {
			app.Main.Menu.SetMenu(ResourcesPageMenu)
			app.Main.Pages.SwitchToPage(Opened)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'b' && !app.Main.Search.HasFocus() {
			if app.Opened.ActivePage > 0 {
				app.Opened.ActivePage--
				app.NavigateTo(app.Opened.ActivePage)
			}
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'f' && !app.Main.Search.HasFocus() {
			if app.Opened.ActivePage < app.Opened.Table.GetRowCount()-1 {
				app.Opened.ActivePage++
				app.NavigateTo(app.Opened.ActivePage)
			}
		}

		return event
	})

	err := app.SetRoot(app.Main.Content, true).Run()
	if err != nil {
		log.Error().Err(err).Msg("Failed Application execution")
	}
	cancel()
	log.Info().Msg("Application terminated")
}

func (app *App) ClustersTableInputHandler(ct *tview.Table) {
	ct.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := ct.GetSelection()
		clusterName := ct.GetCell(row, 0).Text
		cluster := app.Clusters[clusterName]

		if event.Key() == tcell.KeyEnter {
			app.SelectCluster(cluster)
			app.Main.ClearStatus()
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
			if !app.isClusterSelected(app.Selected) {
				app.SelectCluster(cluster)
			}
			statusLineChannel <- "Getting cluster description results..."
			app.Cluster()
		}

		return event
	})
}

func (app *App) isClusterSelected(selected Selected) bool {
	return selected.Cluster != nil
}

func (app *App) SelectCluster(cluster *config.ClusterConfig) {
	app.Selected.Cluster = cluster

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

func (app *App) SelectSchemaRegistry(sr *config.SchemaRegistryConfig) {
	app.Selected.SchemaRegistry = sr

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

func (app *App) SchemaRegistriesTableInputHandler(st *tview.Table) {
	st.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := st.GetSelection()
		name := st.GetCell(row, 0).Text
		sr := app.SchemaRegistries[name]

		if event.Key() == tcell.KeyEnter {
			app.SelectSchemaRegistry(sr)
			app.Main.ClearStatus()
		}

		return event
	})
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

					app.AddAndSwitch(Cluster, desc, ClustersPageMenu)
					app.Main.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe cluster")
				statusLineChannel <- fmt.Sprintf("[red]Failed to describe cluster: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing cluster")
				statusLineChannel <- "[red]Timeout while describing cluster"
				return
			}
		}
	}()
}

func (app *App) NewMainTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Main ")
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	row := 0
	for _, cluster := range app.Clusters {
		table.
			SetCell(row, 0, tview.NewTableCell(cluster.Name)).
			SetCell(row, 1, tview.NewTableCell(cluster.Properties["bootstrap.servers"]))
		row++
	}
	return table
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
	return desc
}

func (app *App) NewClustersTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Clusters ")
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

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

	row := 0
	for _, sr := range app.SchemaRegistries {
		table.
			SetCell(row, 0, tview.NewTableCell(sr.Name)).
			SetCell(row, 1, tview.NewTableCell(sr.SchemaRegistryUrl))
		row++
	}
	return table
}

// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package connect provides a client wrapper for Kafka Connect REST API operations.
package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

// Client wraps the Kafka Connect REST API client with cluster name context.
type Client struct {
	Name       string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Kafka Connect client with the given configuration.
func NewClient(cfg *config.ConnectConfig, timeout time.Duration) (*Client, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	if cfg.Username != "" && cfg.Password != "" {
		client.Transport = &basicAuthTransport{
			Username:  cfg.Username,
			Password:  cfg.Password,
			Transport: http.DefaultTransport,
		}
	}

	return &Client{
		Name:       cfg.Name,
		BaseURL:    cfg.URL,
		HTTPClient: client,
	}, nil
}

// basicAuthTransport adds basic authentication to HTTP requests.
type basicAuthTransport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return t.transport().RoundTrip(req)
}

func (t *basicAuthTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// ListConnectors retrieves all connector names from the Kafka Connect cluster.
func (c *Client) ListConnectors(resultChan chan<- []string, errorChan chan<- error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.HTTPClient.Timeout)
		defer cancel()

		connectors, err := c.listConnectors(ctx)
		if err != nil {
			errorChan <- fmt.Errorf("failed to list connectors: %w", err)
			return
		}

		resultChan <- connectors
	}()
}

// GetConnectorStatus retrieves the status of a specific connector.
func (c *Client) GetConnectorStatus(name string, resultChan chan<- *ConnectorStatus, errorChan chan<- error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.HTTPClient.Timeout)
		defer cancel()

		status, err := c.getConnectorStatus(ctx, name)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get connector status: %w", err)
			return
		}

		resultChan <- status
	}()
}

// GetConnectorConfig retrieves the configuration of a specific connector.
func (c *Client) GetConnectorConfig(name string, resultChan chan<- map[string]interface{}, errorChan chan<- error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.HTTPClient.Timeout)
		defer cancel()

		config, err := c.getConnectorConfig(ctx, name)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get connector config: %w", err)
			return
		}

		resultChan <- config
	}()
}

// DescribeConnector fetches both status and config for a connector in parallel.
// Results are sent via resultChan; any errors are sent via errorChan.
func (c *Client) DescribeConnector(name string, resultChan chan<- *ConnectorDetail, errorChan chan<- error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.HTTPClient.Timeout)
		defer cancel()

		detail := &ConnectorDetail{Name: name}
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			status, err := c.getConnectorStatus(ctx, name)
			if err != nil {
				errorChan <- fmt.Errorf("failed to get connector status: %w", err)
				return
			}
			detail.Status = status
		}()

		go func() {
			defer wg.Done()
			config, err := c.getConnectorConfig(ctx, name)
			if err != nil {
				errorChan <- fmt.Errorf("failed to get connector config: %w", err)
				return
			}
			detail.Config = config
		}()

		wg.Wait()
		resultChan <- detail
	}()
}

func (c *Client) listConnectors(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/connectors", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var connectors []string
	if err := json.NewDecoder(resp.Body).Decode(&connectors); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return connectors, nil
}

func (c *Client) getConnectorStatus(ctx context.Context, name string) (*ConnectorStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/connectors/"+name+"/status", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var rawStatus connectorStatusRaw
	if err := json.NewDecoder(resp.Body).Decode(&rawStatus); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	status := &ConnectorStatus{
		Name: name,
		Connector: ConnectorStateInfo{
			State:    rawStatus.Connector.State,
			WorkerID: rawStatus.Connector.WorkerID,
		},
		Type: rawStatus.Type,
	}

	for _, t := range rawStatus.Tasks {
		status.Tasks = append(status.Tasks, TaskStateInfo{
			ID:       t.ID,
			State:    t.State,
			WorkerID: t.WorkerID,
			Trace:    t.Trace,
		})
	}

	return status, nil
}

func (c *Client) getConnectorConfig(ctx context.Context, name string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/connectors/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Name   string                 `json:"name"`
		Config map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	log.Debug().Str("connector", name).Msg("fetched connector config")
	return response.Config, nil
}

// connectorStatusRaw matches the raw JSON response from /connectors/{name}/status.
type connectorStatusRaw struct {
	Name      string `json:"name"`
	Connector struct {
		State    string `json:"state"`
		WorkerID string `json:"worker_id"`
	} `json:"connector"`
	Tasks []struct {
		ID       int    `json:"id"`
		State    string `json:"state"`
		WorkerID string `json:"worker_id"`
		Trace    string `json:"trace"`
	} `json:"tasks"`
	Type string `json:"type"`
}

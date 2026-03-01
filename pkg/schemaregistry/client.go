// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package schemaregistry provides a client wrapper for Confluent Schema Registry operations.
package schemaregistry

import (
	"sort"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

// Client wraps the Schema Registry client with cluster name context.
type Client struct {
	ClusterName string
	schemaregistry.Client
}

// SchemaResult contains the schema metadata.
type SchemaResult struct {
	Metadata schemaregistry.SchemaMetadata
}

// NewSchemaRegistryClient creates a new Schema Registry client with the given configuration.
func NewSchemaRegistryClient(config *config.SchemaRegistryConfig) (*Client, error) {
	client, err := schemaregistry.NewClient(schemaregistry.NewConfigWithBasicAuthentication(
		config.SchemaRegistryURL,
		config.SchemaRegistryUsername,
		config.SchemaRegistryPassword))
	if err != nil {
		log.Err(err).Msg("failed to connect to schema registry")
		return nil, err
	}

	return &Client{config.Name, client}, nil
}

func (client *Client) DescribeSchemaRegistry(resultChan chan<- []string, errorChan chan<- error) {
	go func() {
		subjects, err := client.GetAllSubjects()
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- subjects
	}()
}

// Subjects retrieves all schema subjects from the Schema Registry.
func (client *Client) Subjects(resultChan chan<- []string, errorChan chan<- error) {
	go func() {
		subjects, err := client.GetAllSubjects()
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- subjects
	}()
}

// VersionsBySubject retrieves all versions for a specific subject.
func (client *Client) VersionsBySubject(
	subject string,
	resultChan chan<- []int,
	errorChan chan<- error,
) {
	go func() {
		versions, err := client.GetAllVersions(subject)
		if err != nil {
			errorChan <- err
			return
		}
		sort.Sort(sort.Reverse(sort.IntSlice(versions)))
		resultChan <- versions
	}()
}

// Schema retrieves a specific schema version for a subject.
func (client *Client) Schema(
	subject string,
	version int,
	resultChan chan<- SchemaResult,
	errorChan chan<- error,
) {
	go func() {
		metadata, err := client.GetSchemaMetadata(subject, version)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- SchemaResult{metadata}
	}()
}

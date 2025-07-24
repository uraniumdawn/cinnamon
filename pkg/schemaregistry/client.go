// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package schemaregistry

import (
	"cinnamon/pkg/config"
	"fmt"
	"sort"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
)

type Client struct {
	ClusterName string
	schemaregistry.Client
}

type SchemaResult struct {
	Metadata schemaregistry.SchemaMetadata
}

func NewSchemaRegistryClient(config *config.SchemaRegistryConfig) (*Client, error) {
	client, err := schemaregistry.NewClient(schemaregistry.NewConfigWithBasicAuthentication(
		config.SchemaRegistryUrl,
		config.SchemaRegistryUsername,
		config.SchemaRegistryPassword))
	if err != nil {
		fmt.Printf("Failed to create schema registry client: %s\n", err)
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
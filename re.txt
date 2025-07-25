# Cinnamon

[![Go Report Card](https://goreportcard.com/badge/github.com/petrovsky-s/cinnamon)](https://goreportcard.com/report/github.com/petrovsky-s/cinnamon)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**A TUI for Apache Kafka and Schema Registry.**

Cinnamon provides a simple and efficient terminal-based user interface for managing and observing Apache Kafka clusters and Confluent Schema Registries. Inspired by tools like `k9s` and `lazygit`, Cinnamon aims to make Kafka development and administration faster and more intuitive, without leaving the comfort of your terminal.

<!-- TODO: Add a GIF of Cinnamon in action here -->
<!-- ![Cinnamon Demo](./assets/demo.gif) -->

## What is it?

Cinnamon is a dedicated terminal client for Apache Kafka that allows you to:
- **Navigate** your Kafka clusters and Schema Registries.
- **View** topics, consumer groups, cluster nodes, and schema subjects.
- **Describe** resources to see detailed configurations and metadata.
- **Search** and filter resources with fuzzy search.
<!-- - **Consume** messages from your topics directly in the TUI. -->

It's designed for developers and administrators who work with Kafka and prefer keyboard-driven workflows and the speed of a terminal interface.

## Features

- **Cluster Management**:
  - View and switch between multiple configured Kafka clusters.
  - See cluster overview, including broker nodes.
- **Topic Management**:
  - List all topics with partition and replication factor counts.
  - Describe topic configurations and partition details.
  - Fuzzy search to quickly find topics.
- **Consumer Group Management**:
  - List all consumer groups and their states.
  - Describe consumer groups to see members and offset details.
- **Schema Registry Integration**:
  - List all subjects in your Schema Registry.
  - View all registered versions for a subject.
  - Inspect the schema for a specific version.
<!-- - **Message Consuming**:
  - Built-in consumer to view topic messages.
  - Customizable consuming parameters (latest N records, offset, timestamp).
  - Support for Avro-encoded messages with automatic deserialization. -->

## Installation

### From Source

Ensure you have Go installed (version 1.19+).

```shell
git clone https://github.com/uraniumdawn/cinnamon.git
cd cinnamon
make install
```

This will build the `cinnamon` binary and install it in your `$GOPATH/bin` directory.

## Configuration

Cinnamon requires a configuration file to connect to your Kafka clusters and Schema Registries.

By default, Cinnamon looks for `config.yaml` in `~/.cinnamon/`. You can specify a different directory by setting the `CINNAMON_CONFIG_DIR` environment variable.

**Example `config.yaml`:**

```yaml
cinnamon:
  clusters:
    - name: "dev-cluster"
      properties:
        bootstrap.servers: "kafka-dev:9092"
        # Add any other kafka client properties here
        # security.protocol: "SASL_SSL"
        # sasl.mechanisms: "PLAIN"
        # sasl.username: "$USER"
        # sasl.password: "$PASSWORD"
      schema.registry.name: "dev-sr"

    - name: "prod-cluster"
      properties:
        bootstrap.servers: "kafka-prod:9092"
      schema.registry.name: "prod-sr"

  schema-registries:
    - name: "dev-sr"
      schema.registry.url: "http://schema-registry-dev:8081"
      # Optional: for basic auth
      # schema.registry.sasl.username: "sr-user"
      # schema.registry.sasl.password: "sr-password"

    - name: "prod-sr"
      schema.registry.url: "http://schema-registry-prod:8081"
```

Environment variables in the format `${VAR_NAME}` or `$VAR_NAME` will be expanded.

## Usage

Simply run the application from your terminal:
```shell
cinnamon
```

### Keybindings

Cinnamon is controlled through keyboard shortcuts. The available actions are always displayed in the footer menu.

#### Global Navigation
| Key         | Action                               |
|-------------|--------------------------------------|
| `<j>`, `↓`    | Navigate down                        |
| `<k>`, `↑`    | Navigate up                          |
| `<Enter>`   | Select item                          |
| `<b>`         | Go back to the previous view         |
| `<f>`         | Go forward to the next view          |
| `<:>`         | Open the Resources selection modal   |
| `</>`         | Focus the search/filter input        |
| `<Ctrl+p>`  | Show list of all opened pages/views  |
| `<Esc>`     | Close modal/dialog or clear search   |
| `q`         | Quit the application                 |

#### Resource-Specific Actions
| Key         | Action                               | Context              |
|-------------|--------------------------------------|----------------------|
| `<d>`         | Describe the selected resource       | Most resource lists  |
| `<Ctrl+u>`  | Refresh/update the current view      | Most resource lists  |

<!-- ## Contributing

Contributions are welcome! Please feel free to open an issue or submit a pull request. -->

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

# Cinnamon 🎯

A powerful Terminal UI (TUI) for Apache Kafka that provides an intuitive, keyboard-driven interface for managing and monitoring your Kafka clusters, topics, consumer groups, and Schema Registry.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)

## Table of Contents

- [Core Capabilities](#core-capabilities)
- [Available Resources](#available-resources)
- [Installation](#installation)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Development](#development)
- [License](#license)
- [Acknowledgments](#acknowledgments)
- [Support](#support)


## Core Capabilities

- **Multi-Cluster Management** - Connect and switch between multiple Kafka clusters seamlessly
- **Topics Management** - Browse, create, edit, and delete Kafka topics with full configuration support
- **Consumer Groups** - Monitor consumer groups, view lag, partition assignments, and member details
- **Schema Registry Integration** - Browse subjects, view schema versions, and inspect Avro schemas
- **Broker & Node Management** - View cluster node information, configurations, and health status
- **CLI Command Templates** - Execute external tools (kcat, kafka-console-consumer) with auto-filled parameters


## Available Resources

Access via `:` (colon) key:

| Resource | Description | Operations |
|----------|-------------|------------|
| **Clusters** | Kafka cluster management | Select, describe, view brokers |
| **Schema-registries** | Schema Registry instances | Select, browse subjects |
| **Topics** | Kafka topics | List, create, edit, delete, describe, search |
| **Consumer groups** | Consumer groups | List, describe, view lag, search |
| **Nodes** | Kafka brokers | List, view configuration |
| **Subjects** | Schema Registry subjects | List, view versions, inspect schemas, search |

## Installation

### Dependencies

**Required System Library:**
- `librdkafka` - Must be installed on your system
    - macOS: `brew install librdkafka`
    - Ubuntu/Debian: `apt-get install librdkafka-dev`
    - RHEL/CentOS: `yum install librdkafka-devel`


### Homebrew (macOS)

```bash
# Add the tap
brew tap uraniumdawn/cinnamon

# Install cinnamon
brew install cinnamon
```

### From Source

```bash
# Clone the repository
git clone https://github.com/uraniumdawn/cinnamon.git
cd cinnamon

# Build
go build -o cinnamon

# Move to your PATH
mv cinnamon /usr/local/bin/
```

## Getting Started

### 1. Create Configuration Files

Cinnamon requires at least one configuration file:

- `~/.config/cinnamon/config.yaml` - Application and cluster configuration (required)
- `~/.config/cinnamon/style.yaml` - UI color customization (optional)

### 2. Run Cinnamon

```bash
cinnamon
```

To check the version:

```bash
cinnamon --version
```

## Configuration

### config.yaml

Create `~/.config/cinnamon/config.yaml` with your Kafka cluster and Schema Registry configurations:

```yaml
cinnamon:
  # API Configuration
  api:
    timeout: 10  # API call timeout in seconds (default: 10)

  # Define your Kafka clusters
  clusters:
    - name: prod
      # Standard librdkafka configuration properties as documented in:
      # https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md
      properties:
        bootstrap.servers: kafka-prod:9094
        # Add any librdkafka properties here:
        # security.protocol: SASL_SSL
        # sasl.mechanisms: PLAIN
        # sasl.username: your-username
        # sasl.password: your-password
      selected: true  # Auto-select this cluster on startup

    - name: dev
      properties:
        bootstrap.servers: kafka-dev:29094
      selected: false

  # Schema Registry configurations (optional)
  schema-registries:
    - name: prod
      # Required: Schema Registry URL
      schema.registry.url: http://schema-registry-prod:8081
      
      # Optional: Basic authentication for Schema Registry
      # schema.registry.sasl.username: registry-user
      # schema.registry.sasl.password: registry-pass
      selected: true  # Auto-select this registry on startup
    
    - name: dev
      schema.registry.url: http://schema-registry-dev:8081
      selected: false
    
  # CLI Templates for external tool integration (optional)
  # Use placeholders: {{bootstrap}} for broker address, {{topic}} for topic name
  cli_templates:
    # kcat example - consume from beginning with JSON formatting
    - kcat -b {{bootstrap}} -t {{topic}} -o beginning -f '{"Key":"%k","Value":%s,"Timestamp":%T,"Partition":%p,"Offset":%o,"Headers":"%h","Size":%S}\n' -u | jq .
    
    # kcat example - consume from end (live)
    - kcat -b {{bootstrap}} -t {{topic}}
    
    # kafka-console-consumer
    - kafka-console-consumer --bootstrap-server {{bootstrap}} --topic {{topic}} --from-beginning
    
    # Custom script example
    - ./scripts/analyze-topic.sh {{bootstrap}} {{topic}}
```

#### Important Configuration Notes

**librdkafka Properties:**
All properties in the `properties` map are passed directly to librdkafka. Cinnamon supports:
- Connection settings (bootstrap.servers, security.protocol)
- Authentication (SASL, SSL, OAuth)
- Client configuration (request.timeout.ms, retry settings)
- Debug settings (debug: all)

**Environment Variables:**
Use `${VAR_NAME}` syntax for environment variable expansion:
```yaml
sasl.password: ${KAFKA_PASSWORD}
```

**Selected Flag:**
- Only one cluster and one schema registry should have `selected: true`
- Selection is persisted when changed via UI

**API Timeout:**
- Controls timeout for all Kafka Admin API calls
- Affects cluster describe, topic operations, consumer group queries
- Default: 10 seconds if not specified

### style.yaml

Create `~/.config/cinnamon/style.yaml` to customize the UI colors (optional):

```yaml
# Color configuration for Cinnamon UI
# You can use tcell color names or RGB hex values (e.g., "#ffffff")
# Available color names: black, white, red, green, blue, yellow, orange, purple, 
# pink, grey, brown, beige, cyan, etc.
# Use "default" to inherit your terminal's default colors

cinnamon:
  # Cluster selector component colors
  cluster:
    # Text color for cluster names
    fgColor: "white"
    # Background color for cluster selector
    bgColor: "black"

  # Status bar colors (bottom bar showing current context)
  status:
    # Status bar text color
    fgColor: "grey"
    # Status bar background color
    bgColor: "black"

  # Label colors (field labels, form labels)
  label:
    # Label text color
    fgColor: "orange"
    # Label background color
    bgColor: "black"

  # Keybinding hints display
  keybinding:
    # Color for keyboard shortcut keys (e.g., ":", "Enter")
    key: "grey"
    # Color for keybinding descriptions
    value: "grey"

  # Selected/highlighted item colors
  selection:
    # Text color for selected items in lists
    fgColor: "black"
    # Background color for selected items
    bgColor: "white"

  # Placeholder text color (empty states, input hints)
  placeholder: "grey"

  # Title text color (page titles, modal headers)
  title: "orange"

  # Border color for panels and modals
  border: "white"

  # Global background color (main application background)
  background: "black"

  # Global foreground color (main application text)
  foreground: "white"
```

#### Alternative Color Schemes

**Dark Theme with Blue Accents:**
```yaml
cinnamon:
  cluster:
    fgColor: "#4A9EFF"
    bgColor: "#1E1E1E"
  selection:
    fgColor: "#1E1E1E"
    bgColor: "#4A9EFF"
  title: "#4A9EFF"
  border: "#4A9EFF"
  background: "#1E1E1E"
  foreground: "#D4D4D4"
```

**Solarized Dark:**
```yaml
cinnamon:
  cluster:
    fgColor: "#93A1A1"
    bgColor: "#002B36"
  selection:
    fgColor: "#002B36"
    bgColor: "#268BD2"
  title: "#B58900"
  border: "#586E75"
  background: "#002B36"
  foreground: "#839496"
```

**Gruvbox Dark:**
```yaml
cinnamon:
  cluster:
    fgColor: "#EBDBB2"
    bgColor: "#282828"
  status:
    fgColor: "#928374"
    bgColor: "#282828"
  label:
    fgColor: "#FE8019"
    bgColor: "#282828"
  keybinding:
    key: "#928374"
    value: "#A89984"
  selection:
    fgColor: "#282828"
    bgColor: "#FABD2F"
  placeholder: "#928374"
  title: "#FE8019"
  border: "#EBDBB2"
  background: "#282828"
  foreground: "#EBDBB2"
```

## Development

### Prerequisites

- Go 1.21 or higher
- librdkafka (for Kafka client library)
- A running Kafka cluster for testing

### Building from Source

```bash
# Clone repository
git clone https://github.com/uraniumdawn/cinnamon.git
cd cinnamon

# Install dependencies
go mod download

# Build
go build -o cinnamon

# Run
./cinnamon
```

### Technical Architecture

**Event-Driven Design:**
- Each resource type has a dedicated event channel
- Event handlers run in separate goroutines
- Non-blocking operations for responsive UI
- Timeout contexts prevent hanging operations

### Logging

All application logs are written to:
```
~/.config/cinnamon/cinnamon.log
```

Log format: RFC3339 timestamp with caller information (file:line)

**Log Levels:**
- `INFO` - Normal operations (startup, shutdown)
- `DEBUG` - Event handler lifecycle
- `ERROR` - Failed operations, API errors, timeouts

**Useful for debugging:**
```bash
# Watch logs in real-time
tail -f ~/.config/cinnamon/cinnamon.log

# Filter by level
grep ERROR ~/.config/cinnamon/cinnamon.log
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

**Built with excellent open-source libraries:**

- **[tview](https://github.com/rivo/tview)** - Powerful terminal UI framework with rich widgets
- **[tcell](https://github.com/gdamore/tcell)** - Low-level terminal handling and colors
- **[confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go)** - Official Kafka Go client
- **[librdkafka](https://github.com/confluentinc/librdkafka)** - High-performance C library for Kafka
- **[go-cache](https://github.com/patrickmn/go-cache)** - In-memory caching with expiration
- **[fuzzysearch](https://github.com/lithammer/fuzzysearch)** - Fuzzy string matching for search
- **[zerolog](https://github.com/rs/zerolog)** - Fast, structured logging

**Inspiration:**
- `k9s` - Kubernetes terminal UI
- `lazydocker` - Docker terminal UI  



## Support

If you encounter any issues or have questions:
- [Report a Bug](https://github.com/uraniumdawn/cinnamon/issues)
- [Request a Feature](https://github.com/uraniumdawn/cinnamon/issues)
- Contact: sirozhaua@gmail.com

---

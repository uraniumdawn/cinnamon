# Cinnamon 🎯

A Terminal UI (TUI) for Apache Kafka that provides an intuitive interface for managing and monitoring your Kafka clusters, topics, consumer groups, and Schema Registry.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)

## Features

- **Multi-Cluster Management** - Connect and switch between multiple Kafka clusters seamlessly
- **Topics Management** - Browse, create, edit, and delete Kafka topics with detailed metadata
- **Real-time Consuming** - Consume messages from topics with Avro schema support
- **Nodes & Brokers** - View cluster node information and health status
- **Schema Registry Integration** - Browse and manage Avro schemas and subjects
- **Customizable UI** - Fully customizable color scheme to match your terminal preferences
- **CLI Templates** - Pre-configured command templates for quick external tool integration

## Installation

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
  # Define your Kafka clusters
  clusters:
    - name: prod
      # Standard librdkafka configuration properties as documented in:
      # https://github.com/confluentinc/librdkafka/tree/master/CONFIGURATION.md
      properties:
        bootstrap.servers: kafka-prod:9094

  # Schema Registry configurations (optional)
  schema-registries:
    - name: prod
      # Required: Schema Registry URL
      schema.registry.url: http://schema-registry-prod:8081
      
      # Optional: Basic authentication for Schema Registry
      # schema.registry.sasl.username: registry-user
      # schema.registry.sasl.password: registry-pass
    
  # CLI Templates for external tool integration (optional)
  # Use placeholders: {{bootstrap}} for broker address, {{topic}} for topic name
  cli_templates:
    # kcat example - consume from beginning with JSON formatting
    - kcat -b {{bootstrap}} -t {{topic}} -o beginning -f '{"Key":"%k","Value":%s,"Timestamp":%T,"Partition":%p,"Offset":%o,"Headers":"%h","Size":%S}\n' -u | jq .
    
    # kcat example - consume from end (live)
    - kcat -b {{bootstrap}} -t {{topic}}
    
    # Custom script example
    - ./scripts/analyze-topic.sh {{bootstrap}} {{topic}}
```

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

### Key Features

#### Cluster Management
- Switch between configured clusters from the main menu
- View cluster metadata and broker information

#### Topics
- Browse all topics in the selected cluster
- View topic details: partitions, replication factor, configuration
- Create new topics with custom configurations
- Edit existing topic configurations
- Delete topics (with confirmation)

#### Consumer Groups
- List all consumer groups
- View consumer group details and member information
- Monitor offset lag and partition assignments

#### Schema Registry
- Browse subjects in the Schema Registry
- View schema versions and details
- Inspect Avro schemas

#### CLI Templates
- Quick access to pre-configured command templates
- Automatically substitutes cluster and topic information
- Copy generated commands to clipboard for external tool usage

## Development

### Prerequisites

- Go 1.21 or higher
- librdkafka (for Kafka client)

### Building from Source

```bash
# Clone repository
git clone https://github.com/uraniumdawn/cinnamon.git
cd cinnamon

# Install dependencies
go mod download

# Build
go build -o cinnamon
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [tview](https://github.com/rivo/tview) - Terminal UI library
- Uses [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go) - Kafka client
- Powered by [librdkafka](https://github.com/edenhill/librdkafka)

## Support

If you encounter any issues or have questions:
- [Report a Bug](https://github.com/uraniumdawn/cinnamon/issues)
- [Request a Feature](https://github.com/uraniumdawn/cinnamon/issues)
- Contact: sirozhaua@gmail.com

---

Made with ❤️ by [Sergey Petrovsky](https://github.com/uraniumdawn)


# FT_0

FT_0 is a modern terminal-based file transfer application built in Go using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. It provides a seamless way to transfer files between computers through a terminal interface.

## Features ✨

- Beautiful Terminal UI powered by Bubble Tea
- Three operation modes:
  - Send: Direct file transfer to receivers
  - Receive: Accept incoming file transfers
  - Relay: Act as an intermediary server
- Real-time progress monitoring with speed and completion status
- Session-based transfers with unique IDs for security
- Cross-platform compatibility (Windows, macOS, Linux)
- No file size limitations

## Installation 📦

1. Clone the repository:

   ```bash
   git clone https://github.com/PXR05/ft_0.git
   cd ft_0
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Build the application:

   ```bash
   go build -o ft_0
   ```

## Usage 🎮

Launch the application:

```bash
./ft_0
```

### Send Mode 📤

1. Select "Send" from the main menu
2. Navigate through files using arrow keys
3. Press Enter to select a file
4. Share the displayed session ID with the receiver
5. Wait for receiver to connect and accept
6. Monitor transfer progress

### Receive Mode 📥

1. Select "Receive" from the main menu
2. Enter the session ID provided by sender
3. Review file details and accept/reject
4. Choose save location
5. Monitor download progress

### Relay Mode 🔄

1. Select "Relay" from the main menu
2. Press Enter to start relay server
3. Monitor active connections and transfers
4. Use Ctrl+C to stop server

## Project Structure 📁

```bash
FT_0/
├── main.go           # Application entry point
├── go.mod            # Go module definition
├── go.sum            # Dependencies checksum
├── server/           # Server-side logic
│   ├── connection.go # Connection management
│   ├── main.go       # Server configuration
│   ├── receiver.go   # File receiving logic
│   ├── relay.go      # Relay server implementation
│   ├── sender.go     # File sending logic
│   ├── session.go    # Session management
│   ├── types.go      # Type definitions
│   └── utils.go      # Utility functions
└── ui/               # User interface
    ├── main.go       # UI initialization
    ├── mode.go       # Mode selection
    ├── receive.go    # Receive UI
    ├── relay.go      # Relay UI
    └── send.go       # Send UI
```

## Configuration ⚙️

Default settings are defined in the server configuration:

- Chunk Size: 32KB
- Relay Protocol: HTTP
- Relay Server: localhost:3000
- Transfer Port: 3001

These can be modified in `server/main.go`.

## Technical Overview 🔧

FT_0 implements a hybrid architecture combining direct P2P transfers and relay-based communication:

### Network Architecture 🌐

- **Current Implementation** 🔌
  Uses a relay server for connection establishment and direct TCP for transfers

  - Relay server (port 3000)

    - Manages session creation and exchange
    - Handles initial handshake between peers
    - Provides session verification
    - Maintains active session registry

  - Transfer Protocol (port 3001)
    - Uses TCP for reliable file transmission
    - Establishes direct connection after session verification
    - Configurable 32KB chunk size for transfers
    - Full-duplex communication for control signals

- **Future P2P Enhancement** 📡
  Planned WebRTC implementation for true peer-to-peer transfers
  - NAT traversal without port forwarding
  - Browser-based file transfers
  - STUN/TURN server support
  - Direct connection between networks
  - Fallback to relay when direct connection fails

### Session Management 🔑

- Unique session IDs generated using cryptographic random bytes
- Sessions include:
  - Transfer metadata (filename, size, checksum)
  - Connection state tracking
  - Transfer progress monitoring
  - Timeout mechanisms (30s for handshake, configurable for transfer)

### Data Transfer Protocol 📨

1. **Handshake Phase** 🤝

   - Initial metadata exchange (file info)
   - Session ID verification
   - Transfer mode negotiation
   - Ready signal acknowledgment

2. **Transfer Phase** ⚡

   - Chunked streaming with 32KB blocks
   - TCP's built-in flow control
   - Real-time progress calculation
   - Speed monitoring using sliding window

3. **Completion Phase** ✅
   - Transfer verification
   - Connection teardown
   - Resource cleanup

### Error Handling 🛟

- Comprehensive error recovery for:
  - Network timeouts (30s default)
  - Connection drops (automatic session cleanup)
  - Invalid data chunks (transfer abort)
  - Resource exhaustion
  - Permission issues

### Performance Optimizations ⚡

- Buffered I/O operations (bufio package)
- Asynchronous progress updates
- Goroutine-based concurrent transfers
- Efficient memory usage with fixed buffer pools
- Minimal syscall overhead

### Security Considerations 🔒

- Session ID entropy ensures transfer privacy
- Built-in file access validation
- Configurable transfer restrictions
- Clean session termination
- Planned: End-to-end encryption

## Future Roadmap 🗺️

- Fixes:
  - [x] Send mode restart on rejection/error
  - [x] Mode switching stability
  - [x] Enhanced error handling
- Planned Features:
  - [ ] End-to-end encryption
  - [ ] File compression
  - [ ] WebRTC support
  - [ ] Batch file transfers
  - [ ] Directory transfers

## Contributing 🤝

Contributions are welcome. Please feel free to submit pull requests.

## License 📄

[MIT License](LICENSE)

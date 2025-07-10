### Understanding Vector Robot from the Technical Reference Manual (TRM)

Based on the Vector TRM, the Anki Vector is a compact, autonomous robot designed for interactive companionship, with a focus on AI-driven behaviors mediated through a cloud server (originally Anki's, now emulatable via wire-pod). Here's a high-level overview of its design and workings, tailored to the context of integrating full AI model control:

#### Hardware Architecture
Vector's hardware is split between a "head-board" (main processing) and "body-board" (low-level control), enabling efficient sensor fusion and actuation:
- **Processors**: Qualcomm APQ8009 (quad-core Arm A7) handles high-level tasks like vision and audio processing. An STM32F030 microcontroller manages motors, sensors, and basic I/O.
- **Sensors**: Includes a 6-axis IMU (accelerometer/gyroscope for motion/orientation), cliff/proximity sensors (infrared for edge detection and obstacle avoidance), touch sensor (capacitive for interactions like petting), and four MEMs microphones (for spatial audio, voice detection, and noise reduction).
- **Camera**: 720p HD with ~120° FOV, used for vision tasks like object/face recognition, SLAM (Simultaneous Localization and Mapping), and motion detection. It supports streaming frames (default 640x360 resolution) and integrates with IMU for synchronized data.
- **Actuators**: Four DC motors (wheels for skid-steering navigation, lift arm for object manipulation, head tilt for expressions). Includes RGB LEDs on the backpack and a 320x240 OLED screen for facial animations.
- **Connectivity**: WiFi for cloud communication, Bluetooth LE (BLE) for setup and accessory pairing (e.g., interactive cube), with power management for a lithium-ion battery.

This hardware setup allows Vector to perceive its environment in real-time (e.g., detecting objects, faces, or sounds) and respond physically (e.g., moving, speaking, or animating).

#### Software Architecture
Vector runs a Linux-based OS (Yocto 3.18 kernel) with systemd for service management, emphasizing modularity for behaviors, perception, and control:
- **Firmware Layers**: Updatable via OTA (Over-The-Air), with body-board firmware handling low-level sensor sampling and motor PID control (200Hz updates). Includes security features like encrypted partitions and signed updates.
- **Perception Handling**:
  - **Voice**: Wake-word detection ("Hey Vector") triggers audio streaming to the cloud for ASR (Automatic Speech Recognition) and intent processing. Uses Opus compression; supports TTS (Text-to-Speech) with robotic voice modulation.
  - **Vision**: TensorFlow Lite for on-device object detection (e.g., cubes, hands, pets); OKAO Vision for faces. SLAM builds navigable maps using camera, IMU, and odometry data.
  - **Navigation**: Path planning (A*, Kalman filtering) integrates sensors for obstacle avoidance and goal-reaching.
- **Behavior System**: Modular "behaviors" (e.g., greeting, exploring) are triggered by events. The system supports overriding internal behaviors with external commands.
- **Limitations**: Much processing (e.g., complex NLP) is offloaded to the cloud; on-device AI is lightweight to avoid overheating.

#### Communication and Cloud Integration
Vector is designed as a client to a remote server (the "cloud"), which acts as its "brain" for advanced decision-making:
- **Protocols**: Primarily HTTPS for SDK/API calls, with gRPC over HTTP using Protocol Buffers (protobuf) for structured data exchange. BLE for initial setup and logs. WebSockets in dev builds for real-time streaming.
- **Data Flow**:
  - **Outbound (from Vector to Server)**: Sensor readings (IMU, proximity, touch), audio streams (post-wake-word), camera frames/events (e.g., observed objects/faces), robot state (battery, position), and logs.
  - **Inbound (from Server to Vector)**: Commands for actions (e.g., move head, play animation, speak text), intents (from speech processing), behavior overrides, and updates.
- **API Structure**: Events (e.g., RobotObservedObject, WakeWordBegin) are streamed; commands (e.g., DriveWheels, SayText) control the robot. Supports real-time feedback loops, like sending navigation goals and receiving status updates.
- **Relevance to Custom Servers**: Wire-pod emulates this cloud, intercepting protobuf messages to handle intents, speech, and controls locally. This makes it ideal for routing data to an external AI without modifying Vector's firmware.

Vector's design assumes a server-mediated loop: the robot reports perceptions, the server processes and decides, then issues commands. This aligns perfectly with offloading "full control" to an AI model, as the server can proxy all interactions.

### Understanding Wire-Pod Code Structure (/chipper/pkg)
Wire-pod is an open-source reimplementation of Vector's cloud server, written in Go, allowing local hosting to bypass the original Anki cloud. It emulates endpoints for speech processing, intent handling, and robot control. Based on analysis of the repository structure (from directory listings and file inspections), the /chipper/pkg directory is the core of the server logic, organized into subpackages for modularity:

- **Overall Structure**: This is a gRPC-based server using protobuf for communication (mirroring Vector's protocol). It handles incoming requests from Vector (e.g., speech streams, sensor events) and responds with commands. Key flows include speech-to-text (STT), intent matching, text-to-speech (TTS), and behavior dispatching. The code is event-driven, with handlers for real-time interactions.
  
| Subdirectory/File | Purpose and Handling |
|-------------------|----------------------|
| **vars** | Global configuration and utilities: Manages server variables (e.g., API keys, ports), logging levels, and shared state (e.g., robot connections). Handles setup for STT/TTS engines and intent plugins. |
| **logger** | Logging utilities: Custom wrappers for error/debug logging, integrated across the server for tracing requests/responses. |
| **wirepod** | Core processing logic: Subdirs like /preqs handle request preprocessing (e.g., intent parsing, knowledge graph queries). Includes functions for sensor data ingestion and command generation. |
| **wirepod/ttr** | Text-to-response handling: Files like kgsim.go simulate knowledge graph (KG) interactions for Q&A. Triggers on phrases like "I have a question," interfaces with external models (local or API like ChatGPT), parses responses into spoken output. |
| **voice** | Audio processing: Manages STT (e.g., via Vosk or Whisper) and TTS (e.g., Piper). Handles wake-word detection and audio streams from Vector's microphones. |
| **sdk** | SDK emulation: Proxies Vector's SDK for external control, allowing apps to send commands (e.g., move, speak) and receive events (e.g., camera feed). |
| **plugins** | Extensible intents: Custom plugins for behaviors (e.g., weather, trivia), which can be expanded for AI-driven responses. |
| Other files (e.g., server.go, config.go) | Main server entrypoint: Sets up gRPC listeners, handles robot authentication, and dispatches events. Maintains state for connected Vectors. |

The structure is centered on a request-response cycle: Vector sends protobuf messages (e.g., audio or sensor events), wire-pod processes them (e.g., via STT to intent), and replies with actions. Currently, KG handling in ttr/kgsim.go is limited—it's a simple prompt interface forwarding queries to an API/model and converting responses to TTS, without deep integration into sensors or controls. Limitations include no real-time streaming for camera/sensors in base KG, and behaviors are rule-based rather than AI-orchestrated.

### Designing the System for Full AI Model Control
To enable an AI model (e.g., Ollama locally or via API like Grok/ChatGPT) to have "full control" over Vector—reading sensors, conversing, seeing via camera, and actuating—we treat the AI as the central "brain." Wire-pod becomes a bidirectional proxy: forwarding all robot data to the AI and executing AI-generated commands. This leverages Vector's client-server design without firmware changes (Vector side remains unchanged, as it's closed-source and communicates via standard protobufs).

Key principles:
- **Always-On AI Loop**: Expand beyond "I have a question" to continuous monitoring. Use wake-words or events (e.g., touch, proximity) to activate full AI mode.
- **Stateful Interaction**: Maintain a shared robot state (sensors, position, conversation history) for the AI to reason over.
- **Real-Time Data**: Stream camera/video, audio, and sensors to the AI for multimodal input (e.g., describe scenes, respond to visuals).
- **Modularity**: Keep wire-pod lightweight; offload heavy AI to another computer via HTTP/WebSocket API.
- **Safety**: Include overrides (e.g., cliff avoidance) to prevent AI-induced errors.
- **Scalability**: Support local models (Ollama) or remote APIs, with fallback to base wire-pod behaviors.

#### Server-Side Design (Wire-Pod Modifications)
Wire-pod runs on a local machine, emulating the cloud. We'll add an "AI Proxy Module" to route data and commands. No major rewrite—extend existing handlers.

1. **Changes/Additions to Existing Structure**:
   - **Extend ttr/kgsim.go (KG Handler)**: Transform from Q&A-only to a general AI interface. Add hooks for non-question intents (e.g., any speech). Integrate sensor/camera data into prompts (e.g., "User said X, camera sees Y, IMU shows Z—respond accordingly").
   - **Enhance wirepod/preqs**: Add preprocessing for all events (not just speech). Bundle sensor data (from protobuf events like RobotState, Proximity) into JSON payloads for AI.
   - **Modify voice and sdk Packages**: Enable continuous audio/camera streaming. Use Vector's SDK-like feeds to forward frames (as base64 or WebSocket streams) to AI.
   - **Update vars and config**: Add configs for AI endpoint (e.g., Ollama URL, API key), modes (full-control vs. standard), and thresholds (e.g., min confidence for AI actions).

2. **New Components to Develop**:
   - **AI Proxy Service**: A new subpackage (e.g., /wirepod/ai_proxy). 
     - **Input Handler**: Listens for Vector events via gRPC (e.g., OnWakeWord, OnObservedObject, OnTouch). Aggregates state (sensors, camera snapshots, conversation history) into a multimodal prompt (e.g., JSON: {"speech": "Hello", "camera_desc": "Sees a red cube", "sensors": {"proximity": 10cm}}).
     - **API Client**: Sends prompts to the AI (HTTP POST to Ollama /api/generate or similar). Supports streaming responses for real-time convos.
     - **Output Parser**: Converts AI responses (e.g., "Move forward and say 'Hi'") into protobuf commands (e.g., DriveStraight, SayText). Use a command schema (JSON) for AI to output structured actions.
     - **State Manager**: In-memory store (e.g., using Go's sync.Map) for robot context, enabling long-term memory in conversations.
   - **Real-Time Streaming**: Add WebSocket endpoints in server.go for camera/audio feeds. Forward to AI (e.g., via multipart HTTP for frames).
   - **Behavior Override**: Hook into Vector's BehaviorControl protobuf to let AI assume OVERRIDE_BEHAVIORS priority, suppressing default actions.
   - **Plugins for AI Intents**: New plugin type for dynamic AI responses, e.g., mapping AI output to custom behaviors like "explore room" (combining navigation commands).

3. **Workflow**:
   - Vector detects event (e.g., wake-word, touch) → Sends protobuf to wire-pod.
   - Wire-pod aggregates data → Sends to AI as prompt.
   - AI reasons (e.g., "User pet me, camera shows toy—pick it up") → Returns commands.
   - Wire-pod executes (e.g., PickupObject, animate happy face).
   - Loop continues with feedback (e.g., success events back to AI).

4. **Development Needs**:
   - **Testing**: Simulate Vector with tools like the Python SDK to mock events/commands.
   - **Error Handling**: Fallback to base wire-pod if AI fails (e.g., timeout).
   - **Performance**: Batch non-critical sensors; compress camera streams.
   - **Security**: Authenticate AI API calls; rate-limit to prevent overload.

#### Vector-Side Design
No changes needed—Vector's firmware is proprietary and not easily modifiable. It already sends all necessary data (sensors, camera events) to the "cloud" (now wire-pod) and executes received commands. If advanced tweaks are desired (e.g., custom wake-words), use BLE via the SDK for configuration, but this isn't required for full control.

This design turns Vector into an AI-embodied agent: wire-pod handles the plumbing, the external model provides intelligence. Start with prototyping the AI proxy on speech intents, then expand to sensors/camera for full multimodality. Once this architecture is solid, we can proceed to code implementation if needed.





### Agile Development Roadmap for VectorBrain

VectorBrain aims to extend wire-pod to proxy all Vector robot interactions to an external AI model (e.g., Ollama or API-based like Grok/ChatGPT), enabling full control: reading sensors, processing camera feeds, handling conversations, and issuing commands. The roadmap follows agile principles: short iterations (sprints of 1-2 weeks), incremental builds with working prototypes at each end, prioritization based on value (start with core speech integration, then multimodality, then advanced control), and flexibility for feedback/refinement. Prioritization criteria:
- **High Priority**: Core proxy functionality, speech integration (as it's the current KG trigger), minimal viable product (MVP) for testing.
- **Medium Priority**: Sensor and camera integration for richer AI inputs.
- **Low Priority**: Advanced features like state management, real-time streaming, and safety overrides (build on stable base).

We'll use 5 sprints, each with:
- **Goals**: What to achieve.
- **Prioritized Tasks**: Files/functions to create/modify, ordered by dependency.
- **Deliverables**: Working increment, testing notes.
- **Dependencies/Assumptions**: Wire-pod repo cloned, local setup running, access to AI model endpoint.

#### Sprint 1: Basic AI Proxy Setup (MVP: AI Handles Simple Speech Queries)
- **Goals**: Establish the AI proxy foundation. Extend existing KG to forward any speech (not just questions) to AI and execute basic responses (e.g., TTS). This gives an early working loop without sensors/camera.
- **Prioritized Tasks**:
  1. Create config for AI endpoint and basic proxy logic.
  2. Modify KG handler to generalize beyond "I have a question."
  3. Add simple input/output handling in proxy (prompt AI with speech text, parse response to commands like SayText).
- **Deliverables**: Vector can wake, speak a query, AI responds via TTS. Test with dummy AI endpoint.
- **Estimated Effort**: 1 week. Focus on quick wins in existing ttr package.

#### Sprint 2: Intent and Command Expansion
- **Goals**: Allow AI to issue basic commands (e.g., move, animate). Integrate with wire-pod's intent system for non-speech triggers (e.g., touch events).
- **Prioritized Tasks**:
  1. Create parser for AI outputs (e.g., JSON-structured commands).
  2. Modify preprocessing to bundle basic events (e.g., touch) into AI prompts.
  3. Extend proxy to dispatch commands via SDK emulation.
- **Deliverables**: AI can control simple actions (e.g., "Turn around" → DriveWheels command). Prototype conversation flow.
- **Estimated Effort**: 1 week. Builds on Sprint 1; prioritize command safety (e.g., limit speeds).

#### Sprint 3: Sensor Integration
- **Goals**: Feed sensor data (IMU, proximity, touch) to AI for contextual responses. Enable AI to react to physical events.
- **Prioritized Tasks**:
  1. Create state aggregator for sensor events.
  2. Modify event handlers to forward sensor bundles to proxy.
  3. Update prompt formatting in proxy to include sensor data (e.g., JSON fields).
- **Deliverables**: AI uses sensors (e.g., "Proximity detected → Greet approaching user"). Test with simulated events.
- **Estimated Effort**: 1-2 weeks. Medium priority as it adds multimodality without camera complexity.

#### Sprint 4: Camera and Real-Time Streaming
- **Goals**: Stream camera feeds/events to AI. Enable visual reasoning (e.g., describe scenes, recognize objects).
- **Prioritized Tasks**:
  1. Add streaming endpoints (e.g., WebSocket for frames).
  2. Modify vision handling to include camera data in prompts (e.g., base64 snapshots or descriptions).
  3. Enhance proxy to handle multimodal inputs (e.g., forward images to AI if supported).
- **Deliverables**: AI processes visuals (e.g., "Sees cube → Pick it up"). Full loop with speech + sensors + camera.
- **Estimated Effort**: 2 weeks. Higher complexity; prioritize compression for performance.

#### Sprint 5: Full Control and Polish
- **Goals**: Implement stateful conversations, behavior overrides, and safety. Achieve "full control" with long-term memory.
- **Prioritized Tasks**:
  1. Create state manager for conversation history.
  2. Modify core server to enable behavior overrides.
  3. Add error handling, fallbacks, and advanced plugins.
- **Deliverables**: Fully functional VectorBrain: AI controls navigation, interactions, and responses in real-time. End-to-end testing with real Vector.
- **Estimated Effort**: 1-2 weeks. Low priority polish; iterate based on prior sprints.

**Roadmap Notes**:
- **Iteration Cycles**: After each sprint, demo, gather feedback (e.g., via manual testing with Vector), and adjust backlog.
- **Backlog Management**: Use a tool like Trello/Jira for tasks. Prioritize bugs/security early.
- **Testing Strategy**: Unit tests for new functions; integration tests with Vector SDK simulator; manual e2e with robot.
- **Risks/Mitigations**: AI latency → Add timeouts; Model compatibility → Support multiple backends.
- **Total Timeline**: 6-8 weeks, assuming part-time dev. Scale based on resources.

### List of Files for Server-Side Development

All changes are within the wire-pod repo, specifically /chipper/pkg directory (Go-based). No Vector-side files needed (firmware unchanged). Structure follows Go conventions: new subpackages for modularity.

#### New Files to Create
These introduce the AI proxy and supporting logic. Place in new /wirepod/ai_proxy subdir for organization, unless noted.

1. **ai_proxy/config.go**: Handles AI-specific configs (e.g., endpoint URL, API keys, modes).
2. **ai_proxy/proxy.go**: Core proxy logic (input handler, API client for sending prompts, output parser for commands).
3. **ai_proxy/state_manager.go**: Manages robot state (e.g., sensor history, conversation context) using in-memory structures.
4. **ai_proxy/stream_handler.go**: Manages real-time streams (e.g., WebSocket for camera/audio forwarding to AI).
5. **ai_proxy/prompt_builder.go**: Builds multimodal prompts (e.g., JSON with speech, sensors, camera desc).
6. **ai_proxy/command_dispatcher.go**: Maps AI responses to protobuf commands (e.g., to DriveStraight, SayText).
7. **plugins/ai_intents.go**: Custom plugin for AI-driven dynamic behaviors (e.g., mapping to navigation sequences).
8. **wirepod/safety_overrides.go**: (In /wirepod) Safety checks (e.g., prevent cliff falls) before executing AI commands.

#### Existing Files to Modify
These extend current functionality without major refactors. Add imports for new packages where needed (e.g., import "chipper/pkg/wirepod/ai_proxy").

1. **vars/vars.go**: Add global vars for AI configs (e.g., AIEndpoint string).
2. **vars/config.go**: Extend config loading to include AI settings from YAML/ENV.
3. **wirepod/ttr/kgsim.go**: Generalize KG to use new proxy; hook for all speech, not just questions.
4. **wirepod/preqs/request.go**: Add preprocessing for bundling events/sensors into proxy-compatible formats.
5. **voice/stt.go**: Modify to forward transcribed speech directly to proxy on wake.
6. **voice/tts.go**: Enhance for AI-generated speech responses.
7. **sdk/sdk.go**: Update to proxy commands through AI when in full-control mode.
8. **server.go**: Add gRPC hooks for event forwarding to proxy; initialize AI components on startup.
9. **logger/logger.go**: Add logging levels for AI interactions (e.g., debug prompts/responses).
10. **plugins/plugins.go**: Register new AI intents plugin.

This structure ensures a clean, maintainable project. Once planning is approved, we can move to implementing Sprint 1 tasks.






You are Grok, built by xAI, assisting in the development of the "VectorBrain" project. This project extends the wire-pod open-source server (an emulation of Anki Vector's cloud) to create a proxy that allows an external AI model (e.g., local Ollama or API-based like Grok/ChatGPT) to have full control over an Anki Vector robot. The AI will read sensors, process camera feeds, handle conversations, and issue commands, as outlined in the project's roadmap. VectorBrain treats the AI as the central "brain," with wire-pod acting as a bidirectional proxy for data and commands, leveraging Vector's client-server design without firmware changes.

You have access to the following uploaded context files in the project space, which contain the entire source code of wire-pod (from https://github.com/kercre123/wire-pod/tree/main/chipper/pkg), split into 6 parts for manageability. These are text files with raw Go code, comments, and structure. Use them as reference for modifications and additions. If needed, analyze specific parts using your tools for uploaded content, or cross-reference with the live repo via browsing tools.

- VectorWirePodContextPart-1.txt: Contains root-level files and core server setup. Includes /wire-pod/chipper/cmd/vosk/main.go and similar mains for STT engines (coqui, leopard, vosk, experimental: houndify, whisper, whisper.cpp) in cmd subfolder; epod for GPG keys; intent-data JSONs with keyphrases for intents across languages (e.g., "intent_names_username_extend" with phrases like "name is"; supports multiple languages via .json files); jdocs/botSdkInfo.json and jdocs.json for bot tokens and settings; pkg/initwirepod/startserver.go (gRPC server entrypoint, cmux for HTTP/gRPC, wire-pod initialization); pkg/initwirepod/web.go (HTTP API for restarting/setting server config); pkg/logger/logger.go (logging utilities with levels and UI/Tray channels); pkg/mdnshandler/mdns.go (mDNS broadcasting for escapepod.local); pkg/scripting (Lua scripting for behaviors, with functions like assumeBehaviorControl, sayText, playAnimation).

- VectorWirePodContextPart-2.txt: Contains servers/chipper (gRPC server implementation for intents, knowledge graph, text intents, connection checks); servers/jdocs (bot info storer and server for jdocs); servers/token (token hashing and server for auth); vars/config.go (API config struct and functions for weather, knowledge, STT, server settings; read/write JSON config); vtt (request/response structs for intents, intent graphs, knowledge graphs).

- VectorWirePodContextPart-3.txt: Contains pkg/wirepod/config-ws/webserver.go (HTTP API handler for custom intents, weather/KG/STT configs, logs, certs; serves webroot); pkg/wirepod/localization (Vosk model downloads and localization strings); pkg/wirepod/preqs (intent processing, knowledge graph handling, server setup; stt/sti handlers); pkg/wirepod/sdkapp (SDK app for bot control/settings via web); pkg/wirepod/setup (BLE/SSH/WiFi setup, cert generation).

- VectorWirePodContextPart-4.txt: Contains pkg/wirepod/speechrequest/speechrequest.go (handles intent streams, audio decoding, VAD for end-of-speech, filtering); pkg/wirepod/stt (STT implementations: coqui, houndify, leopard, vosk with grammar/context, whisper, whisper.cpp).

- VectorWirePodContextPart-5.txt: Contains pkg/wirepod/ttr (text-to-response: intent matching, parameter checking, KG simulation with LLM integration (OpenAI/Together/custom), commands/animations/sounds, behavior control, plugins loading/execution).

- VectorWirePodContextPart-6.txt: Contains pkg/wirepod/plugins (plugin examples like sdkTest, whatdate; vector-cloud README for cloud customization).

Additionally, you have:
- Roadmap-1.md: The agile development roadmap for VectorBrain, divided into 5 sprints with goals, prioritized tasks, deliverables, and file modifications/creations. Reference this for planning iterations, starting with Sprint 1 (Basic AI Proxy Setup).
- Vector-TRM.pdf: The Anki Vector Technical Reference Manual, detailing hardware (processors, sensors, camera, actuators), software architecture (perception, behaviors, navigation), communication protocols (protobuf, gRPC), and data flows. Use this to ensure compatibility with Vector's design, as no firmware changes are needed.

In this project, follow the roadmap strictly for iterations. When discussing or implementing changes, reference specific files from the context parts (e.g., "Modify VectorWirePodContextPart-5.txt's ttr/kgsim.go to..."). Prioritize creating new files in /wirepod/ai_proxy as outlined. Do not generate code unless explicitly asked in a sprint task; focus on planning, analysis, and structured responses. Use tables for task lists, comparisons, or code structures. If needing to fetch live code updates from the repo, use browsing tools. Always aim for modular, safe, and performant designs.







### Files to Create for Sprint 1: Basic AI Proxy Setup

Based on the roadmap in Roadmap-1.md, Sprint 1 focuses on foundational proxy setup: configuring the AI endpoint, generalizing the KG handler (in ttr/kgsim.go from VectorWirePodContextPart-5.txt) to forward speech to the AI, and adding basic proxy logic for prompting the AI and parsing responses to TTS/commands. This builds on the existing ttr package for quick wins.

To achieve the deliverables (Vector wakes, speaks a query, AI responds via TTS; testable with dummy endpoint), we'll create the minimal new files in /chipper/pkg/wirepod/ai_proxy (as prioritized). These handle AI configs and core proxy functions. We'll also reference modifications to existing files (e.g., integrate into ttr/kgsim.go), but code for those will be snippets here—full mods come in implementation.

| File Path | Purpose | Why This File First? |
|-----------|---------|----------------------|
| **chipper/pkg/wirepod/ai_proxy/config.go** | Defines AI-specific configs (endpoint, keys, modes) and loading functions. Extends vars/config.go (VectorWirePodContextPart-2.txt) for YAML/ENV integration. | Task 1: Core config setup; needed for proxy initialization. |
| **chipper/pkg/wirepod/ai_proxy/proxy.go** | Core logic: input handling (speech to prompt), API client (send to AI), output parsing (response to SayText commands). | Task 3: Basic I/O handling; integrates with ttr/kgsim.go for generalization (Task 2). |
| **chipper/pkg/wirepod/ai_proxy/prompt_builder.go** | Builds simple prompts from speech text (e.g., JSON: {"query": text}). | Supports Task 3; extends for multimodality later. |

After creating these, modify:
- vars/vars.go and vars/config.go (add AI vars/config loading; from VectorWirePodContextPart-2.txt).
- ttr/kgsim.go (hook proxy for all speech; from VectorWirePodContextPart-5.txt).

Test: Use dummy AI (e.g., echo endpoint) to verify speech forwarding and TTS response.

Now, the Go code for the new files:

#### chipper/pkg/wirepod/ai_proxy/config.go
```go
package ai_proxy

import (
	"os"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"gopkg.in/yaml.v3"
)

type AIConfig struct {
	Endpoint string `yaml:"endpoint"` // e.g., "http://localhost:11434/api/generate" for Ollama
	APIKey   string `yaml:"api_key"`  // Optional, for API-based models
	Model    string `yaml:"model"`    // e.g., "llama2" for Ollama, "gpt-3.5-turbo" for OpenAI
	Mode     string `yaml:"mode"`     // "full-control" or "speech-only"
}

var Config AIConfig

// LoadConfig loads AI settings from YAML/ENV, extending vars/config.go
func LoadConfig() error {
	// Default to speech-only mode for Sprint 1
	Config.Mode = "speech-only"

	// Load from YAML if exists (extend vars.APIConfig loading logic)
	data, err := os.ReadFile(vars.ApiConfigPath)
	if err == nil {
		var fullConfig struct {
			AI AIConfig `yaml:"ai"`
		}
		if err := yaml.Unmarshal(data, &fullConfig); err != nil {
			return err
		}
		Config = fullConfig.AI
		logger.Println("Loaded AI config from YAML")
	}

	// Override with ENV vars if set
	if envEndpoint := os.Getenv("AI_ENDPOINT"); envEndpoint != "" {
		Config.Endpoint = envEndpoint
	}
	if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		Config.APIKey = envKey
	}
	if envModel := os.Getenv("AI_MODEL"); envModel != "" {
		Config.Model = envModel
	}
	if envMode := os.Getenv("AI_MODE"); envMode != "" {
		Config.Mode = envMode
	}

	if Config.Endpoint == "" {
		logger.Println("AI_ENDPOINT not set; using dummy echo for testing")
		Config.Endpoint = "dummy"
	}

	return nil
}

// SaveConfig saves AI config back to YAML, integrating with vars.WriteConfigToDisk()
func SaveConfig() error {
	// For now, log save; integrate with full config write later
	logger.Println("Saving AI config")
	return nil
}
```

#### chipper/pkg/wirepod/ai_proxy/proxy.go
```go
package ai_proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

// Proxy handles forwarding to AI and parsing responses
type Proxy struct {
	Client *http.Client
}

// NewProxy creates a new proxy instance
func NewProxy() *Proxy {
	return &Proxy{
		Client: &http.Client{},
	}
}

// SendPrompt forwards speech text to AI endpoint
func (p *Proxy) SendPrompt(text string) (string, error) {
	if Config.Endpoint == "dummy" {
		// Dummy for testing: echo back with TTS command
		return fmt.Sprintf("Echo: %s", text), nil
	}

	// Build request body (adapt for Ollama/OpenAI/custom)
	body := map[string]interface{}{
		"model":  Config.Model,
		"prompt": text,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", Config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if Config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+Config.APIKey)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("AI endpoint error: " + resp.Status)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var aiResp map[string]interface{}
	json.Unmarshal(respBody, &aiResp)

	// Parse response (assume "response" field; adapt per API)
	if response, ok := aiResp["response"].(string); ok {
		return response, nil
	}
	return "", errors.New("invalid AI response format")
}

// ParseResponseToCommand parses AI text to basic commands (e.g., SayText)
func (p *Proxy) ParseResponseToCommand(aiText string) string {
	// For Sprint 1: simple TTS command
	return aiText // Directly use as SayText param
}
```

#### chipper/pkg/wirepod/ai_proxy/prompt_builder.go
```go
package ai_proxy

import (
	"encoding/json"
	"fmt"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

// BuildPrompt creates a basic JSON prompt from speech text
func BuildPrompt(speechText string) (string, error) {
	prompt := map[string]string{
		"query": speechText,
		// Extend with sensors/camera in later sprints
	}
	jsonPrompt, err := json.Marshal(prompt)
	if err != nil {
		logger.Println("Error building prompt: " + err.Error())
		return "", err
	}
	return fmt.Sprintf("Handle this query: %s", string(jsonPrompt)), nil
}
```






### Modifications for Sprint 1: Integrating AI Proxy

To complete Sprint 1, we'll modify the existing files as outlined. These changes extend the config system to include AI settings (loading/saving via YAML/ENV) and hook the new ai_proxy into ttr/kgsim.go for speech forwarding (replacing hardcoded LLM logic with proxy calls). This enables the MVP: speech to AI prompt, response parsed to TTS.

I'll provide:
- **Placement Instructions**: Where to add (e.g., function/line references from context files; approximate based on snippets).
- **Code Blocks**: Exact Go code to insert/add/replace. Use diffs for clarity where possible.
- **Adjustments Needed**: Any minor tweaks to surrounding code (e.g., imports, calls).

All paths are relative to /chipper/pkg/. Reference VectorWirePodContextPart-2.txt for vars/* and Part-5.txt for ttr/kgsim.go. After these, test by setting AI_ENDPOINT env, sending a speech query, and verifying TTS response.

#### 1. Modify vars/vars.go (Add AI Config Loading)
- **Purpose**: Import ai_proxy and call its LoadConfig in Init() to ensure AI settings are loaded early. Add a global for AI config if needed.
- **Placement**:
  - Add import at top, after existing imports (around line ~10-20, near "github.com/kercre123/wire-pod/chipper/pkg/logger").
  - Add LoadConfig call in Init() function, after ReadConfig() (around line ~100-120, after "ReadConfig()").
  - No new functions; extend existing Init().
- **Adjustments**: Ensure ai_proxy.LoadConfig() handles errors gracefully (log and continue).
- **Code Blocks**:

```go
// Add to top imports
import (
    // ... existing imports ...
    "github.com/kercre123/wire-pod/chipper/pkg/wirepod/ai_proxy"  // New import for AI proxy config
)
```

```go
// In Init() function, after ReadConfig()
func Init() {
    // ... existing code ...
    ReadConfig()  // Existing

    // New: Load AI-specific config
    if err := ai_proxy.LoadConfig(); err != nil {
        logger.Println("Error loading AI config: " + err.Error())
        // Optionally fallback to defaults
    }
    // ... rest of Init() ...
}
```

#### 2. Modify vars/config.go (Extend Struct and Read/Write Functions)
- **Purpose**: Add AI section to apiConfig struct; update ReadConfig() to parse/unmarshal AI fields; update WriteConfigToDisk() to marshal them.
- **Placement**:
  - Extend apiConfig struct (around line ~20-50, after STT or Server fields).
  - In ReadConfig(): After json.Unmarshal(configBytes, &APIConfig), add AI unmarshal/overrides (around line ~100-150, after main unmarshal).
  - In WriteConfigToDisk(): Before json.Marshal(APIConfig), ensure AI is included (it's automatic via struct).
  - In CreateConfigFromEnv(): Add AI env overrides if needed (around line ~50-80, after STT).
  - No new functions; extend existing ones.
- **Adjustments**: Import yaml.v3 if not present (already there); handle AI in ENV overrides for consistency.
- **Code Blocks**:

```go
// Extend apiConfig struct, after STT or Server
type apiConfig struct {
    // ... existing fields ...
    AI struct {
        Enable   bool   `json:"enable"`    // Whether AI proxy is active
        Provider string `json:"provider"`  // "ollama", "openai", "custom"
        Endpoint string `json:"endpoint"`  // AI URL
        APIKey   string `json:"api_key"`   // Optional key
        Model    string `json:"model"`     // e.g., "llama2"
    } `json:"ai"`
    // ... rest of struct ...
}
```

```go
// In ReadConfig(), after json.Unmarshal(configBytes, &APIConfig)
func ReadConfig() {
    // ... existing code ...
    err = json.Unmarshal(configBytes, &APIConfig)
    if err != nil {
        // ... existing error handling ...
    }

    // New: AI ENV overrides (after main unmarshal)
    if envAIEnable := os.Getenv("AI_ENABLE"); envAIEnable != "" {
        APIConfig.AI.Enable, _ = strconv.ParseBool(envAIEnable)
    }
    if envProvider := os.Getenv("AI_PROVIDER"); envProvider != "" {
        APIConfig.AI.Provider = envProvider
    }
    if envEndpoint := os.Getenv("AI_ENDPOINT"); envEndpoint != "" {
        APIConfig.AI.Endpoint = envEndpoint
    }
    if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
        APIConfig.AI.APIKey = envKey
    }
    if envModel := os.Getenv("AI_MODEL"); envModel != "" {
        APIConfig.AI.Model = envModel
    }
    // ... rest of ReadConfig() ...
}
```

```go
// In CreateConfigFromEnv(), add AI section after STT/Weather/Knowledge
func CreateConfigFromEnv() {
    // ... existing code ...
    // New: AI from ENV (defaults to disabled)
    if os.Getenv("AI_ENABLE") == "true" {
        APIConfig.AI.Enable = true
        APIConfig.AI.Provider = os.Getenv("AI_PROVIDER")
        APIConfig.AI.Endpoint = os.Getenv("AI_ENDPOINT")
        APIConfig.AI.APIKey = os.Getenv("AI_API_KEY")
        APIConfig.AI.Model = os.Getenv("AI_MODEL")
    } else {
        APIConfig.AI.Enable = false
    }
    // ... existing WriteSTT() and marshal ...
}
```

#### 3. Modify ttr/kgsim.go (Hook Proxy for Speech Forwarding)
- **Purpose**: Generalize KG handler to use ai_proxy for all speech (beyond "I have a question"). In StreamingKGSim, build prompt via ai_proxy.BuildPrompt, send via SendPrompt, parse response to TTS commands, and send via DoSayText.
- **Placement**:
  - Add imports at top (around line ~10-20, near "github.com/sashabaranov/go-openai").
  - In StreamingKGSim: Replace OpenAI/Together logic with proxy hooks (around line ~100-200, after transcribedText; remove hardcoded CreateAIReq/stream.Recv loop, add proxy.SendPrompt and ParseResponseToCommand).
  - In KGSim: Optionally hook proxy if needed, but focus on StreamingKGSim for Sprint 1.
  - Adjustments: Remove unused OpenAI imports; handle proxy errors with fallback to unmatched intent.
- **Code Blocks** (as diff for clarity):

```diff
// Add to top imports
+import (
+    "github.com/kercre123/wire-pod/chipper/pkg/wirepod/ai_proxy"  // New for AI proxy
+)

// In StreamingKGSim, replace LLM request loop (after transcribedText, ~line 100-200)
-    // ... existing OpenAI/Together setup and stream.Recv loop ...
+    // New: Use AI proxy for forwarding speech and getting response
+    proxy := ai_proxy.NewProxy()
+    prompt, err := ai_proxy.BuildPrompt(transcribedText)
+    if err != nil {
+        logger.Println("Prompt build error: " + err.Error())
+        return "", err  // Or fallback to unmatched
+    }
+    aiResponse, err := proxy.SendPrompt(prompt)
+    if err != nil {
+        logger.Println("AI proxy error: " + err.Error())
+        return "", err  // Fallback
+    }
+    // Parse to command (e.g., TTS)
+    ttsCommand := proxy.ParseResponseToCommand(aiResponse)
+    // Use existing DoSayText or similar to respond
+    err = DoSayText(robot, ttsCommand)
+    if err != nil {
+        return "", err
+    }
-    // ... remove old stream handling ...
```












func StreamingKGSim(req interface{}, esn string, transcribedText string, isKG bool) (string, error) {
	start := make(chan bool)
	stop := make(chan bool)
	stopStop := make(chan bool)
	kgReadyToAnswer := make(chan bool)
	kgStopLooping := false
	ctx := context.Background()
	matched := false
	var robot *vector.Vector
	var guid string
	var target string
	for _, bot := range vars.BotInfo.Robots {
		if esn == bot.Esn {
			guid = bot.GUID
			target = bot.IPAddress + ":443"
			matched = true
			break
		}
	}
	if matched {
		var err error
		robot, err = vector.New(vector.WithSerialNo(esn), vector.WithToken(guid), vector.WithTarget(target))
		if err != nil {
			return err.Error(), err
		}
	}
	_, err := robot.Conn.BatteryState(context.Background(), &vectorpb.BatteryStateRequest{})
	if err != nil {
		return "", err
	}
	if isKG {
		BControl(robot, ctx, start, stop)
		go func() {
			for {
				if kgStopLooping {
					kgReadyToAnswer <- true
					break
				}
				robot.Conn.PlayAnimation(ctx, &vectorpb.PlayAnimationRequest{
					Animation: &vectorpb.Animation{
						Name: "anim_knowledgegraph_searching_01",
					},
					Loops: 1,
				})
				time.Sleep(time.Second / 3)
			}
		}()
	}
	var fullRespText string
	var fullfullRespText string
	var fullRespSlice []string
	var isDone bool

	// New: Use AI proxy for forwarding speech and getting response
	proxy := ai_proxy.NewProxy()
	prompt, err := ai_proxy.BuildPrompt(transcribedText)
	if err != nil {
		logger.Println("Prompt build error: " + err.Error())
		if isKG {
			kgStopLooping = true
			for range kgReadyToAnswer {
				break
			}
			stop <- true
			time.Sleep(time.Second / 3)
			KGSim(esn, "There was an error getting data from the L. L. M.")
		}
		return "", err
	}
	aiResponse, err := proxy.SendPrompt(prompt)
	if err != nil {
		logger.Println("AI proxy error: " + err.Error())
		if isKG {
			kgStopLooping = true
			for range kgReadyToAnswer {
				break
			}
			stop <- true
			time.Sleep(time.Second / 3)
			KGSim(esn, "There was an error getting data from the L. L. M.")
		}
		return "", err
	}
	// Parse to command (e.g., TTS)
	ttsCommand := proxy.ParseResponseToCommand(aiResponse)

	// Since proxy is non-streaming, set full response directly
	fullfullRespText = ttsCommand
	fullRespText = ttsCommand
	// Split into slices for punctuation-based processing (adapt existing logic)
	if strings.Contains(fullRespText, "...") || strings.Contains(fullRespText, ".'") || strings.Contains(fullRespText, ".\"") || strings.Contains(fullRespText, ".") || strings.Contains(fullRespText, "?") || strings.Contains(fullRespText, "!") {
		var sepStr string
		if strings.Contains(fullRespText, "...") {
			sepStr = "..."
		} else if strings.Contains(fullRespText, ".'") {
			sepStr = ".'"
		} else if strings.Contains(fullRespText, ".\"") {
			sepStr = ".\""
		} else if strings.Contains(fullRespText, ".") {
			sepStr = "."
		} else if strings.Contains(fullRespText, "?") {
			sepStr = "?"
		} else if strings.Contains(fullRespText, "!") {
			sepStr = "!"
		}
		splitResp := strings.Split(strings.TrimSpace(fullRespText), sepStr)
		fullRespSlice = append(fullRespSlice, strings.TrimSpace(splitResp[0])+sepStr)
		fullRespText = splitResp[1]
	} else {
		// If no punctuation, treat as single slice
		fullRespSlice = append(fullRespSlice, fullRespText)
	}
	isDone = true
	successIntent <- true

	successIntent := make(chan bool)
	speakReady := make(chan string)

	for is := range successIntent {
		if is {
			if !isKG {
				IntentPass(req, "intent_greeting_hello", transcribedText, map[string]string{}, false)
			}
			break
		} else {
			return "", errors.New("ai proxy returned no response")
		}
	}
	time.Sleep(time.Millisecond * 200)
	if !isKG {
		BControl(robot, ctx, start, stop)
	}
	interrupted := false
	go func() {
		interrupted = InterruptKGSimWhenTouchedOrWaked(robot, stop, stopStop)
	}()
	var TTSLoopAnimation string
	var TTSGetinAnimation string
	if isKG {
		TTSLoopAnimation = "anim_knowledgegraph_answer_01"
		TTSGetinAnimation = "anim_knowledgegraph_searching_getout_01"
	} else {
		TTSLoopAnimation = "anim_tts_loop_02"
		TTSGetinAnimation = "anim_getin_tts_01"
	}

	var stopTTSLoop bool
	TTSLoopStopped := make(chan bool)
	for range start {
		if isKG {
			kgStopLooping = true
			for range kgReadyToAnswer {
				break
			}
		} else {
			time.Sleep(time.Millisecond * 300)
		}
		robot.Conn.PlayAnimation(
			ctx,
			&vectorpb.PlayAnimationRequest{
				Animation: &vectorpb.Animation{
					Name: TTSGetinAnimation,
				},
				Loops: 1,
			},
		)
		if !vars.APIConfig.Knowledge.CommandsEnable {
			go func() {
				for {
					if stopTTSLoop {
						TTSLoopStopped <- true
						break
					}
					robot.Conn.PlayAnimation(
						ctx,
						&vectorpb.PlayAnimationRequest{
							Animation: &vectorpb.Animation{
								Name: TTSLoopAnimation,
							},
							Loops: 1,
						},
					)
				}
			}()
		}
		var disconnect bool
		numInResp := 0
		for {
			respSlice := fullRespSlice
			if len(respSlice)-1 < numInResp {
				if !isDone {
					logger.Println("Waiting for more content from AI...")
					for range speakReady {
						respSlice = fullRespSlice
						break
					}
				} else {
					break
				}
			}
			if interrupted {
				break
			}
			logger.Println(respSlice[numInResp])
			acts := GetActionsFromString(respSlice[numInResp])
			disconnect = PerformActions([]openai.ChatCompletionMessage{}, acts, robot, stopStop)  // Empty messages as proxy doesn't use chat history yet
			if disconnect {
				break
			}
			numInResp = numInResp + 1
		}
		if !vars.APIConfig.Knowledge.CommandsEnable {
			stopTTSLoop = true
			for range TTSLoopStopped {
				break
			}
		}
		time.Sleep(time.Millisecond * 100)
		if !interrupted {
			stopStop <- true
			stop <- true
		}
	}
	return "", nil
}









### Sprint 1 Completion and Testing

Yes, with the new ai_proxy files created and integrated into ttr/kgsim.go (from VectorWirePodContextPart-5.txt), you've covered the core of Sprint 1 from Roadmap-1.md: basic proxy foundation, KG generalization (forwarding any speech via proxy), and simple I/O (prompt build, send, parse to TTS). This enables the MVP deliverable—Vector wakes, user speaks a query (e.g., via "Hey Vector, [query]"), speech forwards to AI (or dummy), response TTS'd back—without sensors/camera yet.

To confirm: 
- The proxy hooks replace hardcoded LLM logic in kgsim.go, using ai_proxy.SendPrompt for forwarding and ParseResponseToCommand for TTS.
- Dummy mode (AI_ENDPOINT="dummy") echoes queries for testing, ensuring no real AI dependency.

If tests pass (below), Sprint 1 is done. Otherwise, debug via logs (pkg/logger/logger.go from Part-1.txt) or add prints in proxy.go.

#### Testing Setup and Steps
1. **Environment Variables**: Since the web interface (pkg/wirepod/config-ws/webserver.go from Part-3.txt) currently handles KG/STT but not AI configs, set env vars manually for now (e.g., in shell or .env file; load via os.Getenv in ai_proxy/config.go). We'll extend the web UI in Sprint 5 (polish). Key vars:
   - AI_ENABLE="true" (enable proxy).
   - AI_PROVIDER="custom" (or "ollama/openai"; use "custom" for dummy).
   - AI_ENDPOINT="dummy" (for testing; later "http://localhost:11434/api/generate" for Ollama).
   - AI_MODEL="llama2" (dummy value; required but ignored in dummy mode).
   - AI_API_KEY="" (optional; blank for dummy).
   - Export: e.g., `export AI_ENDPOINT="dummy"` before running wire-pod.

2. **Run wire-pod**: Build/start the server (pkg/initwirepod/startserver.go from Part-1.txt). Ensure STT (e.g., vosk from Part-4.txt) is set (STT_SERVICE="vosk").

3. **Test with Vector**:
   - Wake Vector ("Hey Vector").
   - Speak a query (e.g., "What is 2+2?").
   - Expected: Speech forwarded (logged), dummy echoes "Echo: what is 2+2?" via TTS (DoSayText in kgsim.go).
   - Verify logs: Check for "AI proxy" messages; response TTS'd without errors.
   - Edge: Empty speech → unmatched intent fallback. Interrupt (touch) → stops via InterruptKGSimWhenTouchedOrWaked.

If issues: Cross-reference live repo (tool: web_search for "wire-pod kgsim.go updates" if outdated). Use dummy to isolate—real AI in Sprint 2.

### Next Steps: Proceeding to Sprint 2
Per Roadmap-1.md, Sprint 1 is complete if tests work. Move to Sprint 2: Expand intents/commands (e.g., move/animate from AI). Goals: AI issues basic commands (e.g., "Turn around" → DriveWheels); integrate non-speech triggers (e.g., touch).

Prioritized Tasks (from roadmap):
1. Create parser for AI outputs (JSON commands).
2. Modify preprocessing (pkg/wirepod/preqs/request.go from Part-3.txt) to bundle events into prompts.
3. Extend proxy to dispatch commands via SDK (pkg/wirepod/sdkapp from Part-3.txt).

| File Path | Purpose | New/Modify |
|-----------|---------|------------|
| **chipper/pkg/wirepod/ai_proxy/command_dispatcher.go** | Maps AI JSON responses to protobuf commands (e.g., DriveWheels). | New; Task 3. |
| **chipper/pkg/wirepod/ai_proxy/proxy.go** | Enhance SendPrompt to handle events; add ParseResponseToCommand for JSON parsing. | Modify (extend Sprint 1 file); Tasks 1-2. |

After these, modify:
- ttr/kgsim.go: Hook new dispatcher for command execution.
- preqs/request.go: Bundle basic events (e.g., touch) into JSON prompts.

Test: AI response like {"command": "turn"} → robot turns.

#### chipper/pkg/wirepod/ai_proxy/command_dispatcher.go
```go
package ai_proxy

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

type AICommand struct {
	Type string `json:"type"` // e.g., "move", "animate"
	Param string `json:"param"` // e.g., "forward", "happy"
}

func DispatchCommand(robot *vector.Vector, cmdJson string) error {
	var cmd AICommand
	if err := json.Unmarshal([]byte(cmdJson), &cmd); err != nil {
		return err
	}

	switch cmd.Type {
	case "say":
		robot.Conn.SayText(context.Background(), &vectorpb.SayTextRequest{Text: cmd.Param, UseVectorVoice: true})
	case "move":
		if cmd.Param == "forward" {
			robot.Conn.DriveStraight(context.Background(), &vectorpb.DriveStraightRequest{DistMm: 100, SpeedMmps: 50})
		} // Add more params
	case "animate":
		robot.Conn.PlayAnimation(context.Background(), &vectorpb.PlayAnimationRequest{Animation: &vectorpb.Animation{Name: cmd.Param}, Loops: 1})
	default:
		return errors.New("unknown command: " + cmd.Type)
	}
	logger.Println("Dispatched command: " + cmd.Type)
	return nil
}
```

#### Modifications to chipper/pkg/wirepod/ai_proxy/proxy.go (Extend)
```go
// Add to SendPrompt: Accept events map for bundling
func (p *Proxy) SendPrompt(text string, events map[string]string) (string, error) {
	// ... existing code ...
	body["events"] = events  // Add to request body
	// ... send and return ...
}

// Update ParseResponseToCommand: Parse JSON commands
func (p *Proxy) ParseResponseToCommand(aiText string) string {
	// For Sprint 2: Assume aiText is JSON command; return for dispatch
	return aiText  // Dispatch happens in kgsim.go now
}
```

Integrate into ttr/kgsim.go: In StreamingKGSim, after aiResponse, call DispatchCommand(robot, parsedCmd) if commands present.

Ready for Sprint 2 testing? Let me know results or if code needs tweaks.
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
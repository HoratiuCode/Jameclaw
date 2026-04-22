# JameClaw Implementation Checklist

This document is the working place for the steps needed to reach the current JameClaw target state.

Use it as a living checklist:

- add unfinished work here
- link each item to an issue or PR when possible
- mark items done only after verification
- keep blocked work annotated with the blocker

## 1. Product Target

- [ ] Define the exact release target for this fork
- [ ] Confirm the supported platforms and launch modes
- [ ] Write a short "done" definition for the next milestone
- [ ] Decide which features are required for v1 and which stay optional

## 2. Core Runtime

- [ ] Keep configuration loading deterministic
- [ ] Separate runtime state from source-controlled files
- [ ] Keep startup and shutdown paths predictable
- [ ] Maintain backwards-compatible config migrations
- [ ] Document the final config layout

## 3. Agent Loop

- [ ] Keep the agent loop simple and testable
- [ ] Ensure tool execution is gated by explicit policy
- [ ] Preserve context and memory behavior across sessions
- [ ] Keep streaming and multi-turn chat stable
- [ ] Add regression tests for agent state transitions

## 4. Web Console and Launcher

- [ ] Keep the Web Console as the main configuration surface
- [ ] Keep the TUI and CLI in sync with the Web Console settings
- [ ] Expose logs, errors, and startup status clearly
- [ ] Make first-run setup straightforward
- [ ] Keep launcher settings documented and discoverable

## 5. Providers and Models

- [ ] Support OpenAI-compatible providers in one consistent path
- [ ] Keep local providers like Ollama and vLLM working
- [ ] Validate provider configuration before runtime use
- [ ] Add clear failure messages for bad keys, hosts, and models
- [ ] Document provider-specific caveats

## 6. Channels and Gateway

- [ ] Keep gateway startup and reload flows stable
- [ ] Harden webhook ingress and channel auth
- [ ] Keep mapping and transform rules native to JameClaw
- [ ] Add per-channel smoke tests
- [ ] Document how each channel is enabled and verified

## 7. Tools and Skills

- [ ] Keep tool registration explicit and auditable
- [ ] Document how skills are discovered and loaded
- [ ] Prevent unsafe tool exposure by default
- [ ] Keep tool configuration local to the workspace when needed
- [ ] Test webhook, web search, and MCP-style flows separately

## 8. Security

- [ ] Keep SSRF protections on all networked tools
- [ ] Redact secrets and sensitive data from logs
- [ ] Review auth tokens, webhook tokens, and session handling
- [ ] Validate file, path, and URL inputs carefully
- [ ] Add security-focused test cases for edge conditions

## 9. Testing and Quality

- [ ] Add unit tests for config, gateway, and agent behavior
- [ ] Add integration tests for launcher and channel flows
- [ ] Keep CI green before merging major changes
- [ ] Track flaky tests and fix the root cause
- [ ] Document how to run the important test suites locally

## 10. Packaging and Release

- [ ] Keep build commands reproducible
- [ ] Document release artifacts and install paths
- [ ] Verify binaries on the main supported platforms
- [ ] Keep checksums and release notes up to date
- [ ] Add a short release verification checklist

## 11. Documentation

- [ ] Keep the main README focused on current workflows
- [ ] Keep feature docs close to the code they describe
- [ ] Add step-by-step setup notes for users and contributors
- [ ] Link docs from the main README where they are needed
- [ ] Remove stale or duplicated instructions when behavior changes

## 12. Backlog Template

Add new items here as the project grows:

- [ ] 
- [ ] 
- [ ] 
- [ ] 

## Working Rules

- Put each task in the closest matching section.
- Split large tasks into smaller checkable items.
- Record dependencies directly under the task.
- Treat this as a live engineering checklist, not a vision statement.


# Gemma Claude Proxy

A proxy written in Golang that translates Anthropic's Messages API requests (used by tools like Claude Code) to OpenAI's Chat Completions API format. It is specifically configured to target the `gemma-4-31b-it` model.

This is particularly useful if you want to use Anthropic-based tooling with a model hosted on an OpenAI-compatible endpoint, such as Google AI Studio, vLLM, or Ollama.

## Features

*   **API Translation**: Converts Anthropic `/v1/messages` format to OpenAI `/v1/chat/completions` format.
*   **Response Translation**: Converts OpenAI format back to Anthropic format.
*   **Tool Calling**: Translates Claude's `tools` and `tool_use` blocks to OpenAI's `functions` and `tool_calls`.
*   **Streaming**: Supports translation of Server-Sent Events (SSE) for real-time responses, which is crucial for interactive CLI tools like `claude code`.

## Build Instructions

To build the proxy from source, ensure you have Go installed, then run:

```bash
go build -o gemma-claude-proxy
```

## Running the Proxy

The proxy is configured using environment variables:

*   `UPSTREAM_URL` **(Required)**: The URL of the upstream OpenAI-compatible `/chat/completions` endpoint.
*   `UPSTREAM_API_KEY` *(Optional)*: The API key required to authenticate with the upstream service.
*   `PORT` *(Optional)*: The port the proxy will listen on. Defaults to `8080`.

### Example: Using with Google AI Studio

If you want to use the model via Google AI Studio's OpenAI compatibility layer, run the proxy like this:

```bash
export UPSTREAM_URL="https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
export UPSTREAM_API_KEY="your_ai_studio_api_key_here"
export PORT="8080"

./gemma-claude-proxy
```

### Example: Using with a local server (e.g., vLLM or Ollama)

```bash
export UPSTREAM_URL="http://localhost:8000/v1/chat/completions"
export UPSTREAM_API_KEY="dummy_key_if_needed"

./gemma-claude-proxy
```

## Configuring Claude Code

Once the proxy is running, you need to tell `claude code` (or any other Anthropic client) to route its requests to your local proxy instead of Anthropic's servers.

Set the following environment variables in the terminal where you plan to run `claude code`:

```bash
# Provide a dummy API key to bypass client-side validation
export ANTHROPIC_API_KEY="dummy"

# Point the base URL to your local proxy
export ANTHROPIC_BASE_URL="http://localhost:8080"

# Run Claude Code
claude
```

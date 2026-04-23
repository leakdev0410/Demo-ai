# Gemma Claude Proxy

A proxy written in Golang that translates Anthropic's Messages API requests (used by Claude Code and other tools) to OpenAI's Chat Completions API format, specifically targeting the `gemma-4-31b-it` model.

## Features

*   **API Translation**: Converts Anthropic `/v1/messages` format to OpenAI `/v1/chat/completions` format.
*   **Response Translation**: Converts OpenAI format back to Anthropic format.
*   **Tool Calling**: Translates Claude's `tools` and `tool_use` to OpenAI's `functions` and `tool_calls`.
*   **Streaming**: Supports translation of Server-Sent Events (SSE) for real-time responses, crucial for tools like `claude code`.

## Usage

1.  Build the proxy:
    ```bash
    go build -o gemma-claude-proxy
    ```
2.  Run the proxy, specifying the upstream URL to your Gemma API server (e.g., an OpenAI-compatible endpoint like vLLM, Ollama, or a proxy):
    ```bash
    UPSTREAM_URL="http://localhost:8000/v1/chat/completions" ./gemma-claude-proxy
    ```
    You can optionally provide `UPSTREAM_API_KEY` and `PORT`.

3.  Configure `claude code` or other Anthropic clients to point to the proxy:
    ```bash
    export ANTHROPIC_API_KEY="dummy"
    export ANTHROPIC_BASE_URL="http://localhost:8080"
    ```

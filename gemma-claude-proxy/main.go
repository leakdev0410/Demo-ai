package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/user/gemma-claude-proxy/translator"
	"github.com/user/gemma-claude-proxy/types"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	upstreamURL := os.Getenv("UPSTREAM_URL")
	if upstreamURL == "" {
		log.Fatal("UPSTREAM_URL environment variable is required (e.g., http://localhost:11434/v1/chat/completions)")
	}

	apiKey := os.Getenv("UPSTREAM_API_KEY")

	http.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		var anthropicReq types.AnthropicRequest
		if err := json.Unmarshal(bodyBytes, &anthropicReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("Received request for model: %s, stream: %v", anthropicReq.Model, anthropicReq.Stream)

		// Target Gemma model
		targetModel := "gemma-4-31b-it"

		openAIReq, err := translator.TranslateRequest(&anthropicReq, targetModel)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to translate request: %v", err), http.StatusInternalServerError)
			return
		}

		openAIReqBytes, err := json.Marshal(openAIReq)
		if err != nil {
			http.Error(w, "Failed to marshal OpenAI request", http.StatusInternalServerError)
			return
		}

		proxyReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewBuffer(openAIReqBytes))
		if err != nil {
			http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
			return
		}

		proxyReq.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			proxyReq.Header.Set("Authorization", "Bearer "+apiKey)
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Upstream request failed: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Upstream error: %s", string(body))
			http.Error(w, fmt.Sprintf("Upstream returned %d", resp.StatusCode), resp.StatusCode)
			return
		}

		if anthropicReq.Stream {
			if err := translator.TranslateStream(w, resp.Body, anthropicReq.Model); err != nil {
				log.Printf("Streaming error: %v", err)
			}
		} else {
			respBodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read upstream response", http.StatusInternalServerError)
				return
			}

			var openAIResp types.OpenAIResponse
			if err := json.Unmarshal(respBodyBytes, &openAIResp); err != nil {
				http.Error(w, "Failed to parse upstream response", http.StatusInternalServerError)
				return
			}

			anthropicResp, err := translator.TranslateResponse(&openAIResp, anthropicReq.Model)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to translate response: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(anthropicResp)
		}
	})

	log.Printf("Proxy listening on :%s, forwarding to %s", port, upstreamURL)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

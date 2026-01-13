package openai

import (
	"net/http"
	"strings"
	"sync"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

var (
	openAIConfigMu          sync.RWMutex
	openAIBaseURL           = defaultOpenAIBaseURL
	openAIHTTPClientFactory = defaultOpenAIHTTPClientFactory
)

// SetOpenAIBaseURL overrides the OpenAI API base URL.
func SetOpenAIBaseURL(url string) {
	openAIConfigMu.Lock()
	defer openAIConfigMu.Unlock()

	if strings.TrimSpace(url) == "" {
		openAIBaseURL = defaultOpenAIBaseURL

		return
	}

	openAIBaseURL = url
}

// SetOpenAIHTTPClientFactory overrides the HTTP client factory used for OpenAI list models.
func SetOpenAIHTTPClientFactory(factory func() *http.Client) {
	openAIConfigMu.Lock()
	defer openAIConfigMu.Unlock()

	if factory == nil {
		openAIHTTPClientFactory = defaultOpenAIHTTPClientFactory

		return
	}

	openAIHTTPClientFactory = factory
}

func getOpenAIBaseURL() string {
	openAIConfigMu.RLock()
	defer openAIConfigMu.RUnlock()

	return openAIBaseURL
}

func getOpenAIHTTPClientFactory() func() *http.Client {
	openAIConfigMu.RLock()
	defer openAIConfigMu.RUnlock()

	return openAIHTTPClientFactory
}

func defaultOpenAIHTTPClientFactory() *http.Client {
	return &http.Client{Timeout: openAITimeout}
}

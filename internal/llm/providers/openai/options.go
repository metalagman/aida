package openai

import "net/http"

//go:generate go tool options-gen -from-struct=Options -out-filename=options_generated.go
type Options struct {
	apiKey string       `option:"mandatory"   validate:"required"`
	model  string       `option:"mandatory"   validate:"required"`
	client *http.Client `validate:"omitempty"`
}

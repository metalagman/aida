package aistudio

//go:generate go tool options-gen -from-struct=Options -out-filename=options_generated.go
type Options struct {
	apiKey string `validate:"omitempty"`
	model  string `option:"mandatory"   validate:"required"`
}

package templater_test

import (
	"testing"

	"github.com/metalagman/aida/internal/llm/templater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	t.Run("renders template", func(t *testing.T) {
		got, err := templater.Render("Hello {{.Name}}!", map[string]string{"Name": "Aida"})
		require.NoError(t, err)
		assert.Equal(t, "Hello Aida!", got)
	})

	t.Run("parse error", func(t *testing.T) {
		_, err := templater.Render("Hello {{", map[string]string{"Name": "Aida"})
		require.Error(t, err)
	})

	t.Run("execute error", func(t *testing.T) {
		type inner struct {
			Bar string
		}

		type data struct {
			Foo *inner
		}

		_, err := templater.Render("Value: {{.Foo.Bar}}", data{})
		require.Error(t, err)
	})
}

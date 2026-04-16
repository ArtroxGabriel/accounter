package web_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/ArtroxGabriel/accounter/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fsys    fs.FS
		wantErr string
	}{
		{
			name: "loads all html templates",
			fsys: fstest.MapFS{
				"templates/layout.html": {
					Data: []byte(`{{define "layout.html"}}{{template "content" .}}{{end}}`),
				},
				"templates/dashboard.html":     {Data: []byte(`{{define "content"}}ok{{end}}`)},
				"templates/expenses/list.html": {Data: []byte(`{{define "expense-list"}}list{{end}}`)},
				"templates/README.txt":         {Data: []byte("ignored")},
			},
		},
		{
			name:    "returns error when no html templates found",
			fsys:    fstest.MapFS{"templates/readme.md": {Data: []byte("none")}},
			wantErr: "no template files found",
		},
		{
			name: "returns error for invalid template syntax",
			fsys: fstest.MapFS{
				"templates/layout.html": {Data: []byte(`{{define "layout.html"}`)},
			},
			wantErr: "parsing templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := web.LoadTemplates(tt.fsys)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tmpl)
			assert.NotNil(t, tmpl.Lookup("layout.html"))
			assert.NotNil(t, tmpl.Lookup("content"))
			assert.NotNil(t, tmpl.Lookup("expense-list"))
		})
	}
}

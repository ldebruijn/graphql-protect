package schema

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempSchema(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.graphql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

const minimalSchema = `type Query { hello: String }`

func TestNewSchema_PathOnly(t *testing.T) {
	// Backward-compat: Config with only Path set (no Loader field) must load from the filesystem.
	path := writeTempSchema(t, minimalSchema)

	p, err := NewSchema(Config{Path: path}, slog.Default())

	require.NoError(t, err)
	assert.NotNil(t, p.Get())
}

func TestNewSchema_ExplicitLocalLoader(t *testing.T) {
	path := writeTempSchema(t, minimalSchema)

	cfg := Config{
		Path:   path,
		Loader: LoaderConfig{Type: "local"},
	}

	p, err := NewSchema(cfg, slog.Default())

	require.NoError(t, err)
	assert.NotNil(t, p.Get())
}

func TestNewSchema_UnknownLoaderTypeFallsBackToLocal(t *testing.T) {
	// Any unrecognised loader type falls back to local file loading.
	path := writeTempSchema(t, minimalSchema)

	cfg := Config{
		Path:   path,
		Loader: LoaderConfig{Type: "s3"},
	}

	p, err := NewSchema(cfg, slog.Default())

	require.NoError(t, err)
	assert.NotNil(t, p.Get())
}

func TestNewSchema_FileNotFound(t *testing.T) {
	cfg := Config{Path: "/nonexistent/schema.graphql"}

	_, err := NewSchema(cfg, slog.Default())

	assert.Error(t, err)
}

func TestNewSchema_InvalidGraphQL(t *testing.T) {
	path := writeTempSchema(t, "this is not valid graphql {{{")

	_, err := NewSchema(Config{Path: path}, slog.Default())

	assert.Error(t, err)
}

func TestSchemaGetNoRaceWithReload(t *testing.T) {
	path := writeTempSchema(t, minimalSchema)

	cfg := Config{
		Path: path,
		AutoReload: struct {
			Enabled  bool          `yaml:"enabled"`
			Interval time.Duration `yaml:"interval"`
		}{
			Enabled: false, // we drive reloads manually below
		},
	}

	p, err := NewSchema(cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	// Writer goroutine: repeatedly calls loadFromFs (simulating what the reload ticker does).
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = p.loadFromFs()
			}
		}
	}()

	// Reader goroutines: repeatedly call Get() concurrently with the writer.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				_ = p.Get()
			}
		}()
	}

	// Let readers finish, then stop the writer.
	// (wg.Wait covers all goroutines, but we stop the writer after a short time)
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}

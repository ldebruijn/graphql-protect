package schema

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
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

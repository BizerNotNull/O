package main

import (
	json "encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkRead(b *testing.B) {
	for _, size := range []int{4 << 10, 64 << 10, 1 << 20} {
		for _, benchmark := range []struct {
			name string
			read func(T) string
		}{{"before", readBefore}, {"after", run}} {
			b.Run(byteCount(size)+"/"+benchmark.name, func(b *testing.B) {
				dir := b.TempDir()
				name := "fixture"
				if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Repeat("x", size)), 0o644); err != nil {
					b.Fatal(err)
				}
				oldRoot := root
				root = dir
				b.Cleanup(func() { root = oldRoot })
				call := T{Function: F{Name: "read", Arguments: `{"path":"fixture"}`}}
				b.ReportAllocs()
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if got := benchmark.read(call); len(got) != size {
						b.Fatalf("read %d bytes, want %d", len(got), size)
					}
				}
			})
		}
	}
}

// readBefore preserves the previous read path so benchmark results remain
// reproducible without checking out an older commit.
func readBefore(t T) string {
	a := map[string]string{}
	_ = json.Unmarshal([]byte(t.Function.Arguments), &a)
	p, err := path(a["path"])
	if err != nil {
		return "error: " + err.Error()
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "error: " + err.Error() + "\n" + string(b)
	}
	return string(b)
}

func byteCount(n int) string {
	if n >= 1<<20 {
		return "1MiB"
	}
	return strconv.Itoa(n/(1<<10)) + "KiB"
}

func TestRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fixture"), []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldRoot := root
	root = dir
	t.Cleanup(func() { root = oldRoot })

	if got := run(T{Function: F{Name: "read", Arguments: `{"path":"fixture"}`}}); got != "contents" {
		t.Fatalf("read returned %q", got)
	}
	if got := run(T{Function: F{Name: "read", Arguments: `{"path":"../outside"}`}}); !strings.HasPrefix(got, "error: outside project") {
		t.Fatalf("outside read returned %q", got)
	}
}

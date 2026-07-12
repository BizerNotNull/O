package main

import (
	"bufio"
	"bytes"
	"context"
	j "encoding/json"
	"fmt"
	"io"
	"net/http"
	o "os"
	"os/exec"
	f "path/filepath"
	"runtime"
	s "strings"
	"time"
	"unsafe"
)

var url, key, model, root string

type A = map[string]any

type F struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type T struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function F      `json:"function"`
}
type M struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolCalls  []T    `json:"tool_calls,omitempty"`
}

type arguments struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Command string `json:"command"`
}

func out(p string) bool {
	r, e := f.Rel(root, p)
	return e != nil || r == ".." || s.HasPrefix(r, ".."+string(o.PathSeparator))
}

func path(x string) (string, error) {
	p, e := f.Abs(f.Join(root, x))
	if e != nil || out(p) {
		return "", fmt.Errorf("outside project")
	}
	for q := p; ; q = f.Dir(q) {
		if x, e := f.EvalSymlinks(q); e == nil {
			if out(x) {
				return "", fmt.Errorf("outside project")
			}
			break
		}
	}
	return p, nil
}

func run(t T) string {
	var a arguments
	j.Unmarshal([]byte(t.Function.Arguments), &a)
	n := t.Function.Name
	p, e := path(a.Path)
	var b []byte
	switch n {
	case "read":
		if e == nil {
			b, e = o.ReadFile(p)
		}
	case "write":
		if e == nil {
			if e = o.MkdirAll(f.Dir(p), 0755); e == nil {
				e = o.WriteFile(p, []byte(a.Content), 0644)
			}
		}
	case "edit":
		if e == nil {
			b, e = o.ReadFile(p)
		}
		if e == nil && !bytes.Contains(b, []byte(a.Old)) {
			e = fmt.Errorf("old text not found")
		}
		if e == nil {
			e = o.WriteFile(p, bytes.Replace(b, []byte(a.Old), []byte(a.New), 1), 0644)
		}
	case "bash":
		c, x := context.WithTimeout(context.Background(), 30*time.Second)
		defer x()
		v := []string{"bash", "-lc", a.Command}
		if runtime.GOOS == "windows" {
			v = []string{"powershell", "-NoProfile", "-Command", a.Command}
		}
		q := exec.CommandContext(c, v[0], v[1:]...)
		q.Dir = root
		b, e = q.CombinedOutput()
	default:
		e = fmt.Errorf("unknown tool")
	}
	if e != nil {
		return "error: " + e.Error() + "\n" + string(b)
	}
	if n == "read" || n == "bash" {
		// The returned string owns b's backing array, so avoiding a second copy is
		// safe as long as the buffer is not mutated after this point.
		return unsafe.String(unsafe.SliceData(b), len(b))
	}
	return "ok"
}

func schema(n string, p ...string) A {
	x := A{}
	for _, k := range p {
		x[k] = map[string]string{"type": "string"}
	}
	return A{"type": "function", "function": A{"name": n, "parameters": A{"type": "object", "properties": x, "required": p}}}
}

var tools = []A{
	schema("read", "path"),
	schema("write", "path", "content"),
	schema("edit", "path", "old", "new"),
	schema("bash", "command"),
}

func chat(m []M) (M, error) {
	b, _ := j.Marshal(A{"model": model, "messages": m, "tools": tools})
	r, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	r.Header = map[string][]string{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	x, e := http.DefaultClient.Do(r)
	if e != nil {
		return M{}, e
	}
	defer x.Body.Close()
	b, _ = io.ReadAll(x.Body)
	var z struct{ Choices []struct{ Message M } }
	j.Unmarshal(b, &z)
	return z.Choices[0].Message, nil
}

func prompt() string {
	x := "You are a code agent. Use tools only inside " + root + ". Never access anything outside it."
	for _, n := range []string{"AGENT.md", "SKILL.md"} {
		if b, e := o.ReadFile(f.Join(root, n)); e == nil {
			x += "\n\n# " + n + "\n" + string(b)
		}
	}
	return x
}

func logo() {
	for y := -4; y <= 4; y++ {
		for x := -9; x <= 9; x++ {
			c := " "
			if d := x*x + 4*y*y - 64; d > -12 && d < 12 {
				c = "*"
			}
			fmt.Print(c)
		}
		fmt.Println()
	}
}

func main() {
	root, _ = o.Getwd()
	if r, e := f.EvalSymlinks(root); e == nil {
		root = r
	}
	logo()
	m := []M{{Role: "system", Content: prompt()}}
	q := bufio.NewScanner(o.Stdin)
	for fmt.Print("> "); q.Scan(); fmt.Print("> ") {
		x := s.TrimSpace(q.Text())
		if x == "" {
			continue
		}
		if x == "/exit" || x == "/quit" {
			return
		}
		m = append(m, M{Role: "user", Content: x})
		for {
			a, e := chat(m)
			if e != nil {
				fmt.Fprintln(o.Stderr, e)
				break
			}
			m = append(m, a)
			if len(a.ToolCalls) == 0 {
				fmt.Println(a.Content)
				break
			}
			for _, t := range a.ToolCalls {
				m = append(m, M{Role: "tool", Content: run(t), ToolCallID: t.ID})
			}
		}
	}
}

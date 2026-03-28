package utils

import (
	"bytes"
	"testing"
)

func TestLimitedWriter(t *testing.T) {
	t.Run("writing within limit - all bytes written", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: 100}
		p := []byte("hello")
		n, err := lw.Write(p)
		if err != nil {
			t.Fatalf("Write() unexpected error: %v", err)
		}
		if n != len(p) {
			t.Errorf("Write() returned n=%d, want %d", n, len(p))
		}
		if got := buf.String(); got != "hello" {
			t.Errorf("underlying writer contains %q, want %q", got, "hello")
		}
		if lw.N != 95 {
			t.Errorf("remaining N=%d, want 95", lw.N)
		}
	})

	t.Run("writing exactly at limit - all bytes written", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: 5}
		p := []byte("hello")
		n, err := lw.Write(p)
		if err != nil {
			t.Fatalf("Write() unexpected error: %v", err)
		}
		if n != len(p) {
			t.Errorf("Write() returned n=%d, want %d", n, len(p))
		}
		if got := buf.String(); got != "hello" {
			t.Errorf("underlying writer contains %q, want %q", got, "hello")
		}
		if lw.N != 0 {
			t.Errorf("remaining N=%d, want 0", lw.N)
		}
	})

	t.Run("writing beyond limit - truncated silently, reports full len(p)", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: 3}
		p := []byte("hello")
		n, err := lw.Write(p)
		if err != nil {
			t.Fatalf("Write() unexpected error: %v", err)
		}
		if n != len(p) {
			t.Errorf("Write() returned n=%d, want %d (full original length)", n, len(p))
		}
		if got := buf.String(); got != "hel" {
			t.Errorf("underlying writer contains %q, want %q (truncated)", got, "hel")
		}
		if lw.N != 0 {
			t.Errorf("remaining N=%d, want 0", lw.N)
		}
	})

	t.Run("multiple writes crossing the limit boundary", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: 7}

		// First write: 5 bytes, all within limit
		n1, err := lw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("first Write() unexpected error: %v", err)
		}
		if n1 != 5 {
			t.Errorf("first Write() returned n=%d, want 5", n1)
		}

		// Second write: 5 bytes, only 2 fit (limit is 7, 5 used)
		n2, err := lw.Write([]byte("world"))
		if err != nil {
			t.Fatalf("second Write() unexpected error: %v", err)
		}
		if n2 != 5 {
			t.Errorf("second Write() returned n=%d, want 5 (full original length)", n2)
		}

		if got := buf.String(); got != "hellowo" {
			t.Errorf("underlying writer contains %q, want %q", got, "hellowo")
		}
		if lw.N != 0 {
			t.Errorf("remaining N=%d, want 0", lw.N)
		}

		// Third write: limit exhausted, nothing written
		n3, err := lw.Write([]byte("!"))
		if err != nil {
			t.Fatalf("third Write() unexpected error: %v", err)
		}
		if n3 != 1 {
			t.Errorf("third Write() returned n=%d, want 1 (full original length)", n3)
		}
		if got := buf.String(); got != "hellowo" {
			t.Errorf("underlying writer contains %q after exhausted write, want %q", got, "hellowo")
		}
	})

	t.Run("zero limit - nothing written", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: 0}
		p := []byte("hello")
		n, err := lw.Write(p)
		if err != nil {
			t.Fatalf("Write() unexpected error: %v", err)
		}
		if n != len(p) {
			t.Errorf("Write() returned n=%d, want %d (full original length)", n, len(p))
		}
		if buf.Len() != 0 {
			t.Errorf("underlying writer should be empty, got %q", buf.String())
		}
	})

	t.Run("negative limit - nothing written", func(t *testing.T) {
		var buf bytes.Buffer
		lw := &LimitedWriter{W: &buf, N: -10}
		p := []byte("hello")
		n, err := lw.Write(p)
		if err != nil {
			t.Fatalf("Write() unexpected error: %v", err)
		}
		if n != len(p) {
			t.Errorf("Write() returned n=%d, want %d (full original length)", n, len(p))
		}
		if buf.Len() != 0 {
			t.Errorf("underlying writer should be empty, got %q", buf.String())
		}
	})
}

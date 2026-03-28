package utils

import "io"

// LimitedWriter wraps an io.Writer and silently discards writes that would
// exceed the configured byte limit. This prevents unbounded memory growth
// when capturing output from external processes.
//
// After the limit is reached, Write still reports consuming all bytes so that
// callers such as io.Copy do not treat a truncated write as io.ErrShortWrite.
type LimitedWriter struct {
	W io.Writer // underlying writer
	N int64     // remaining bytes allowed
}

// Write writes p to the underlying writer, truncating silently once the byte
// limit is exhausted. It always returns len(p), nil to avoid spurious
// io.ErrShortWrite errors in callers like io.Copy.
func (lw *LimitedWriter) Write(p []byte) (int, error) {
	if lw.N <= 0 {
		return len(p), nil // silently discard
	}
	orig := len(p)
	if int64(len(p)) > lw.N {
		p = p[:lw.N]
	}
	n, err := lw.W.Write(p)
	lw.N -= int64(n)
	if err != nil {
		return n, err
	}
	// Report the original length to callers so that io.Copy does not
	// treat a truncated write as io.ErrShortWrite.
	return orig, nil
}

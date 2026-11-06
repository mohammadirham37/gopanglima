package httpx

import "net/http"

// Response wraps http.ResponseWriter and records the status code, the
// number of body bytes written, and whether the response has been
// committed (final status line and headers sent).
//
// Response supports http.ResponseController: it implements FlushError
// directly and exposes the underlying writer through Unwrap for other
// features such as deadlines.
type Response struct {
	w         http.ResponseWriter
	status    int
	size      int64
	committed bool
}

func newResponse(w http.ResponseWriter) *Response {
	return &Response{w: w}
}

// Header returns the header map that will be sent with WriteHeader.
func (r *Response) Header() http.Header { return r.w.Header() }

// WriteHeader sends the status line and headers.
//
// Informational 1xx codes (except 101 Switching Protocols) are forwarded
// without committing the response, so a final status can still follow.
// Calls after the response has been committed are ignored, matching
// net/http semantics without emitting the "superfluous WriteHeader" log
// line.
func (r *Response) WriteHeader(code int) {
	if r.committed {
		return
	}
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		r.w.WriteHeader(code)
		return
	}
	r.status = code
	r.committed = true
	r.w.WriteHeader(code)
}

// Write writes body bytes, committing a 200 status first if needed.
func (r *Response) Write(b []byte) (int, error) {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.w.Write(b)
	r.size += int64(n)
	return n, err
}

// FlushError commits a 200 status if nothing has been written yet and then
// flushes buffered data to the client. It is used by
// http.ResponseController.Flush.
func (r *Response) FlushError() error {
	if !r.committed {
		r.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(r.w).Flush()
}

// Unwrap returns the underlying http.ResponseWriter for
// http.ResponseController.
func (r *Response) Unwrap() http.ResponseWriter { return r.w }

// Status returns the committed status code, or 0 if nothing was written.
func (r *Response) Status() int { return r.status }

// Size returns the number of body bytes written.
func (r *Response) Size() int64 { return r.size }

// Committed reports whether the final status line and headers have been
// sent.
func (r *Response) Committed() bool { return r.committed }

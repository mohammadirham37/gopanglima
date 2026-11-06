package httpx

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeGracefulShutdown(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	r := NewRouter()
	r.Get("/slow", func(c *Context) error {
		close(started)
		<-release
		// The request context must survive the server context's cancellation.
		if err := c.Context().Err(); err != nil {
			return err
		}
		return c.String(200, "done")
	})
	h := mustBuild(t, r)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, ln, h, ServerConfig{ShutdownTimeout: 5 * time.Second}, nil) }()

	type result struct {
		body string
		err  error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/slow")
		if err != nil {
			resCh <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		resCh <- result{body: string(b), err: err}
	}()

	<-started
	cancel()
	time.Sleep(50 * time.Millisecond) // let Shutdown begin
	close(release)

	res := <-resCh
	if res.err != nil || res.body != "done" {
		t.Fatalf("in-flight request: body=%q err=%v", res.body, res.err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve returned %v", err)
	}
}

func TestServeShutdownTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	r := NewRouter()
	r.Get("/hang", func(c *Context) error {
		close(started)
		<-release
		return nil
	})
	h := mustBuild(t, r)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, ln, h, ServerConfig{ShutdownTimeout: 50 * time.Millisecond}, nil) }()
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/hang")
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	cancel()
	if err := <-served; err == nil {
		t.Fatal("expected shutdown timeout error")
	}
}

func TestServeReturnsListenerErrors(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln.Close()
	if err := Serve(context.Background(), ln, http.NotFoundHandler(), ServerConfig{}, nil); err == nil {
		t.Fatal("expected error from closed listener")
	}
}

func TestServerConfigDefaults(t *testing.T) {
	got := ServerConfig{ReadTimeout: time.Second}.withDefaults()
	want := DefaultServerConfig()
	want.ReadTimeout = time.Second
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestServerNegativeTimeouts(t *testing.T) {
	if got := disabledIfNegative(-time.Second); got != 0 {
		t.Fatalf("negative timeout should disable, got %v", got)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	err = Serve(context.Background(), ln, http.NotFoundHandler(), ServerConfig{ShutdownTimeout: -1}, nil)
	if err == nil {
		t.Fatal("negative ShutdownTimeout should be rejected")
	}
}

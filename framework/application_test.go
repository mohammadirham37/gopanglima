package framework

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mohammadirham37/gopanglima/framework/foundation"
	"github.com/mohammadirham37/gopanglima/framework/logging"
)

func quietApp(opts ...Option) *Application {
	return New(append([]Option{WithLogger(logging.Discard()), WithEnv(foundation.NewEnv(nil, nil))}, opts...)...)
}

func TestNewDefaults(t *testing.T) {
	app := quietApp()
	if app.Name() != "goPanglima" || app.Environment() != "production" || !app.IsProduction() || app.Debug() {
		t.Fatalf("unexpected defaults: name=%q env=%q debug=%v", app.Name(), app.Environment(), app.Debug())
	}
	if app.Config() == nil || app.Container() == nil || app.Router() == nil || app.Logger() == nil || app.Env() == nil {
		t.Fatal("accessors must not return nil")
	}
	if err := app.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
}

func TestConfigFromEnvAndOverrides(t *testing.T) {
	env := foundation.NewEnv(map[string]string{
		"APP_NAME": "Shop", "APP_ENV": "local", "APP_DEBUG": "true", "LOG_LEVEL": "debug",
	}, nil)
	app := New(WithEnv(env), WithLogger(logging.Discard()),
		WithConfig(map[string]any{"app.name": "Override", "custom.key": 1}),
		WithConfig(map[string]any{"custom.other": "x"}))
	if app.Name() != "Override" || app.Environment() != "local" || !app.Debug() || app.IsProduction() {
		t.Fatalf("name=%q env=%q debug=%v", app.Name(), app.Environment(), app.Debug())
	}
	if app.Config().String("log.level", "") != "debug" || app.Config().Int("custom.key", 0) != 1 ||
		app.Config().String("custom.other", "") != "x" {
		t.Fatalf("config keys: %v", app.Config().Keys())
	}
}

func TestWithEnvFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, ".env")
	if err := os.WriteFile(good, []byte("APP_NAME=FromFile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := New(WithEnvFile(good), WithLogger(logging.Discard()))
	if app.Name() != "FromFile" && os.Getenv("APP_NAME") == "" {
		t.Fatalf("Name = %q", app.Name())
	}

	bad := filepath.Join(dir, "bad.env")
	if err := os.WriteFile(bad, []byte("NOT VALID\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app = New(WithEnvFile(bad), WithLogger(logging.Discard()))
	err := app.Bootstrap(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad.env") {
		t.Fatalf("expected init error mentioning file, got %v", err)
	}
	// The error is sticky.
	if err2 := app.Bootstrap(context.Background()); err2 == nil {
		t.Fatal("second Bootstrap should return the same error")
	}
	if _, err := app.Handler(); err == nil {
		t.Fatal("Handler should surface init errors")
	}
}

func TestInitErrorsAreReported(t *testing.T) {
	cases := map[string]*Application{
		"bad APP_DEBUG":  New(WithEnv(foundation.NewEnv(map[string]string{"APP_DEBUG": "maybe"}, nil)), WithLogger(logging.Discard())),
		"bad LOG_LEVEL":  New(WithEnv(foundation.NewEnv(map[string]string{"LOG_LEVEL": "loud"}, nil))),
		"bad LOG_FORMAT": New(WithEnv(foundation.NewEnv(map[string]string{"LOG_FORMAT": "xml"}, nil))),
		"bad body limit": quietApp(WithMaxBodyBytes(0)),
		"nil provider":   quietApp(WithProviders(nil)),
	}
	for name, app := range cases {
		t.Run(name, func(t *testing.T) {
			if err := app.Bootstrap(context.Background()); err == nil {
				t.Fatal("expected Bootstrap error")
			}
		})
	}
}

type recordingProvider struct {
	name    string
	log     *[]string
	regErr  error
	bootErr error
}

func (p *recordingProvider) Register(*Application) error {
	*p.log = append(*p.log, "register:"+p.name)
	return p.regErr
}

func (p *recordingProvider) Boot(context.Context, *Application) error {
	*p.log = append(*p.log, "boot:"+p.name)
	return p.bootErr
}

func TestProviderLifecycleOrder(t *testing.T) {
	var log []string
	a := &recordingProvider{name: "a", log: &log}
	b := &recordingProvider{name: "b", log: &log}
	c := ProviderFunc(func(*Application) error { log = append(log, "register:c"); return nil })

	app := quietApp(WithProviders(a, b))
	if err := app.Register(c); err != nil {
		t.Fatal(err)
	}
	if err := app.Bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := app.Bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "register:a register:b register:c boot:a boot:b"
	if got := strings.Join(log, " "); got != want {
		t.Fatalf("order:\n got  %s\n want %s", got, want)
	}
	if err := app.Register(&recordingProvider{name: "late", log: &log}); !errors.Is(err, ErrAlreadyBooted) {
		t.Fatalf("late Register = %v", err)
	}
}

func TestProviderErrors(t *testing.T) {
	var log []string
	regFail := &recordingProvider{name: "x", log: &log, regErr: errors.New("reg boom")}
	if err := quietApp().Register(regFail); err == nil || !strings.Contains(err.Error(), "reg boom") {
		t.Fatalf("Register error = %v", err)
	}

	log = nil
	app := quietApp(WithProviders(
		&recordingProvider{name: "a", log: &log, bootErr: errors.New("boot boom")},
		&recordingProvider{name: "b", log: &log},
	))
	err := app.Bootstrap(context.Background())
	if err == nil || !strings.Contains(err.Error(), "boot boom") {
		t.Fatalf("Bootstrap error = %v", err)
	}
	if strings.Contains(strings.Join(log, " "), "boot:b") {
		t.Fatal("boot must stop at the first error")
	}
}

func TestBootstrapHonorsContext(t *testing.T) {
	var log []string
	app := quietApp(WithProviders(&recordingProvider{name: "a", log: &log}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.Bootstrap(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Bootstrap = %v", err)
	}
}

func TestShutdownHooks(t *testing.T) {
	var order []string
	app := quietApp()
	app.OnShutdown(func(context.Context) error { order = append(order, "first"); return errors.New("e1") })
	app.OnShutdown(nil)
	app.OnShutdown(func(context.Context) error { order = append(order, "second"); return errors.New("e2") })

	err := app.Shutdown(context.Background())
	if got := strings.Join(order, " "); got != "second first" {
		t.Fatalf("hook order = %s", got)
	}
	if err == nil || !strings.Contains(err.Error(), "e1") || !strings.Contains(err.Error(), "e2") {
		t.Fatalf("errors not joined: %v", err)
	}
	if err2 := app.Shutdown(context.Background()); err2 == nil || len(order) != 2 {
		t.Fatal("Shutdown must run once and return the first result")
	}
	if err := app.Bootstrap(context.Background()); !errors.Is(err, ErrShutdown) {
		t.Fatalf("Bootstrap after Shutdown = %v", err)
	}
}

func TestHandlerServesRoutes(t *testing.T) {
	app := quietApp()
	app.Router().Get("/hello/{name}", func(c *Context) error {
		return c.JSON(http.StatusOK, M{"hello": c.Param("name")})
	})
	app.Router().Get("/missing", func(c *Context) error { return NotFound("User not found") })
	app.Router().Get("/panic", func(c *Context) error { panic("boom") })

	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/hello/ana", nil))
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"hello":"ana"}` {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("default RequestID middleware not installed")
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/missing", nil))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), "User not found") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))
	if rec.Code != 500 {
		t.Fatalf("panic should become 500, got %d", rec.Code)
	}
}

func TestWithoutDefaultMiddlewareAndCustomErrorHandler(t *testing.T) {
	app := quietApp(WithoutDefaultMiddleware(), WithErrorHandler(func(c *Context, err error) {
		_ = c.String(599, "custom")
	}))
	app.Router().Get("/", func(c *Context) error { return errors.New("x") })
	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 599 || rec.Header().Get("X-Request-ID") != "" {
		t.Fatalf("got %d headers=%v", rec.Code, rec.Header())
	}
}

func TestHandlerReportsRouteErrors(t *testing.T) {
	app := quietApp()
	app.Router().Get("bad", func(c *Context) error { return nil })
	if _, err := app.Handler(); err == nil {
		t.Fatal("expected route error")
	}
}

func TestDebugModeFlowsToHandler(t *testing.T) {
	app := quietApp(WithConfig(map[string]any{"app.debug": true}))
	app.Router().Get("/", func(c *Context) error { return errors.New("internal detail") })
	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "internal detail") {
		t.Fatalf("debug detail missing: %q", rec.Body.String())
	}
}

func TestServeLifecycle(t *testing.T) {
	var logs bytes.Buffer
	app := New(WithEnv(foundation.NewEnv(nil, nil)), WithLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	app.Router().Get("/", func(c *Context) error { return c.String(200, "up") })
	hookRan := make(chan struct{})
	app.OnShutdown(func(context.Context) error { close(hookRan); return nil })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Serve(ctx, ln) }()

	resp, err := waitGet("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "up" {
		t.Fatalf("body = %q", body)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve = %v", err)
	}
	select {
	case <-hookRan:
	default:
		t.Fatal("shutdown hook did not run")
	}
	if !strings.Contains(logs.String(), "server started") || !strings.Contains(logs.String(), "server stopped") {
		t.Fatalf("lifecycle logs missing: %s", logs.String())
	}

	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	if err := app.Serve(context.Background(), ln2); err == nil {
		t.Fatal("Serve after shutdown should fail")
	}
}

func TestServeRunsShutdownHooksOnBootFailure(t *testing.T) {
	app := quietApp(WithProviders(ProviderFunc(func(a *Application) error {
		a.OnShutdown(func(context.Context) error { return errors.New("hook ran") })
		return nil
	}), &recordingProvider{name: "x", log: new([]string), bootErr: errors.New("boot failed")}))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	err = app.Serve(context.Background(), ln)
	if err == nil || !strings.Contains(err.Error(), "boot failed") || !strings.Contains(err.Error(), "hook ran") {
		t.Fatalf("Serve = %v", err)
	}
}

func TestHTTPErrorHelpers(t *testing.T) {
	cases := map[int]*HTTPError{
		400: BadRequest(""), 401: Unauthorized(""), 403: Forbidden(""), 404: NotFound(""),
		409: Conflict(""), 422: UnprocessableEntity(""), 429: TooManyRequests(""),
		500: InternalServerError(""), 418: NewHTTPError(418, ""),
	}
	for code, e := range cases {
		if e.Code != code || e.Message != http.StatusText(code) {
			t.Errorf("%d: %+v", code, e)
		}
	}
}

func waitGet(url string) (*http.Response, error) {
	var lastErr error
	for i := 0; i < 50; i++ {
		resp, err := http.Get(url)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(20 * time.Millisecond)
	}
	return nil, lastErr
}

type slowBooter struct{ release chan struct{} }

func (slowBooter) Register(*Application) error { return nil }
func (s slowBooter) Boot(ctx context.Context, app *Application) error {
	<-s.release
	app.Router().Get("/late", func(c *Context) error { return c.String(200, "late") })
	return nil
}

func TestConcurrentBootstrapWaitsForFirst(t *testing.T) {
	release := make(chan struct{})
	app := quietApp(WithProviders(slowBooter{release: release}))
	first := make(chan error, 1)
	go func() { first <- app.Bootstrap(context.Background()) }()

	// Wait until the first Bootstrap is in progress.
	for {
		app.mu.Lock()
		started := app.bootDone != nil
		app.mu.Unlock()
		if started {
			break
		}
		time.Sleep(time.Millisecond)
	}

	second := make(chan error, 1)
	go func() { second <- app.Bootstrap(context.Background()) }()
	select {
	case <-second:
		t.Fatal("second Bootstrap returned before the first finished")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app2 := quietApp(WithProviders(slowBooter{release: make(chan struct{})}))
	go app2.Bootstrap(context.Background())
	for {
		app2.mu.Lock()
		started := app2.bootDone != nil
		app2.mu.Unlock()
		if started {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err := app2.Bootstrap(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting Bootstrap should honor ctx, got %v", err)
	}
}

func TestConcurrentShutdownWaitsForHooks(t *testing.T) {
	app := quietApp()
	hookStarted := make(chan struct{})
	release := make(chan struct{})
	app.OnShutdown(func(context.Context) error {
		close(hookStarted)
		<-release
		return errors.New("hook error")
	})

	first := make(chan error, 1)
	go func() { first <- app.Shutdown(context.Background()) }()
	<-hookStarted

	second := make(chan error, 1)
	go func() { second <- app.Shutdown(context.Background()) }()
	select {
	case <-second:
		t.Fatal("second Shutdown returned while hooks were running")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-first; err == nil {
		t.Fatal("first Shutdown should return the hook error")
	}
	if err := <-second; err == nil || !strings.Contains(err.Error(), "hook error") {
		t.Fatalf("second Shutdown = %v", err)
	}
}

func TestShutdownStopsServeBeforeHooks(t *testing.T) {
	app := quietApp()
	inFlight := make(chan struct{})
	finishRequest := make(chan struct{})
	var requestDone, hookRan bool
	app.Router().Get("/slow", func(c *Context) error {
		close(inFlight)
		<-finishRequest
		requestDone = true
		return c.String(200, "ok")
	})
	app.OnShutdown(func(context.Context) error {
		if !requestDone {
			return errors.New("hook ran before in-flight request finished")
		}
		hookRan = true
		return nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- app.Serve(context.Background(), ln) }()

	go func() {
		resp, err := waitGet("http://" + ln.Addr().String() + "/slow")
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-inFlight

	shut := make(chan error, 1)
	go func() { shut <- app.Shutdown(context.Background()) }()
	time.Sleep(50 * time.Millisecond)
	close(finishRequest)

	if err := <-shut; err != nil {
		t.Fatalf("Shutdown = %v", err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve = %v", err)
	}
	if !hookRan {
		t.Fatal("hook did not run")
	}
}

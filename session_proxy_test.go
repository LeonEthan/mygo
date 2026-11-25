package mygo

import (
	"context"
	"net/http"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestSessionProxyOptions(t *testing.T) {
	config := SessionProxyConfig{HTTPProxy: "http://127.0.0.1:8080"}
	s, err := NewProxiedSession(t.Name(), config)
	if err != nil {
		t.Fatal(err)
	}
	same, err := NewProxiedSession(t.Name(), config)
	if err != nil || same != s {
		t.Fatal("named session identity", err)
	}
	config.Password = "changed"
	if _, err = NewProxiedSession(t.Name(), config); err == nil {
		t.Fatal("reconfigured an existing session")
	}
	a, err := NewProxiedSession("", config)
	if err != nil || !a.IsEphemeral() {
		t.Fatal("ephemeral session", err)
	}
	config.HTTPProxy = "http://127.0.0.1:8080 --bad"
	if _, err = NewProxiedSession("", config); err == nil {
		t.Fatal("accepted invalid endpoint")
	}
}

func TestSchemeReservationReleasedAfterHandler(t *testing.T) {
	if err := Protocol.SetBodyLimits(SchemeBodyLimits{MaxInFlightRequestBodyBytes: 1}); err != nil {
		t.Fatal(err)
	}
	defer Protocol.SetBodyLimits(SchemeBodyLimits{})
	w, _ := testWindow(t, WindowOptions{})
	for _, mode := range []string{"finish", "panic", "cancel"} {
		ctx, cancel := context.WithCancel(context.Background())
		req := &platform.SchemeRequest{Context: ctx, Method: "GET", URL: "test://localhost/", Responder: &recorder{finished: make(chan struct{})}}
		if status := req.BodyReservation.Reserve(1); status != 0 {
			t.Fatal(mode, status)
		}
		serveScheme(w, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			switch mode {
			case "panic":
				panic("handler failed")
			case "cancel":
				cancel()
				<-r.Context().Done()
			}
		}), req)
		cancel()
	}
	var next platform.SchemeBodyReservation
	if status := next.Reserve(1); status != 0 {
		t.Fatal(status)
	}
	next.Release()
}

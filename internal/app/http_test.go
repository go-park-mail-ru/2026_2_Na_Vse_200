package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/config"

	"github.com/rs/zerolog"
)

func TestRunWithoutDatabase(t *testing.T) {
	if err := Run(context.Background(), config.Config{}, zerolog.Nop()); err == nil {
		t.Fatal("missing DSN accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, config.Config{}, zerolog.Nop()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestServeWaitsForActiveRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	shutdownStarted := make(chan struct{})
	address := make(chan string, 1)
	server := newServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusNoContent)
	}))
	server.BaseContext = func(l net.Listener) context.Context {
		address <- l.Addr().String()
		return context.Background()
	}
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer server.Close()
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, 2*time.Second, zerolog.Nop())
	}()
	var addr string
	select {
	case addr = <-address:
	case err := <-done:
		t.Fatalf("server failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start")
	}
	response := make(chan error, 1)
	client := &http.Client{Timeout: 3 * time.Second}
	go func() {
		resp, err := client.Get("http://" + addr)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				err = errors.New("unexpected status")
			}
		}
		response <- err
	}()
	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case <-shutdownStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-done:
		t.Fatalf("server stopped before request completed: %v", err)
	default:
	}
	close(releaseRequest)
	select {
	case err := <-response:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request did not finish")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServeInvalidAddress(t *testing.T) {
	server := newServer("invalid::address", http.NotFoundHandler())
	logger := zerolog.Nop()
	if err := serve(context.Background(), server, time.Second, logger); err == nil {
		t.Fatal("invalid address accepted")
	}
}

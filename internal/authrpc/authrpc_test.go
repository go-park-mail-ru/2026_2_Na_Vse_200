package authrpc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "2026_2_Na_Vse_200/api/auth/v1"
	"2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/rpcmetrics"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type sessionStore struct {
	mu       sync.Mutex
	sessions map[models.SessionID]models.Session
}

func (s *sessionStore) Create(_ context.Context, session models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *sessionStore) GetByID(_ context.Context, id models.SessionID) (models.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return models.Session{}, models.ErrSessionNotFound
	}
	return session, nil
}

func (s *sessionStore) Delete(_ context.Context, id models.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func TestSessionRPC(t *testing.T) {
	const (
		owner     = models.UserID("f27b54ad-6cc6-4779-af11-2123653e9cdc")
		other     = models.UserID("578d67e2-1955-4711-9ac8-a41c5f09243f")
		sessionID = models.SessionID("d860d412-bcd2-4b7e-8d99-a73ff9a0ab06")
		expiredID = models.SessionID("0e6d46a1-566b-43a1-b948-8ee6182614d1")
		requestID = "5c77bbb2-2a25-4f12-ae4c-e9a7861be5c8"
	)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	store := &sessionStore{sessions: map[models.SessionID]models.Session{
		expiredID: {ID: expiredID, UserID: owner, ExpiresAt: now},
	}}
	manager := authusecase.NewSessionManager(store, func() models.SessionID { return sessionID }, func() time.Time { return now }, time.Hour)
	listener := bufconn.Listen(1024 * 1024)
	clientMetrics := rpcmetrics.New("music-api")
	serverMetrics := rpcmetrics.New("music-auth")
	var logs strings.Builder
	server := grpc.NewServer(grpc.UnaryInterceptor(serverMetrics.Server(zerolog.New(&logs))))
	authv1.RegisterAuthServiceServer(server, NewServer(manager))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///auth-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithUnaryInterceptor(clientMetrics.Client),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := NewClient(conn)
	ctx, cancel := context.WithTimeout(logger.WithRequestID(t.Context(), requestID), 5*time.Second)
	defer cancel()

	session, err := client.CreateSession(ctx, owner)
	if err != nil || session.ID != sessionID || session.UserID != owner || !session.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected issued session: %+v, err=%v", session, err)
	}
	if actual, err := client.GetSession(ctx, sessionID); err != nil || actual != session {
		t.Fatalf("unexpected stored session: %+v, err=%v", actual, err)
	}
	if err := client.CheckProfileAccess(ctx, sessionID, owner); err != nil {
		t.Fatalf("owner denied: %v", err)
	}
	if err := client.CheckProfileAccess(ctx, sessionID, other); !errors.Is(err, models.ErrForbidden) {
		t.Fatalf("other account access must be forbidden: %v", err)
	}
	if _, err := client.GetSession(ctx, expiredID); !errors.Is(err, models.ErrSessionNotFound) {
		t.Fatalf("expired session must be rejected: %v", err)
	}
	for range 2 {
		if err := client.DeleteSession(ctx, sessionID); err != nil {
			t.Fatalf("delete session: %v", err)
		}
	}
	if err := client.CheckProfileAccess(ctx, sessionID, owner); !errors.Is(err, models.ErrUnauthorized) {
		t.Fatalf("revoked session must be rejected: %v", err)
	}

	server.Stop()
	if !strings.Contains(logs.String(), `"request_id":"`+requestID+`"`) || strings.Contains(logs.String(), string(sessionID)) {
		t.Fatalf("unexpected Auth logs: %s", &logs)
	}
	for _, metrics := range []*rpcmetrics.Metrics{clientMetrics, serverMetrics} {
		response := httptest.NewRecorder()
		metrics.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		body := response.Body.String()
		if response.Code != http.StatusOK ||
			!strings.Contains(body, `grpc_requests_total{code="PermissionDenied",method="/auth.v1.AuthService/CheckProfileAccess"`) ||
			!strings.Contains(body, "grpc_request_duration_seconds_bucket") || strings.Contains(body, requestID) {
			t.Fatalf("unexpected metrics: %d %s", response.Code, body)
		}
	}
}

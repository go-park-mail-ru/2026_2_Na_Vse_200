package authrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "2026_2_Na_Vse_200/auth-microservice/api/auth/v1"
	"2026_2_Na_Vse_200/auth-microservice/sessions"
	"2026_2_Na_Vse_200/pkg/logger"
	"2026_2_Na_Vse_200/pkg/rpcmetrics"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type sessionStore struct {
	mu       sync.Mutex
	sessions map[sessions.SessionID]sessions.Session
	failedID sessions.SessionID
}

func (s *sessionStore) Create(_ context.Context, session sessions.Session) error {
	if session.CredentialVersion != 1 {
		return sessions.ErrCredentialsChanged
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *sessionStore) GetByID(_ context.Context, id sessions.SessionID) (sessions.Session, error) {
	if id == s.failedID {
		return sessions.Session{}, errors.New("repository error for " + string(id))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return sessions.Session{}, sessions.ErrSessionNotFound
	}
	return session, nil
}

func (s *sessionStore) Delete(_ context.Context, id sessions.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func TestSessionRPC(t *testing.T) {
	const (
		owner     = sessions.UserID("f27b54ad-6cc6-4779-af11-2123653e9cdc")
		other     = sessions.UserID("578d67e2-1955-4711-9ac8-a41c5f09243f")
		sessionID = sessions.SessionID("d860d412-bcd2-4b7e-8d99-a73ff9a0ab06")
		expiredID = sessions.SessionID("0e6d46a1-566b-43a1-b948-8ee6182614d1")
		failedID  = sessions.SessionID("6db61b8c-a202-4eaf-939b-358ff9d3e292")
		requestID = "5c77bbb2-2a25-4f12-ae4c-e9a7861be5c8"
	)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	store := &sessionStore{failedID: failedID, sessions: map[sessions.SessionID]sessions.Session{
		expiredID: {ID: expiredID, UserID: owner, ExpiresAt: now},
	}}
	manager := sessions.NewSessionManager(store, func() sessions.SessionID { return sessionID }, func() time.Time { return now }, time.Hour)
	listener := bufconn.Listen(1024 * 1024)
	clientMetrics := rpcmetrics.New("music-api")
	serverMetrics := rpcmetrics.New("music-auth")
	var logs strings.Builder
	log := zerolog.New(&logs).Level(zerolog.DebugLevel)
	server := grpc.NewServer(grpc.UnaryInterceptor(serverMetrics.Server(log)))
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
	client := authv1.NewAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(logger.WithRequestID(t.Context(), requestID), 5*time.Second)
	defer cancel()

	if _, err := client.CreateSession(ctx, &authv1.CreateSessionRequest{UserId: string(owner)}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing credential version must return InvalidArgument: %v", err)
	}
	if _, err := client.CreateSession(ctx, &authv1.CreateSessionRequest{
		UserId: string(owner), CredentialVersion: 2,
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("credential version mismatch must return Unauthenticated: %v", err)
	}
	session, err := client.CreateSession(ctx, &authv1.CreateSessionRequest{UserId: string(owner), CredentialVersion: 1})
	if err != nil || session.GetId() != string(sessionID) || session.GetUserId() != string(owner) ||
		!session.GetExpiresAt().AsTime().Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected issued session: %+v, err=%v", session, err)
	}
	if actual, err := client.GetSession(ctx, &authv1.SessionRequest{SessionId: string(sessionID)}); err != nil || !proto.Equal(actual, session) {
		t.Fatalf("unexpected stored session: %+v, err=%v", actual, err)
	}
	if _, err := client.CheckProfileAccess(ctx, &authv1.CheckProfileAccessRequest{
		SessionId: string(sessionID), TargetUserId: string(owner),
	}); err != nil {
		t.Fatalf("owner denied: %v", err)
	}
	if _, err := client.CheckProfileAccess(ctx, &authv1.CheckProfileAccessRequest{
		SessionId: string(sessionID), TargetUserId: string(other),
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("other account access must be forbidden: %v", err)
	}
	if _, err := client.GetSession(ctx, &authv1.SessionRequest{SessionId: string(expiredID)}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expired session must be rejected: %v", err)
	}
	if _, err := client.GetSession(ctx, &authv1.SessionRequest{SessionId: string(failedID)}); status.Code(err) != codes.Internal {
		t.Fatalf("repository failure must return Internal: %v", err)
	}
	for range 2 {
		if _, err := client.DeleteSession(ctx, &authv1.SessionRequest{SessionId: string(sessionID)}); err != nil {
			t.Fatalf("delete session: %v", err)
		}
	}
	if _, err := client.CheckProfileAccess(ctx, &authv1.CheckProfileAccessRequest{
		SessionId: string(sessionID), TargetUserId: string(owner),
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("revoked session must be rejected: %v", err)
	}

	server.Stop()
	for _, id := range []sessions.SessionID{sessionID, expiredID, failedID} {
		if strings.Contains(logs.String(), string(id)) {
			t.Fatal("Auth logs contain a session ID")
		}
	}
	checkSessionLogs(t, logs.String(), requestID)
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

func checkSessionLogs(t *testing.T, logs, requestID string) {
	t.Helper()
	type result struct {
		Operation string `json:"operation"`
		Level     string `json:"level"`
		Outcome   string `json:"outcome"`
		Reason    string `json:"reason"`
	}
	var got []result
	for line := range strings.SplitSeq(strings.TrimSpace(logs), "\n") {
		var entry struct {
			result
			Layer     string `json:"layer"`
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode Auth log: %v", err)
		}
		if entry.Layer != "usecase" {
			continue
		}
		if entry.RequestID != requestID {
			t.Errorf("request_id for %s = %q, want %q", entry.Operation, entry.RequestID, requestID)
		}
		got = append(got, entry.result)
	}
	want := []result{
		{Operation: "sessions.create", Level: "debug", Outcome: "denied", Reason: "credentials_changed"},
		{Operation: "sessions.create", Level: "info", Outcome: "ok"},
		{Operation: "sessions.get", Level: "info", Outcome: "ok"},
		{Operation: "sessions.check_profile_access", Level: "info", Outcome: "ok"},
		{Operation: "sessions.check_profile_access", Level: "debug", Outcome: "denied", Reason: "profile_forbidden"},
		{Operation: "sessions.get", Level: "debug", Outcome: "denied", Reason: "session_expired"},
		{Operation: "sessions.get", Level: "error", Outcome: "error", Reason: "repository_error"},
		{Operation: "sessions.delete", Level: "info", Outcome: "ok"},
		{Operation: "sessions.delete", Level: "info", Outcome: "ok"},
		{Operation: "sessions.check_profile_access", Level: "debug", Outcome: "denied", Reason: "session_not_found"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Auth use-case logs = %+v, want %+v", got, want)
	}
}

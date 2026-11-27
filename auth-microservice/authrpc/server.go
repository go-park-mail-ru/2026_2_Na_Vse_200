package authrpc

import (
	"context"
	"errors"
	"uuid"

	authv1 "2026_2_Na_Vse_200/auth-microservice/api/auth/v1"
	"2026_2_Na_Vse_200/auth-microservice/sessions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	sessions *sessions.SessionManager
}

func NewServer(sessions *sessions.SessionManager) *Server {
	return &Server{sessions: sessions}
}

func (s *Server) CreateSession(ctx context.Context, req *authv1.CreateSessionRequest) (*authv1.Session, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}
	version := req.GetCredentialVersion()
	if version <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid credential version")
	}
	session, err := s.sessions.CreateSession(ctx, sessions.UserID(userID.String()), version)
	if err != nil {
		return nil, sessionStatus(err)
	}
	return encodeSession(session), nil
}

func (s *Server) GetSession(ctx context.Context, req *authv1.SessionRequest) (*authv1.Session, error) {
	session, err := s.sessions.GetSession(ctx, sessions.SessionID(req.GetSessionId()))
	if err != nil {
		return nil, sessionStatus(err)
	}
	return encodeSession(session), nil
}

func (s *Server) DeleteSession(ctx context.Context, req *authv1.SessionRequest) (*emptypb.Empty, error) {
	if err := s.sessions.DeleteSession(ctx, sessions.SessionID(req.GetSessionId())); err != nil {
		return nil, sessionStatus(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CheckProfileAccess(ctx context.Context, req *authv1.CheckProfileAccessRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetTargetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target user ID")
	}
	if err := s.sessions.CheckProfileAccess(ctx, sessions.SessionID(req.GetSessionId()), sessions.UserID(id.String())); err != nil {
		return nil, sessionStatus(err)
	}
	return &emptypb.Empty{}, nil
}

func encodeSession(session sessions.Session) *authv1.Session {
	return &authv1.Session{
		Id:        string(session.ID),
		UserId:    string(session.UserID),
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}
}

func sessionStatus(err error) error {
	switch {
	case errors.Is(err, sessions.ErrCredentialsChanged):
		return status.Error(codes.Unauthenticated, "credentials changed")
	case errors.Is(err, sessions.ErrSessionNotFound),
		errors.Is(err, sessions.ErrSessionExpired),
		errors.Is(err, sessions.ErrSessionRevoked):
		return status.Error(codes.Unauthenticated, "invalid session")
	case errors.Is(err, sessions.ErrForbidden):
		return status.Error(codes.PermissionDenied, "profile access denied")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "session operation failed")
	}
}

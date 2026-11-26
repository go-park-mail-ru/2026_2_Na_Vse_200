package authrpc

import (
	"context"
	"fmt"

	authv1 "2026_2_Na_Vse_200/auth-microservice/api/auth/v1"
	"2026_2_Na_Vse_200/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	rpc authv1.AuthServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{rpc: authv1.NewAuthServiceClient(conn)}
}

func (c *Client) CreateSession(ctx context.Context, userID models.UserID) (models.Session, error) {
	session, err := c.rpc.CreateSession(ctx, &authv1.CreateSessionRequest{UserId: string(userID)})
	if err != nil {
		return models.Session{}, err
	}
	return decodeSession(session)
}

func (c *Client) GetSession(ctx context.Context, id models.SessionID) (models.Session, error) {
	session, err := c.rpc.GetSession(ctx, &authv1.SessionRequest{SessionId: string(id)})
	if status.Code(err) == codes.Unauthenticated {
		return models.Session{}, models.ErrSessionNotFound
	}
	if err != nil {
		return models.Session{}, err
	}
	return decodeSession(session)
}

func (c *Client) DeleteSession(ctx context.Context, id models.SessionID) error {
	_, err := c.rpc.DeleteSession(ctx, &authv1.SessionRequest{SessionId: string(id)})
	return err
}

func (c *Client) CheckProfileAccess(ctx context.Context, id models.SessionID, target models.UserID) error {
	_, err := c.rpc.CheckProfileAccess(ctx, &authv1.CheckProfileAccessRequest{
		SessionId:    string(id),
		TargetUserId: string(target),
	})
	switch status.Code(err) {
	case codes.Unauthenticated:
		return models.ErrUnauthorized
	case codes.PermissionDenied:
		return models.ErrForbidden
	default:
		return err
	}
}

func decodeSession(session *authv1.Session) (models.Session, error) {
	if err := session.GetExpiresAt().CheckValid(); err != nil {
		return models.Session{}, fmt.Errorf("invalid session expiration: %w", err)
	}
	return models.Session{
		ID:        models.SessionID(session.GetId()),
		UserID:    models.UserID(session.GetUserId()),
		ExpiresAt: session.GetExpiresAt().AsTime(),
	}, nil
}

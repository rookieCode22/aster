package auth

import (
	"aster/internal/store"
	"context"
)

// Service handles JWT authentication backed by database storage.
type Service struct {
	secret []byte
	store  *store.DB
}

func NewService(jwtSecret string, db *store.DB) *Service {
	return &Service{
		secret: []byte(jwtSecret),
		store:  db,
	}
}

// Store returns the underlying database for use by handlers.
func (s *Service) Store() *store.DB {
	return s.store
}

type contextKey string

const userContextKey contextKey = "auth_user"

func WithUser(ctx context.Context, user *store.UserRow) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func GetUser(r interface{ Context() context.Context }) *store.UserRow {
	if user, ok := r.Context().Value(userContextKey).(*store.UserRow); ok {
		return user
	}
	return nil
}

package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type UserService struct {
	db *DB
}

func (s *UserService) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*UserService).GetByID")
	defer span.End()

	// query user
	return nil, nil
}

func (s *UserService) Create(ctx context.Context, user *User) error {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*UserService).Create")
	defer span.End()

	// create user
	return nil
}

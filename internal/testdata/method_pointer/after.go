package service

import (
	"context"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type UserService struct {
	db *DB
}

func (s *UserService) GetByID(ctx context.Context, id string) (*User, error) {
	defer newrelic.FromContext(ctx).StartSegment("service.(*UserService).GetByID").End() //ctxweaver:generated

	// query user
	return nil, nil
}

func (s *UserService) Create(ctx context.Context, user *User) error {
	defer newrelic.FromContext(ctx).StartSegment("service.(*UserService).Create").End() //ctxweaver:generated

	// create user
	return nil
}

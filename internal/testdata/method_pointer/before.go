package service

import (
	"context"
)

type DB struct{}
type User struct{}

type UserService struct {
	db *DB
}

func (s *UserService) GetByID(ctx context.Context, id string) (*User, error) {

	// query user
	return nil, nil
}

func (s *UserService) Create(ctx context.Context, user *User) error {

	// create user
	return nil
}

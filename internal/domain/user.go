package domain

import "context"

type User struct {
	ID   int
	Name string
}

type UserRepository interface {
	GetUser(ctx context.Context, id int) (*User, error)
}

//go:generate rig-mock -type=UserRepository -outdir=../test/mocks -outfile=user_repository_mock.go

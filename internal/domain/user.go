package domain

import "context"

/* ========================================================================== */
/*                  Example interface, to test the generator                  */
/* ========================================================================== */

type User struct {
	ID   int
	Name string
}

type UserRepository interface {
	GetUser(ctx context.Context, id int) (*User, error)
	CreateUSer(ctx context.Context, name string, age int) error
}

//go:generate rig-mock -type=UserRepository -outdir=../mocks -outfile=user_repository_mock.go

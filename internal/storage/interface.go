package storage

import (
	"context"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
)

// Storage defines the interface for persisting watchclub data
type Storage interface {
	CreateUser(ctx context.Context, user *v1.User) error
	GetUser(ctx context.Context, id string) (*v1.User, error)
	GetUserByEmail(ctx context.Context, email string) (*v1.User, error)
	ListUsers(ctx context.Context) ([]*v1.User, error)
	DeleteUser(ctx context.Context, id string) error

	CreateClub(ctx context.Context, club *v1.Club) error
	GetClub(ctx context.Context, id string) (*v1.Club, error)
	ListClubs(ctx context.Context) ([]*v1.Club, error)
	DeleteClub(ctx context.Context, id string) error

	CreatePick(ctx context.Context, pick *v1.Pick) error
	GetPick(ctx context.Context, id string) (*v1.Pick, error)
	ListPicks(ctx context.Context, clubID string) ([]*v1.Pick, error)
	DeletePick(ctx context.Context, id string) error

	CreateWeeklyAssignment(ctx context.Context, assignment *v1.WeeklyAssignment) error
	GetWeeklyAssignment(ctx context.Context, id string) (*v1.WeeklyAssignment, error)
	ListWeeklyAssignments(ctx context.Context, clubID string) ([]*v1.WeeklyAssignment, error)
	DeleteWeeklyAssignment(ctx context.Context, id string) error
}

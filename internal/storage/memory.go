package storage

import (
	"context"
	"fmt"
	"sync"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
)

// NewMemoryStorage creates a new in-memory storage implementation
func NewMemoryStorage() Storage {
	return &memoryStorage{
		users:             make(map[string]*v1.User),
		clubs:             make(map[string]*v1.Club),
		moviePicks:        make(map[string]*v1.MoviePick),
		weeklyAssignments: make(map[string]*v1.WeeklyAssignment),
	}
}

type memoryStorage struct {
	mu                sync.RWMutex
	users             map[string]*v1.User
	clubs             map[string]*v1.Club
	moviePicks        map[string]*v1.MoviePick
	weeklyAssignments map[string]*v1.WeeklyAssignment
}

// User operations

func (m *memoryStorage) CreateUser(ctx context.Context, user *v1.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.Id]; exists {
		return fmt.Errorf("user already exists: %s", user.Id)
	}
	m.users[user.Id] = user
	return nil
}

func (m *memoryStorage) GetUser(ctx context.Context, id string) (*v1.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

func (m *memoryStorage) GetUserByEmail(ctx context.Context, email string) (*v1.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found with email: %s", email)
}

func (m *memoryStorage) ListUsers(ctx context.Context) ([]*v1.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*v1.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *memoryStorage) DeleteUser(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(m.users, id)
	return nil
}

// Club operations

func (m *memoryStorage) CreateClub(ctx context.Context, club *v1.Club) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clubs[club.Id]; exists {
		return fmt.Errorf("club already exists: %s", club.Id)
	}
	m.clubs[club.Id] = club
	return nil
}

func (m *memoryStorage) GetClub(ctx context.Context, id string) (*v1.Club, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	club, ok := m.clubs[id]
	if !ok {
		return nil, fmt.Errorf("club not found: %s", id)
	}
	return club, nil
}

func (m *memoryStorage) ListClubs(ctx context.Context) ([]*v1.Club, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clubs := make([]*v1.Club, 0, len(m.clubs))
	for _, club := range m.clubs {
		clubs = append(clubs, club)
	}
	return clubs, nil
}

func (m *memoryStorage) DeleteClub(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clubs[id]; !ok {
		return fmt.Errorf("club not found: %s", id)
	}
	delete(m.clubs, id)
	return nil
}

// MoviePick operations

func (m *memoryStorage) CreateMoviePick(ctx context.Context, pick *v1.MoviePick) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.moviePicks[pick.Id]; exists {
		return fmt.Errorf("movie pick already exists: %s", pick.Id)
	}
	m.moviePicks[pick.Id] = pick
	return nil
}

func (m *memoryStorage) GetMoviePick(ctx context.Context, id string) (*v1.MoviePick, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pick, ok := m.moviePicks[id]
	if !ok {
		return nil, fmt.Errorf("movie pick not found: %s", id)
	}
	return pick, nil
}

func (m *memoryStorage) ListMoviePicks(ctx context.Context, clubID string) ([]*v1.MoviePick, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	picks := make([]*v1.MoviePick, 0)
	for _, pick := range m.moviePicks {
		if pick.ClubId == clubID {
			picks = append(picks, pick)
		}
	}
	return picks, nil
}

func (m *memoryStorage) DeleteMoviePick(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.moviePicks[id]; !ok {
		return fmt.Errorf("movie pick not found: %s", id)
	}
	delete(m.moviePicks, id)
	return nil
}

// WeeklyAssignment operations

func (m *memoryStorage) CreateWeeklyAssignment(ctx context.Context, assignment *v1.WeeklyAssignment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.weeklyAssignments[assignment.Id]; exists {
		return fmt.Errorf("weekly assignment already exists: %s", assignment.Id)
	}
	m.weeklyAssignments[assignment.Id] = assignment
	return nil
}

func (m *memoryStorage) GetWeeklyAssignment(ctx context.Context, id string) (*v1.WeeklyAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assignment, ok := m.weeklyAssignments[id]
	if !ok {
		return nil, fmt.Errorf("weekly assignment not found: %s", id)
	}
	return assignment, nil
}

func (m *memoryStorage) ListWeeklyAssignments(ctx context.Context, clubID string) ([]*v1.WeeklyAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assignments := make([]*v1.WeeklyAssignment, 0)
	for _, assignment := range m.weeklyAssignments {
		if assignment.ClubId == clubID {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

func (m *memoryStorage) DeleteWeeklyAssignment(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.weeklyAssignments[id]; !ok {
		return fmt.Errorf("weekly assignment not found: %s", id)
	}
	delete(m.weeklyAssignments, id)
	return nil
}

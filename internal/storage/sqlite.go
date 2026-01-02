package storage

import (
	"context"
	"database/sql"
	"fmt"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
	_ "modernc.org/sqlite"
	"google.golang.org/protobuf/proto"
)

// NewSQLiteStorage creates a new SQLite storage implementation
func NewSQLiteStorage(dbPath string) (Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &sqliteStorage{db: db}, nil
}

type sqliteStorage struct {
	db *sql.DB
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		data BLOB NOT NULL
	);

	CREATE TABLE IF NOT EXISTS clubs (
		id TEXT PRIMARY KEY,
		data BLOB NOT NULL
	);

	CREATE TABLE IF NOT EXISTS picks (
		id TEXT PRIMARY KEY,
		club_id TEXT NOT NULL,
		data BLOB NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_picks_club_id ON picks(club_id);

	CREATE TABLE IF NOT EXISTS scheduled_picks (
		id TEXT PRIMARY KEY,
		club_id TEXT NOT NULL,
		data BLOB NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_scheduled_picks_club_id ON scheduled_picks(club_id);
	`

	_, err := db.Exec(schema)
	return err
}

// User operations

func (s *sqliteStorage) CreateUser(ctx context.Context, user *v1.User) error {
	data, err := proto.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO users (id, data) VALUES (?, ?)", user.Id, data)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (s *sqliteStorage) GetUser(ctx context.Context, id string) (*v1.User, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, "SELECT data FROM users WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user := &v1.User{}
	if err := proto.Unmarshal(data, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return user, nil
}

func (s *sqliteStorage) GetUserByEmail(ctx context.Context, email string) (*v1.User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}

		user := &v1.User{}
		if err := proto.Unmarshal(data, user); err != nil {
			continue
		}

		if user.Email == email {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found with email: %s", email)
}

func (s *sqliteStorage) ListUsers(ctx context.Context) ([]*v1.User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*v1.User
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user := &v1.User{}
		if err := proto.Unmarshal(data, user); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user: %w", err)
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *sqliteStorage) DeleteUser(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// Club operations

func (s *sqliteStorage) CreateClub(ctx context.Context, club *v1.Club) error {
	data, err := proto.Marshal(club)
	if err != nil {
		return fmt.Errorf("failed to marshal club: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO clubs (id, data) VALUES (?, ?)", club.Id, data)
	if err != nil {
		return fmt.Errorf("failed to insert club: %w", err)
	}

	return nil
}

func (s *sqliteStorage) GetClub(ctx context.Context, id string) (*v1.Club, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, "SELECT data FROM clubs WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("club not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query club: %w", err)
	}

	club := &v1.Club{}
	if err := proto.Unmarshal(data, club); err != nil {
		return nil, fmt.Errorf("failed to unmarshal club: %w", err)
	}

	return club, nil
}

func (s *sqliteStorage) ListClubs(ctx context.Context) ([]*v1.Club, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM clubs")
	if err != nil {
		return nil, fmt.Errorf("failed to query clubs: %w", err)
	}
	defer rows.Close()

	var clubs []*v1.Club
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan club: %w", err)
		}

		club := &v1.Club{}
		if err := proto.Unmarshal(data, club); err != nil {
			return nil, fmt.Errorf("failed to unmarshal club: %w", err)
		}

		clubs = append(clubs, club)
	}

	return clubs, nil
}

func (s *sqliteStorage) ListClubsForUser(ctx context.Context, userID string) ([]*v1.Club, error) {
	// Get all clubs and filter by membership
	// (More efficient would be to denormalize member_ids or use a join table)
	allClubs, err := s.ListClubs(ctx)
	if err != nil {
		return nil, err
	}

	var userClubs []*v1.Club
	for _, club := range allClubs {
		for _, memberID := range club.MemberIds {
			if memberID == userID {
				userClubs = append(userClubs, club)
				break
			}
		}
	}

	return userClubs, nil
}

func (s *sqliteStorage) DeleteClub(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM clubs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete club: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("club not found: %s", id)
	}

	return nil
}

// Pick operations

func (s *sqliteStorage) CreatePick(ctx context.Context, pick *v1.Pick) error {
	data, err := proto.Marshal(pick)
	if err != nil {
		return fmt.Errorf("failed to marshal pick: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO picks (id, club_id, data) VALUES (?, ?, ?)", pick.Id, pick.ClubId, data)
	if err != nil {
		return fmt.Errorf("failed to insert pick: %w", err)
	}

	return nil
}

func (s *sqliteStorage) GetPick(ctx context.Context, id string) (*v1.Pick, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, "SELECT data FROM picks WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pick not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query pick: %w", err)
	}

	pick := &v1.Pick{}
	if err := proto.Unmarshal(data, pick); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pick: %w", err)
	}

	return pick, nil
}

func (s *sqliteStorage) ListPicks(ctx context.Context, clubID string) ([]*v1.Pick, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM picks WHERE club_id = ?", clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to query picks: %w", err)
	}
	defer rows.Close()

	var picks []*v1.Pick
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan pick: %w", err)
		}

		pick := &v1.Pick{}
		if err := proto.Unmarshal(data, pick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pick: %w", err)
		}

		picks = append(picks, pick)
	}

	return picks, nil
}

func (s *sqliteStorage) DeletePick(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM picks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete pick: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("pick not found: %s", id)
	}

	return nil
}

// ScheduledPick operations

func (s *sqliteStorage) CreateScheduledPick(ctx context.Context, assignment *v1.ScheduledPick) error {
	data, err := proto.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled pick: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO scheduled_picks (id, club_id, data) VALUES (?, ?, ?)", assignment.Id, assignment.ClubId, data)
	if err != nil {
		return fmt.Errorf("failed to insert scheduled pick: %w", err)
	}

	return nil
}

func (s *sqliteStorage) GetScheduledPick(ctx context.Context, id string) (*v1.ScheduledPick, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, "SELECT data FROM scheduled_picks WHERE id = ?", id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduled pick not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled pick: %w", err)
	}

	assignment := &v1.ScheduledPick{}
	if err := proto.Unmarshal(data, assignment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scheduled pick: %w", err)
	}

	return assignment, nil
}

func (s *sqliteStorage) ListScheduledPicks(ctx context.Context, clubID string) ([]*v1.ScheduledPick, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM scheduled_picks WHERE club_id = ?", clubID)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled picks: %w", err)
	}
	defer rows.Close()

	var assignments []*v1.ScheduledPick
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled pick: %w", err)
		}

		assignment := &v1.ScheduledPick{}
		if err := proto.Unmarshal(data, assignment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scheduled pick: %w", err)
		}

		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

func (s *sqliteStorage) DeleteScheduledPick(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_picks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled pick: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("scheduled pick not found: %s", id)
	}

	return nil
}

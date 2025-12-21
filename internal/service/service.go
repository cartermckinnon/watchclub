package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
	"github.com/cartermckinnon/watchclub/internal/email"
	"github.com/cartermckinnon/watchclub/internal/storage"
)

// WatchClubService implements the WatchClubServiceServer interface
type WatchClubService struct {
	v1.UnimplementedWatchClubServiceServer
	storage     storage.Storage
	emailSender email.Sender
	baseURL     string
}

// New creates a new WatchClubService
func New(store storage.Storage, emailSender email.Sender, baseURL string) *WatchClubService {
	return &WatchClubService{
		storage:     store,
		emailSender: emailSender,
		baseURL:     baseURL,
	}
}

// GetUser gets a user by ID
func (s *WatchClubService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.storage.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &v1.GetUserResponse{User: user}, nil
}

// CreateUser creates a new user
func (s *WatchClubService) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	// Check if email already exists
	existingUser, err := s.storage.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "email already registered")
	}

	user := &v1.User{
		Id:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: timestamppb.Now(),
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &v1.CreateUserResponse{User: user}, nil
}

// CreateClub creates a new movie club
func (s *WatchClubService) CreateClub(ctx context.Context, req *v1.CreateClubRequest) (*v1.CreateClubResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.StartDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date is required")
	}

	club := &v1.Club{
		Id:        uuid.New().String(),
		Name:      req.Name,
		MemberIds: []string{},
		StartDate: req.StartDate,
		Started:   false,
		CreatedAt: timestamppb.Now(),
	}

	if err := s.storage.CreateClub(ctx, club); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create club: %v", err)
	}

	return &v1.CreateClubResponse{Club: club}, nil
}

// JoinClub adds a user to a club
func (s *WatchClubService) JoinClub(ctx context.Context, req *v1.JoinClubRequest) (*v1.JoinClubResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get the club
	club, err := s.storage.GetClub(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	// Verify user exists
	if _, err := s.storage.GetUser(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	// Check if user is already a member
	for _, memberID := range club.MemberIds {
		if memberID == req.UserId {
			return nil, status.Error(codes.AlreadyExists, "user already in club")
		}
	}

	// Add user to club
	club.MemberIds = append(club.MemberIds, req.UserId)

	// Update club in storage
	if err := s.storage.DeleteClub(ctx, club.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update club: %v", err)
	}
	if err := s.storage.CreateClub(ctx, club); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update club: %v", err)
	}

	return &v1.JoinClubResponse{Club: club}, nil
}

// AddMoviePick adds a movie pick to a club
func (s *WatchClubService) AddMoviePick(ctx context.Context, req *v1.AddMoviePickRequest) (*v1.AddMoviePickResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	// Verify club exists
	if _, err := s.storage.GetClub(ctx, req.ClubId); err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	// Verify user exists
	if _, err := s.storage.GetUser(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	pick := &v1.MoviePick{
		Id:        uuid.New().String(),
		ClubId:    req.ClubId,
		UserId:    req.UserId,
		Title:     req.Title,
		Year:      req.Year,
		Notes:     req.Notes,
		CreatedAt: timestamppb.Now(),
	}

	if err := s.storage.CreateMoviePick(ctx, pick); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create movie pick: %v", err)
	}

	return &v1.AddMoviePickResponse{MoviePick: pick}, nil
}

// GetClub gets details about a club including members and their picks
func (s *WatchClubService) GetClub(ctx context.Context, req *v1.GetClubRequest) (*v1.GetClubResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}

	club, err := s.storage.GetClub(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	// Get all members
	members := make([]*v1.User, 0, len(club.MemberIds))
	for _, memberID := range club.MemberIds {
		user, err := s.storage.GetUser(ctx, memberID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get user %s: %v", memberID, err)
		}
		members = append(members, user)
	}

	// Get all movie picks
	picks, err := s.storage.ListMoviePicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get movie picks: %v", err)
	}

	return &v1.GetClubResponse{
		Club:       club,
		Members:    members,
		MoviePicks: picks,
	}, nil
}

// StartClub shuffles all picks and generates the weekly viewing schedule
func (s *WatchClubService) StartClub(ctx context.Context, req *v1.StartClubRequest) (*v1.StartClubResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}

	club, err := s.storage.GetClub(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	if club.Started {
		return nil, status.Error(codes.FailedPrecondition, "club already started")
	}

	// Get all movie picks
	picks, err := s.storage.ListMoviePicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get movie picks: %v", err)
	}

	if len(picks) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no movie picks to shuffle")
	}

	// Shuffle the picks
	shuffled := make([]*v1.MoviePick, len(picks))
	copy(shuffled, picks)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Create weekly assignments
	assignments := make([]*v1.WeeklyAssignment, 0, len(shuffled))
	startDate := club.StartDate.AsTime()

	for i, pick := range shuffled {
		weekStart := startDate.Add(time.Duration(i) * 7 * 24 * time.Hour)
		assignment := &v1.WeeklyAssignment{
			Id:            uuid.New().String(),
			ClubId:        req.ClubId,
			WeekNumber:    int32(i + 1),
			WeekStartDate: timestamppb.New(weekStart),
			Movie:         pick,
		}

		if err := s.storage.CreateWeeklyAssignment(ctx, assignment); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create weekly assignment: %v", err)
		}

		assignments = append(assignments, assignment)
	}

	// Mark club as started
	club.Started = true
	if err := s.storage.DeleteClub(ctx, club.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update club: %v", err)
	}
	if err := s.storage.CreateClub(ctx, club); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update club: %v", err)
	}

	return &v1.StartClubResponse{
		Club:        club,
		Assignments: assignments,
	}, nil
}

// GetWeeklyAssignments gets the weekly movie viewing schedule for a club
func (s *WatchClubService) GetWeeklyAssignments(ctx context.Context, req *v1.GetWeeklyAssignmentsRequest) (*v1.GetWeeklyAssignmentsResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}

	// Verify club exists
	if _, err := s.storage.GetClub(ctx, req.ClubId); err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	assignments, err := s.storage.ListWeeklyAssignments(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get weekly assignments: %v", err)
	}

	return &v1.GetWeeklyAssignmentsResponse{
		Assignments: assignments,
	}, nil
}

// SendRecoveryEmail sends an account recovery email
func (s *WatchClubService) SendRecoveryEmail(ctx context.Context, req *v1.SendRecoveryEmailRequest) (*v1.SendRecoveryEmailResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	// Look up user by email
	user, err := s.storage.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal whether the email exists or not for security
		return &v1.SendRecoveryEmailResponse{
			Success: true,
			Message: "If an account with that email exists, a recovery link has been sent.",
		}, nil
	}

	// Send recovery email
	if err := s.emailSender.SendRecoveryEmail(user.Email, user.Name, user.Id, s.baseURL); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send recovery email: %v", err)
	}

	return &v1.SendRecoveryEmailResponse{
		Success: true,
		Message: "If an account with that email exists, a recovery link has been sent.",
	}, nil
}

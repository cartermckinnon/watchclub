package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
	"github.com/cartermckinnon/watchclub/internal/mail"
	"github.com/cartermckinnon/watchclub/internal/storage"
)

// WatchClubService implements the WatchClubServiceServer interface
type WatchClubService struct {
	v1.UnimplementedWatchClubServiceServer
	storage    storage.Storage
	mailSender mail.Sender
	baseURL    string
	logger     *zap.Logger
}

// New creates a new WatchClubService
func New(store storage.Storage, mailSender mail.Sender, baseURL string, logger *zap.Logger) *WatchClubService {
	return &WatchClubService{
		storage:    store,
		mailSender: mailSender,
		baseURL:    baseURL,
		logger:     logger,
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

	// Validate max_picks_per_member
	maxPicks := req.MaxPicksPerMember
	if maxPicks < 0 {
		return nil, status.Error(codes.InvalidArgument, "max_picks_per_member cannot be negative")
	}
	// Default to 1 if not specified (0 means unlimited for backward compatibility)
	if maxPicks == 0 {
		maxPicks = 1
	}

	// Validate schedule interval
	scheduleQty := req.ScheduleIntervalQuantity
	if scheduleQty <= 0 {
		scheduleQty = 1 // Default to 1
	}

	scheduleUnit := req.ScheduleIntervalUnit
	if scheduleUnit == v1.ScheduleIntervalUnit_SCHEDULE_INTERVAL_UNIT_UNSPECIFIED {
		scheduleUnit = v1.ScheduleIntervalUnit_SCHEDULE_INTERVAL_UNIT_WEEKS // Default to weeks
	}

	club := &v1.Club{
		Id:                       uuid.New().String(),
		Name:                     req.Name,
		MemberIds:                []string{},
		StartDate:                req.StartDate,
		Started:                  false,
		CreatedAt:                timestamppb.Now(),
		MaxPicksPerMember:        maxPicks,
		ScheduleIntervalQuantity: scheduleQty,
		ScheduleIntervalUnit:     scheduleUnit,
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

// AddPick adds a pick to a club
func (s *WatchClubService) AddPick(ctx context.Context, req *v1.AddPickRequest) (*v1.AddPickResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	// Get club and verify it exists
	club, err := s.storage.GetClub(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	// Check if club has already started
	if club.Started {
		return nil, status.Error(codes.FailedPrecondition, "cannot add picks after club has started")
	}

	// Verify user exists
	if _, err := s.storage.GetUser(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	// Check if user has reached max picks (0 means unlimited)
	if club.MaxPicksPerMember > 0 {
		existingPicks, err := s.storage.ListPicks(ctx, req.ClubId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list picks: %v", err)
		}

		userPickCount := 0
		for _, pick := range existingPicks {
			if pick.UserId == req.UserId {
				userPickCount++
			}
		}

		if userPickCount >= int(club.MaxPicksPerMember) {
			return nil, status.Errorf(codes.FailedPrecondition,
				"user has already added maximum number of picks (%d)", club.MaxPicksPerMember)
		}
	}

	pick := &v1.Pick{
		Id:        uuid.New().String(),
		ClubId:    req.ClubId,
		UserId:    req.UserId,
		Title:     req.Title,
		Year:      req.Year,
		Notes:     req.Notes,
		Link:      req.Link,
		CreatedAt: timestamppb.Now(),
	}

	if err := s.storage.CreatePick(ctx, pick); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pick: %v", err)
	}

	return &v1.AddPickResponse{Pick: pick}, nil
}

// DeletePick removes a pick from a club (only allowed before club starts)
func (s *WatchClubService) DeletePick(ctx context.Context, req *v1.DeletePickRequest) (*v1.DeletePickResponse, error) {
	if req.PickId == "" {
		return nil, status.Error(codes.InvalidArgument, "pick_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get the pick to verify ownership and club status
	pick, err := s.storage.GetPick(ctx, req.PickId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "pick not found: %v", err)
	}

	// Verify user owns this pick
	if pick.UserId != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "you can only delete your own picks")
	}

	// Verify club hasn't started
	club, err := s.storage.GetClub(ctx, pick.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	if club.Started {
		return nil, status.Error(codes.FailedPrecondition, "cannot delete picks after club has started")
	}

	// Delete the pick
	if err := s.storage.DeletePick(ctx, req.PickId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete pick: %v", err)
	}

	return &v1.DeletePickResponse{Success: true}, nil
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

	// Get all picks
	picks, err := s.storage.ListPicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get picks: %v", err)
	}

	return &v1.GetClubResponse{
		Club:    club,
		Members: members,
		Picks:   picks,
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

	// Get all picks
	picks, err := s.storage.ListPicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get picks: %v", err)
	}

	if len(picks) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no picks to shuffle")
	}

	// Shuffle the picks
	shuffled := make([]*v1.Pick, len(picks))
	copy(shuffled, picks)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Create scheduled picks
	assignments := make([]*v1.ScheduledPick, 0, len(shuffled))
	startDate := club.StartDate.AsTime()

	// Calculate interval duration based on club's schedule settings
	intervalDuration := calculateIntervalDuration(club.ScheduleIntervalQuantity, club.ScheduleIntervalUnit)

	for i, pick := range shuffled {
		periodStart := startDate.Add(intervalDuration * time.Duration(i))
		assignment := &v1.ScheduledPick{
			Id:             uuid.New().String(),
			ClubId:         req.ClubId,
			SequenceNumber: int32(i + 1),
			StartDate:      timestamppb.New(periodStart),
			Pick:           pick,
		}

		if err := s.storage.CreateScheduledPick(ctx, assignment); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create scheduled pick: %v", err)
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

	// Send notification emails to all members with calendar attachment
	go s.sendClubStartedEmails(ctx, club, assignments)

	return &v1.StartClubResponse{
		Club:        club,
		Assignments: assignments,
	}, nil
}

// sendClubStartedEmails sends notification emails to all club members
func (s *WatchClubService) sendClubStartedEmails(ctx context.Context, club *v1.Club, assignments []*v1.ScheduledPick) {
	s.logger.Info("Sending club started emails",
		zap.String("clubId", club.Id),
		zap.String("clubName", club.Name),
		zap.Int("memberCount", len(club.MemberIds)))

	// Get all users for the club
	userMap := make(map[string]*v1.User)
	for _, memberID := range club.MemberIds {
		user, err := s.storage.GetUser(ctx, memberID)
		if err != nil {
			s.logger.Warn("Failed to get user for email notification",
				zap.String("userId", memberID),
				zap.Error(err))
			continue
		}
		userMap[user.Id] = user
	}

	// Generate ICS calendar data
	icsData := generateICSCalendar(club, assignments, userMap, s.baseURL)
	s.logger.Debug("Generated ICS calendar data",
		zap.String("clubId", club.Id),
		zap.Int("icsSize", len(icsData)))

	// Send email to each member
	emailsSent := 0
	for _, user := range userMap {
		if user.Email == "" {
			s.logger.Warn("User has no email address, skipping",
				zap.String("userId", user.Id),
				zap.String("userName", user.Name))
			continue
		}

		s.logger.Info("Sending club started email",
			zap.String("to", user.Email),
			zap.String("userName", user.Name))

		err := s.mailSender.SendClubStarted(
			user.Email,
			user.Name,
			club.Name,
			club.Id,
			s.baseURL,
			[]byte(icsData),
		)
		if err != nil {
			s.logger.Error("Failed to send club started email",
				zap.String("to", user.Email),
				zap.Error(err))
		} else {
			emailsSent++
		}
	}

	s.logger.Info("Finished sending club started emails",
		zap.String("clubId", club.Id),
		zap.Int("emailsSent", emailsSent),
		zap.Int("totalMembers", len(userMap)))
}

// calculateIntervalDuration converts schedule settings to a time.Duration
func calculateIntervalDuration(quantity int32, unit v1.ScheduleIntervalUnit) time.Duration {
	switch unit {
	case v1.ScheduleIntervalUnit_SCHEDULE_INTERVAL_UNIT_DAYS:
		return time.Duration(quantity) * 24 * time.Hour
	case v1.ScheduleIntervalUnit_SCHEDULE_INTERVAL_UNIT_WEEKS:
		return time.Duration(quantity) * 7 * 24 * time.Hour
	case v1.ScheduleIntervalUnit_SCHEDULE_INTERVAL_UNIT_MONTHS:
		// Approximate: 30 days per month
		return time.Duration(quantity) * 30 * 24 * time.Hour
	default:
		// Default to weeks
		return time.Duration(quantity) * 7 * 24 * time.Hour
	}
}

// GetScheduledPicks gets the schedule for a club
func (s *WatchClubService) GetScheduledPicks(ctx context.Context, req *v1.GetScheduledPicksRequest) (*v1.GetScheduledPicksResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}

	// Verify club exists
	if _, err := s.storage.GetClub(ctx, req.ClubId); err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	assignments, err := s.storage.ListScheduledPicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get scheduled picks: %v", err)
	}

	return &v1.GetScheduledPicksResponse{
		Assignments: assignments,
	}, nil
}

// SendLoginEmail sends an account login email
func (s *WatchClubService) SendLoginEmail(ctx context.Context, req *v1.SendLoginEmailRequest) (*v1.SendLoginEmailResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	// response is always the same
	response := v1.SendLoginEmailResponse{
		Success: true,
		Message: "If an account with that email exists, a login link has been sent.",
	}

	user, err := s.storage.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return &response, nil
	}

	if err := s.mailSender.SendLogin(user.Email, user.Name, user.Id, s.baseURL); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send login email: %v", err)
	}

	return &response, nil
}

// GetClubCalendar generates an ICS calendar file for a club's schedule
func (s *WatchClubService) GetClubCalendar(ctx context.Context, req *v1.GetClubCalendarRequest) (*v1.GetClubCalendarResponse, error) {
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "club_id is required")
	}

	club, err := s.storage.GetClub(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "club not found: %v", err)
	}

	if !club.Started {
		return nil, status.Error(codes.FailedPrecondition, "club must be started to generate calendar")
	}

	assignments, err := s.storage.ListScheduledPicks(ctx, req.ClubId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list scheduled picks: %v", err)
	}

	// Get members to include picker names
	users, err := s.storage.ListUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	userMap := make(map[string]*v1.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	icsData := generateICSCalendar(club, assignments, userMap, s.baseURL)

	return &v1.GetClubCalendarResponse{
		IcsData: icsData,
	}, nil
}

// generateICSCalendar creates an ICS calendar file from scheduled picks
func generateICSCalendar(club *v1.Club, assignments []*v1.ScheduledPick, userMap map[string]*v1.User, baseURL string) string {
	var ics string

	// Calendar header
	ics += "BEGIN:VCALENDAR\r\n"
	ics += "VERSION:2.0\r\n"
	ics += "PRODID:-//WatchClub//Schedule//EN\r\n"
	ics += "CALSCALE:GREGORIAN\r\n"
	ics += "METHOD:PUBLISH\r\n"
	ics += "X-WR-CALNAME:" + escapeICSText(club.Name) + " - Schedule\r\n"
	ics += "X-WR-TIMEZONE:UTC\r\n"

	// Calculate interval duration
	intervalDuration := calculateIntervalDuration(club.ScheduleIntervalQuantity, club.ScheduleIntervalUnit)

	// Add events for each assignment
	for _, assignment := range assignments {
		pick := assignment.Pick
		startDate := assignment.StartDate.AsTime()
		endDate := startDate.Add(intervalDuration)

		// Get picker name
		pickerName := "Unknown"
		if user, ok := userMap[pick.UserId]; ok {
			pickerName = user.Name
		}

		// Build description
		description := "Picked by " + pickerName
		if pick.Notes != "" {
			description += "\\n\\nNotes: " + escapeICSText(pick.Notes)
		}
		description += "\\n\\nView details: " + baseURL + "/#/club/" + club.Id + "/pick/" + pick.Id

		// Event
		ics += "BEGIN:VEVENT\r\n"
		ics += "UID:" + pick.Id + "@watchclub\r\n"
		ics += "DTSTAMP:" + formatICSDateTime(time.Now()) + "\r\n"
		ics += "DTSTART;VALUE=DATE:" + formatICSDate(startDate) + "\r\n"
		ics += "DTEND;VALUE=DATE:" + formatICSDate(endDate) + "\r\n"
		ics += "SUMMARY:" + escapeICSText(pick.Title)
		if pick.Year > 0 {
			ics += " (" + string(rune(pick.Year/1000+'0')) + string(rune((pick.Year/100)%10+'0')) + string(rune((pick.Year/10)%10+'0')) + string(rune(pick.Year%10+'0')) + ")"
		}
		ics += "\r\n"
		ics += "DESCRIPTION:" + description + "\r\n"
		if pick.Link != "" {
			ics += "LOCATION:" + escapeICSText(pick.Link) + "\r\n"
			ics += "URL:" + escapeICSText(pick.Link) + "\r\n"
		}
		ics += "TRANSP:TRANSPARENT\r\n"
		ics += "END:VEVENT\r\n"
	}

	ics += "END:VCALENDAR\r\n"

	return ics
}

// formatICSDate formats a time as YYYYMMDD for ICS DATE format
func formatICSDate(t time.Time) string {
	return t.Format("20060102")
}

// formatICSDateTime formats a time as YYYYMMDDTHHMMSSZ for ICS DATETIME format
func formatICSDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escapeICSText escapes special characters in ICS text fields
func escapeICSText(s string) string {
	// Replace special characters
	s = replaceAll(s, "\\", "\\\\")
	s = replaceAll(s, ",", "\\,")
	s = replaceAll(s, ";", "\\;")
	s = replaceAll(s, "\n", "\\n")
	s = replaceAll(s, "\r", "")
	return s
}

// replaceAll is a simple string replace helper
func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i == -1 {
			result += s
			break
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
	return result
}

// indexOf finds the first occurrence of a substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

package mail

// Sender is an interface for sending mail
type Sender interface {
	SendLogin(to, userName, userID, baseURL string) error
	SendClubStarted(to, userName, clubName, clubID, baseURL string, icsData []byte) error
}

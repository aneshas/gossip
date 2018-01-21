package chat

// User represents user entity
type User struct {
	Nick     string `json:"nick"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

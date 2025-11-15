// Package samplepkg contains example structures for our documentation generator.
package samplepkg

import "fmt"

// User represents a user in the system.
// This is a detailed description of the User object, explaining its purpose
// and how it is used throughout the application.
type User struct {
	// ID is the unique identifier for the user.
	// It is a primary key in the database.
	ID int `json:"id"`
	// Username is the user's chosen name. It must be unique.
	Username string `json:"username"`
	// IsActive indicates if the user's account is enabled.
	IsActive bool `json:"is_active" db:"is_active"`
}

// IsValid checks if the user model has a valid state.
// It returns false if the username is empty or the ID is not set.
func (u *User) IsValid() bool {
	return u.ID > 0 && u.Username != ""
}

// Admin represents an administrator with elevated privileges.
type Admin struct {
	// User holds the basic user information.
	// This embeds the User struct.
	User
	// AccessLevel defines the level of administrative access.
	// Level 1 is basic admin, Level 5 is super-admin.
	AccessLevel int `json:"access_level"`
}

// GrantPermissions gives another user a specific permission level.
// This is a complex operation that logs the action and returns
// an error if the target user does not exist or the level is invalid.
func (a *Admin) GrantPermissions(target *User, level int) error {
	if a.AccessLevel < 5 {
		return fmt.Errorf("insufficient access level to grant permissions")
	}
	if target == nil {
		return fmt.Errorf("target user cannot be nil")
	}
	// In a real application, you would save this to a database.
	fmt.Printf("Permissions granted to user %s with level %d\n", target.Username, level)
	return nil
}

// Package samplepkg is a demonstration package.
// It contains types and functions related to users and utilities
// to show how the doc generator works.
package samplepkg

import "fmt"

// User represents a user in the system.
// It contains a public ID and a private name.
type User struct {
   ID   int
   name string // unexported field
}

// NewUser creates and returns a new User.
func NewUser(id int, name string) *User {
   return &User{
      ID:   id,
      name: name,
   }
}

// FullName returns the full name of the user.
// This is a method on the User type.
func (u *User) FullName() string {
   return fmt.Sprintf("User-%d-%s", u.ID, u.name)
}

// HasPermission checks if the user has a given permission.
// It's a dummy function for demonstration.
func (u *User) HasPermission(perm string) bool {
   return perm == "admin"
}

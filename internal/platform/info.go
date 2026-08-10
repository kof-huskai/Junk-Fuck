// Package platform reports Windows-specific system information in a
// portable, testable way. It performs no privileged operations.
package platform

// Info describes the host the application is running on.
type Info struct {
	OS      string `json:"os"`
	Version string `json:"version"`
	IsAdmin bool   `json:"isAdmin"`
	Arch    string `json:"arch"`
}

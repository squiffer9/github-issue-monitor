package models

import (
	"fmt"
	"time"
)

type IssueEvent struct {
	Action string `json:"action"`
	Issue  struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		URL       string    `json:"html_url"`
	} `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
		URL      string `json:"html_url"`
	} `json:"repository"`
}

// FormatMessage is a method that formats the event into a human-readable format
// Example: [15:04:05] New issue #1: Example issue in owner/repo
func (e *IssueEvent) FormatMessage() string {
	return fmt.Sprintf("[%s] New issue #%d: %s in %s\nURL: %s",
		time.Now().Format("15:04:05"),
		e.Issue.Number,
		e.Issue.Title,
		e.Repository.FullName,
		e.Issue.URL,
	)
}

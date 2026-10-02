package main

// Repository represents a starred GitHub repository.
type Repository struct {
	NameWithOwner  string
	Description    string
	Language       string
	URL            string
	StargazerCount int
	IsPrivate      bool
	Topics         []string
}

// Config holds all CLI configuration options.
type Config struct {
	Username    string
	Token       string
	Sort        bool
	Topic       bool
	TopicLimit  int
	Repository  string
	Filename    string
	Message     string
	Private     bool
	OutputFile  string
	ShowVersion bool
}

package core

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusBlocked    Status = "blocked"
	StatusArchived   Status = "archived"
)

func (s Status) String() string { return string(s) }

func AllStatuses() []Status {
	return []Status{StatusNew, StatusInProgress, StatusDone, StatusBlocked, StatusArchived}
}

type Project struct {
	ID               int64
	Name             string
	Description      string
	Status           Status
	BlockedReason    string
	FeatureCount     int
	RequirementCount int
	TaskCount        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Feature struct {
	ID               int64
	ProjectID        int64
	Name             string
	Description      string
	Status           Status
	BlockedReason    string
	RequirementCount int
	TaskCount        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Requirement struct {
	ID            int64
	FeatureID     int64
	Name          string
	Description   string
	Status        Status
	BlockedReason string
	TaskCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Task struct {
	ID            int64
	RequirementID int64
	Title         string
	Description   string
	Status        Status
	BlockedReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

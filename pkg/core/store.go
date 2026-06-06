package core

import "context"

type Store interface {
	// Projects
	CreateProject(ctx context.Context, p Project) (Project, error)
	GetProject(ctx context.Context, id int64) (Project, error)
	ListProjects(ctx context.Context) ([]Project, error)
	UpdateProject(ctx context.Context, p Project) (Project, error)
	DeleteProject(ctx context.Context, id int64) error

	// Features
	CreateFeature(ctx context.Context, f Feature) (Feature, error)
	GetFeature(ctx context.Context, id int64) (Feature, error)
	ListFeatures(ctx context.Context, projectID int64) ([]Feature, error)
	UpdateFeature(ctx context.Context, f Feature) (Feature, error)
	DeleteFeature(ctx context.Context, id int64) error

	// Requirements
	CreateRequirement(ctx context.Context, r Requirement) (Requirement, error)
	GetRequirement(ctx context.Context, id int64) (Requirement, error)
	ListRequirements(ctx context.Context, featureID int64) ([]Requirement, error)
	UpdateRequirement(ctx context.Context, r Requirement) (Requirement, error)
	DeleteRequirement(ctx context.Context, id int64) error

	// Tasks
	CreateTask(ctx context.Context, t Task) (Task, error)
	GetTask(ctx context.Context, id int64) (Task, error)
	ListTasks(ctx context.Context, requirementID int64) ([]Task, error)
	UpdateTask(ctx context.Context, t Task) (Task, error)
	DeleteTask(ctx context.Context, id int64) error
}

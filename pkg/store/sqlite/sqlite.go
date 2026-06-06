package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rskulles/taskit/pkg/core"
	_ "modernc.org/sqlite"
)

// To change the schema: update createSchema below and delete the database file.

type Store struct {
	db        *sql.DB
	statusIDs map[core.Status]int64 // name → id cache, populated after migrations
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // sqlite does not support concurrent writes
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return err
	}
	if err := s.createSchema(); err != nil {
		return err
	}
	return s.loadStatusIDs()
}

func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS statuses (
	id   INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO statuses (name) VALUES
	('new'), ('in_progress'), ('done'), ('blocked'), ('archived');

CREATE TABLE IF NOT EXISTS projects (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	name           TEXT NOT NULL,
	description    TEXT NOT NULL DEFAULT '',
	status_id      INTEGER NOT NULL REFERENCES statuses(id),
	blocked_reason TEXT NOT NULL DEFAULT '',
	created_at     DATETIME NOT NULL,
	updated_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS features (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id     INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	name           TEXT NOT NULL,
	description    TEXT NOT NULL DEFAULT '',
	status_id      INTEGER NOT NULL REFERENCES statuses(id),
	blocked_reason TEXT NOT NULL DEFAULT '',
	created_at     DATETIME NOT NULL,
	updated_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS requirements (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	feature_id     INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
	name           TEXT NOT NULL,
	description    TEXT NOT NULL DEFAULT '',
	status_id      INTEGER NOT NULL REFERENCES statuses(id),
	blocked_reason TEXT NOT NULL DEFAULT '',
	created_at     DATETIME NOT NULL,
	updated_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	requirement_id INTEGER NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
	title          TEXT NOT NULL,
	description    TEXT NOT NULL DEFAULT '',
	status_id      INTEGER NOT NULL REFERENCES statuses(id),
	blocked_reason TEXT NOT NULL DEFAULT '',
	created_at     DATETIME NOT NULL,
	updated_at     DATETIME NOT NULL
);
`)
	return err
}

// loadStatusIDs builds the in-memory name→id cache from the statuses table.
// Called once on startup after the schema is created.
func (s *Store) loadStatusIDs() error {
	rows, err := s.db.Query(`SELECT id, name FROM statuses`)
	if err != nil {
		return err
	}
	defer rows.Close()
	s.statusIDs = make(map[core.Status]int64)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		s.statusIDs[core.Status(name)] = id
	}
	return rows.Err()
}

func (s *Store) statusID(st core.Status) (int64, error) {
	id, ok := s.statusIDs[st]
	if !ok {
		return 0, fmt.Errorf("unknown status %q", st)
	}
	return id, nil
}

// ── Projects ──────────────────────────────────────────────────────────────────

func (s *Store) CreateProject(ctx context.Context, p core.Project) (core.Project, error) {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = core.StatusNew
	}
	sid, err := s.statusID(p.Status)
	if err != nil {
		return p, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, description, status_id, blocked_reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.Description, sid, p.BlockedReason, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.ID, _ = res.LastInsertId()
	return p, nil
}

func (s *Store) GetProject(ctx context.Context, id int64) (core.Project, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.name, p.description, st.name, p.blocked_reason, p.created_at, p.updated_at
		FROM projects p JOIN statuses st ON p.status_id = st.id
		WHERE p.id = ?`, id)
	return scanProject(row)
}

func (s *Store) ListProjects(ctx context.Context) ([]core.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.description, st.name, p.blocked_reason, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM features WHERE project_id = p.id),
		       (SELECT COUNT(*) FROM requirements r JOIN features f ON r.feature_id = f.id WHERE f.project_id = p.id),
		       (SELECT COUNT(*) FROM tasks t JOIN requirements r ON t.requirement_id = r.id JOIN features f ON r.feature_id = f.id WHERE f.project_id = p.id)
		FROM projects p JOIN statuses st ON p.status_id = st.id
		ORDER BY p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Project
	for rows.Next() {
		p, err := scanProjectWithCounts(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdateProject(ctx context.Context, p core.Project) (core.Project, error) {
	sid, err := s.statusID(p.Status)
	if err != nil {
		return p, err
	}
	p.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE projects SET name=?, description=?, status_id=?, blocked_reason=?, updated_at=? WHERE id=?`,
		p.Name, p.Description, sid, p.BlockedReason, p.UpdatedAt, p.ID,
	)
	return p, err
}

func (s *Store) DeleteProject(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
	return err
}

// ── Features ──────────────────────────────────────────────────────────────────

func (s *Store) CreateFeature(ctx context.Context, f core.Feature) (core.Feature, error) {
	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now
	if f.Status == "" {
		f.Status = core.StatusNew
	}
	sid, err := s.statusID(f.Status)
	if err != nil {
		return f, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO features (project_id, name, description, status_id, blocked_reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.ProjectID, f.Name, f.Description, sid, f.BlockedReason, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return f, err
	}
	f.ID, _ = res.LastInsertId()
	return f, nil
}

func (s *Store) GetFeature(ctx context.Context, id int64) (core.Feature, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT f.id, f.project_id, f.name, f.description, st.name, f.blocked_reason, f.created_at, f.updated_at
		FROM features f JOIN statuses st ON f.status_id = st.id
		WHERE f.id = ?`, id)
	return scanFeature(row)
}

func (s *Store) ListFeatures(ctx context.Context, projectID int64) ([]core.Feature, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.project_id, f.name, f.description, st.name, f.blocked_reason, f.created_at, f.updated_at,
		       (SELECT COUNT(*) FROM requirements WHERE feature_id = f.id),
		       (SELECT COUNT(*) FROM tasks t JOIN requirements r ON t.requirement_id = r.id WHERE r.feature_id = f.id)
		FROM features f JOIN statuses st ON f.status_id = st.id
		WHERE f.project_id = ?
		ORDER BY f.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Feature
	for rows.Next() {
		f, err := scanFeatureWithCounts(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) UpdateFeature(ctx context.Context, f core.Feature) (core.Feature, error) {
	sid, err := s.statusID(f.Status)
	if err != nil {
		return f, err
	}
	f.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE features SET name=?, description=?, status_id=?, blocked_reason=?, updated_at=? WHERE id=?`,
		f.Name, f.Description, sid, f.BlockedReason, f.UpdatedAt, f.ID,
	)
	return f, err
}

func (s *Store) DeleteFeature(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM features WHERE id=?`, id)
	return err
}

// ── Requirements ──────────────────────────────────────────────────────────────

func (s *Store) CreateRequirement(ctx context.Context, r core.Requirement) (core.Requirement, error) {
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = core.StatusNew
	}
	sid, err := s.statusID(r.Status)
	if err != nil {
		return r, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO requirements (feature_id, name, description, status_id, blocked_reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.FeatureID, r.Name, r.Description, sid, r.BlockedReason, r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		return r, err
	}
	r.ID, _ = res.LastInsertId()
	return r, nil
}

func (s *Store) GetRequirement(ctx context.Context, id int64) (core.Requirement, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.feature_id, r.name, r.description, st.name, r.blocked_reason, r.created_at, r.updated_at
		FROM requirements r JOIN statuses st ON r.status_id = st.id
		WHERE r.id = ?`, id)
	return scanRequirement(row)
}

func (s *Store) ListRequirements(ctx context.Context, featureID int64) ([]core.Requirement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.feature_id, r.name, r.description, st.name, r.blocked_reason, r.created_at, r.updated_at,
		       (SELECT COUNT(*) FROM tasks WHERE requirement_id = r.id)
		FROM requirements r JOIN statuses st ON r.status_id = st.id
		WHERE r.feature_id = ?
		ORDER BY r.name`, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Requirement
	for rows.Next() {
		r, err := scanRequirementWithCounts(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) UpdateRequirement(ctx context.Context, r core.Requirement) (core.Requirement, error) {
	sid, err := s.statusID(r.Status)
	if err != nil {
		return r, err
	}
	r.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE requirements SET name=?, description=?, status_id=?, blocked_reason=?, updated_at=? WHERE id=?`,
		r.Name, r.Description, sid, r.BlockedReason, r.UpdatedAt, r.ID,
	)
	return r, err
}

func (s *Store) DeleteRequirement(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM requirements WHERE id=?`, id)
	return err
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func (s *Store) CreateTask(ctx context.Context, t core.Task) (core.Task, error) {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = core.StatusNew
	}
	sid, err := s.statusID(t.Status)
	if err != nil {
		return t, err
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (requirement_id, title, description, status_id, blocked_reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.RequirementID, t.Title, t.Description, sid, t.BlockedReason, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}
	t.ID, _ = res.LastInsertId()
	return t, nil
}

func (s *Store) GetTask(ctx context.Context, id int64) (core.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.requirement_id, t.title, t.description, st.name, t.blocked_reason, t.created_at, t.updated_at
		FROM tasks t JOIN statuses st ON t.status_id = st.id
		WHERE t.id = ?`, id)
	return scanTask(row)
}

func (s *Store) ListTasks(ctx context.Context, requirementID int64) ([]core.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.requirement_id, t.title, t.description, st.name, t.blocked_reason, t.created_at, t.updated_at
		FROM tasks t JOIN statuses st ON t.status_id = st.id
		WHERE t.requirement_id = ?
		ORDER BY t.title`, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateTask(ctx context.Context, t core.Task) (core.Task, error) {
	sid, err := s.statusID(t.Status)
	if err != nil {
		return t, err
	}
	t.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE tasks SET title=?, description=?, status_id=?, blocked_reason=?, updated_at=? WHERE id=?`,
		t.Title, t.Description, sid, t.BlockedReason, t.UpdatedAt, t.ID,
	)
	return t, err
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, id)
	return err
}

// ── scanners ──────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(s scanner) (core.Project, error) {
	var p core.Project
	err := s.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.BlockedReason, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func scanProjectWithCounts(s scanner) (core.Project, error) {
	var p core.Project
	err := s.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.BlockedReason, &p.CreatedAt, &p.UpdatedAt,
		&p.FeatureCount, &p.RequirementCount, &p.TaskCount)
	return p, err
}

func scanFeature(s scanner) (core.Feature, error) {
	var f core.Feature
	err := s.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Description, &f.Status, &f.BlockedReason, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func scanFeatureWithCounts(s scanner) (core.Feature, error) {
	var f core.Feature
	err := s.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Description, &f.Status, &f.BlockedReason, &f.CreatedAt, &f.UpdatedAt,
		&f.RequirementCount, &f.TaskCount)
	return f, err
}

func scanRequirement(s scanner) (core.Requirement, error) {
	var r core.Requirement
	err := s.Scan(&r.ID, &r.FeatureID, &r.Name, &r.Description, &r.Status, &r.BlockedReason, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func scanRequirementWithCounts(s scanner) (core.Requirement, error) {
	var r core.Requirement
	err := s.Scan(&r.ID, &r.FeatureID, &r.Name, &r.Description, &r.Status, &r.BlockedReason, &r.CreatedAt, &r.UpdatedAt,
		&r.TaskCount)
	return r, err
}

func scanTask(s scanner) (core.Task, error) {
	var t core.Task
	err := s.Scan(&t.ID, &t.RequirementID, &t.Title, &t.Description, &t.Status, &t.BlockedReason, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

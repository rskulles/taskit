package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rskulles/taskit/pkg/core"
)

// Client implements core.Store over HTTP.
type Client struct {
	base string
	http *http.Client
}

func New(baseURL string) *Client {
	return &Client{base: baseURL, http: &http.Client{}}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, &buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, e["error"])
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func get[T any](c *Client, ctx context.Context, path string) (T, error) {
	var out T
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

func post[T any](c *Client, ctx context.Context, path string, body any) (T, error) {
	var out T
	return out, c.do(ctx, http.MethodPost, path, body, &out)
}

func put[T any](c *Client, ctx context.Context, path string, body any) (T, error) {
	var out T
	return out, c.do(ctx, http.MethodPut, path, body, &out)
}

// ── Projects ──────────────────────────────────────────────────────────────────

func (c *Client) ListProjects(ctx context.Context) ([]core.Project, error) {
	return get[[]core.Project](c, ctx, "/projects")
}

func (c *Client) CreateProject(ctx context.Context, p core.Project) (core.Project, error) {
	return post[core.Project](c, ctx, "/projects", p)
}

func (c *Client) GetProject(ctx context.Context, id int64) (core.Project, error) {
	return get[core.Project](c, ctx, fmt.Sprintf("/projects/%d", id))
}

func (c *Client) UpdateProject(ctx context.Context, p core.Project) (core.Project, error) {
	return put[core.Project](c, ctx, fmt.Sprintf("/projects/%d", p.ID), p)
}

func (c *Client) DeleteProject(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/projects/%d", id), nil, nil)
}

// ── Features ──────────────────────────────────────────────────────────────────

func (c *Client) ListFeatures(ctx context.Context, projectID int64) ([]core.Feature, error) {
	return get[[]core.Feature](c, ctx, fmt.Sprintf("/projects/%d/features", projectID))
}

func (c *Client) CreateFeature(ctx context.Context, f core.Feature) (core.Feature, error) {
	return post[core.Feature](c, ctx, fmt.Sprintf("/projects/%d/features", f.ProjectID), f)
}

func (c *Client) GetFeature(ctx context.Context, id int64) (core.Feature, error) {
	return get[core.Feature](c, ctx, fmt.Sprintf("/features/%d", id))
}

func (c *Client) UpdateFeature(ctx context.Context, f core.Feature) (core.Feature, error) {
	return put[core.Feature](c, ctx, fmt.Sprintf("/features/%d", f.ID), f)
}

func (c *Client) DeleteFeature(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/features/%d", id), nil, nil)
}

// ── Requirements ──────────────────────────────────────────────────────────────

func (c *Client) ListRequirements(ctx context.Context, featureID int64) ([]core.Requirement, error) {
	return get[[]core.Requirement](c, ctx, fmt.Sprintf("/features/%d/requirements", featureID))
}

func (c *Client) CreateRequirement(ctx context.Context, r core.Requirement) (core.Requirement, error) {
	return post[core.Requirement](c, ctx, fmt.Sprintf("/features/%d/requirements", r.FeatureID), r)
}

func (c *Client) GetRequirement(ctx context.Context, id int64) (core.Requirement, error) {
	return get[core.Requirement](c, ctx, fmt.Sprintf("/requirements/%d", id))
}

func (c *Client) UpdateRequirement(ctx context.Context, r core.Requirement) (core.Requirement, error) {
	return put[core.Requirement](c, ctx, fmt.Sprintf("/requirements/%d", r.ID), r)
}

func (c *Client) DeleteRequirement(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/requirements/%d", id), nil, nil)
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func (c *Client) ListTasks(ctx context.Context, requirementID int64) ([]core.Task, error) {
	return get[[]core.Task](c, ctx, fmt.Sprintf("/requirements/%d/tasks", requirementID))
}

func (c *Client) CreateTask(ctx context.Context, t core.Task) (core.Task, error) {
	return post[core.Task](c, ctx, fmt.Sprintf("/requirements/%d/tasks", t.RequirementID), t)
}

func (c *Client) GetTask(ctx context.Context, id int64) (core.Task, error) {
	return get[core.Task](c, ctx, fmt.Sprintf("/tasks/%d", id))
}

func (c *Client) UpdateTask(ctx context.Context, t core.Task) (core.Task, error) {
	return put[core.Task](c, ctx, fmt.Sprintf("/tasks/%d", t.ID), t)
}

func (c *Client) DeleteTask(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/tasks/%d", id), nil, nil)
}

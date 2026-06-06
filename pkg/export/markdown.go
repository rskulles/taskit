package export

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/rskulles/taskit/pkg/core"
)

const dateFmt = "Jan 2, 2006"

// Markdown writes a full project tree to w in Markdown format.
func Markdown(ctx context.Context, store core.Store, projectID int64, w io.Writer) error {
	project, err := store.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	writeHeading(w, 1, project.Name, project.Status)
	writeMeta(w, project.Status, project.BlockedReason, project.CreatedAt.Local().Format(dateFmt), project.UpdatedAt.Local().Format(dateFmt), project.CreatedAt.Equal(project.UpdatedAt))
	writeBody(w, project.Description)

	features, err := store.ListFeatures(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list features: %w", err)
	}

	for _, f := range features {
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w)
		writeHeading(w, 2, f.Name, f.Status)
		writeMeta(w, f.Status, f.BlockedReason, "", "", true)
		writeBody(w, f.Description)

		requirements, err := store.ListRequirements(ctx, f.ID)
		if err != nil {
			return fmt.Errorf("list requirements for feature %d: %w", f.ID, err)
		}

		for _, r := range requirements {
			writeHeading(w, 3, r.Name, r.Status)
			writeMeta(w, r.Status, r.BlockedReason, "", "", true)
			writeBody(w, r.Description)

			tasks, err := store.ListTasks(ctx, r.ID)
			if err != nil {
				return fmt.Errorf("list tasks for requirement %d: %w", r.ID, err)
			}

			for _, t := range tasks {
				writeTodo(w, t)
			}
			if len(tasks) > 0 {
				fmt.Fprintln(w)
			}
		}
	}

	return nil
}

// Filename returns a safe filename derived from the project name.
func Filename(projectName string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(projectName), "-"), "-")
	if slug == "" {
		slug = "project"
	}
	return slug + ".md"
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isDone(s core.Status) bool { return s == core.StatusDone }

func strike(name string, s core.Status) string {
	if isDone(s) {
		return "~~" + name + "~~"
	}
	return name
}

func writeHeading(w io.Writer, level int, name string, status core.Status) {
	fmt.Fprintf(w, "%s %s\n\n", strings.Repeat("#", level), strike(name, status))
}

func writeMeta(w io.Writer, status core.Status, blockedReason, created, edited string, sameDate bool) {
	fmt.Fprintf(w, "**Status:** %s  \n", status)
	if created != "" {
		fmt.Fprintf(w, "**Created:** %s  \n", created)
	}
	if !sameDate && edited != "" {
		fmt.Fprintf(w, "**Last edited:** %s  \n", edited)
	}
	if blockedReason != "" {
		fmt.Fprintf(w, "**Blocked reason:** %s  \n", blockedReason)
	}
	fmt.Fprintln(w)
}

func writeBody(w io.Writer, description string) {
	if description != "" {
		fmt.Fprintf(w, "%s\n\n", description)
	}
}

func writeTodo(w io.Writer, t core.Task) {
	check := "[ ]"
	title := t.Title
	if isDone(t.Status) {
		check = "[x]"
		title = "~~" + title + "~~"
	}
	fmt.Fprintf(w, "- %s %s\n", check, title)
	if t.BlockedReason != "" {
		fmt.Fprintf(w, "  > Blocked: %s\n", t.BlockedReason)
	}
}

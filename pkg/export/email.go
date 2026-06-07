package export

import (
	"context"
	"fmt"
	gohtml "html"
	"io"
	"strings"

	"github.com/rskulles/taskit/pkg/core"
)

const emailDateFmt = "January 2, 2006"

// HTMLEmail writes the project tree as an Outlook-compatible HTML email body to w.
// All styles are inlined — Outlook strips external CSS.
func HTMLEmail(ctx context.Context, store core.Store, projectID int64, w io.Writer) error {
	project, err := store.GetProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	fmt.Fprint(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	fmt.Fprint(w, `<body style="font-family:Arial,sans-serif;color:#333;max-width:800px;margin:0;padding:20px;">`)

	writeEmailHeading(w, 1, project.Name, project.Status,
		"color:#2c3e50;border-bottom:2px solid #eee;padding-bottom:10px;margin-top:0")
	writeEmailMeta(w, project.Status, project.BlockedReason,
		project.CreatedAt.Local().Format(emailDateFmt),
		project.UpdatedAt.Local().Format(emailDateFmt),
		project.CreatedAt.Equal(project.UpdatedAt))
	writeEmailBody(w, project.Description)

	features, err := store.ListFeatures(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list features: %w", err)
	}

	for _, f := range features {
		fmt.Fprint(w, `<hr style="border:none;border-top:1px solid #eee;margin:24px 0">`)
		writeEmailHeading(w, 2, f.Name, f.Status, "color:#34495e")
		writeEmailMeta(w, f.Status, f.BlockedReason, "", "", true)
		writeEmailBody(w, f.Description)

		requirements, err := store.ListRequirements(ctx, f.ID)
		if err != nil {
			return fmt.Errorf("list requirements for feature %d: %w", f.ID, err)
		}

		for _, r := range requirements {
			writeEmailHeading(w, 3, r.Name, r.Status, "color:#7f8c8d")
			writeEmailMeta(w, r.Status, r.BlockedReason, "", "", true)
			writeEmailBody(w, r.Description)

			tasks, err := store.ListTasks(ctx, r.ID)
			if err != nil {
				return fmt.Errorf("list tasks for requirement %d: %w", r.ID, err)
			}
			if len(tasks) > 0 {
				fmt.Fprint(w, `<ul style="list-style:none;padding-left:20px;margin:8px 0">`)
				for _, t := range tasks {
					writeEmailTask(w, t)
				}
				fmt.Fprint(w, `</ul>`)
			}
		}
	}

	fmt.Fprint(w, `</body></html>`)
	return nil
}

// WriteEML writes a minimal RFC 2822 .eml file with an HTML body to w.
func WriteEML(subject, from, to, htmlBody string, w io.Writer) {
	fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		from, to, subject)
	fmt.Fprint(w, htmlBody)
}

// EMLFilename returns a safe .eml filename derived from the project name.
func EMLFilename(projectName string) string {
	return strings.TrimSuffix(Filename(projectName), ".md") + ".eml"
}

// ── helpers ───────────────────────────────────────────────────────────────────

func esc(s string) string { return gohtml.EscapeString(s) }

func writeEmailHeading(w io.Writer, level int, name string, status core.Status, style string) {
	tag := fmt.Sprintf("h%d", level)
	content := esc(name)
	if isDone(status) {
		content = "<s>" + content + "</s>"
	}
	fmt.Fprintf(w, `<%s style="%s">%s</%s>`, tag, style, content, tag)
}

func writeEmailMeta(w io.Writer, status core.Status, blockedReason, created, edited string, sameDate bool) {
	p := `style="margin:4px 0;font-size:13px;color:#888"`
	fmt.Fprintf(w, `<p %s><strong>Status:</strong> %s</p>`, p, esc(status.String()))
	if created != "" {
		fmt.Fprintf(w, `<p %s><strong>Created:</strong> %s</p>`, p, esc(created))
	}
	if !sameDate && edited != "" {
		fmt.Fprintf(w, `<p %s><strong>Last edited:</strong> %s</p>`, p, esc(edited))
	}
	if blockedReason != "" {
		fmt.Fprintf(w, `<p %s><strong>Blocked reason:</strong> %s</p>`, p, esc(blockedReason))
	}
}

func writeEmailBody(w io.Writer, description string) {
	if description != "" {
		fmt.Fprintf(w, `<p style="color:#555">%s</p>`, esc(description))
	}
}

func writeEmailTask(w io.Writer, t core.Task) {
	li := `style="margin:4px 0"`
	if isDone(t.Status) {
		fmt.Fprintf(w, `<li %s>&#x2611; <s>%s</s></li>`, li, esc(t.Title))
	} else {
		fmt.Fprintf(w, `<li %s>&#x2610; %s</li>`, li, esc(t.Title))
	}
	if t.BlockedReason != "" {
		fmt.Fprintf(w, `<li style="margin:4px 0;color:#e74c3c;padding-left:20px">&#x26A0; Blocked: %s</li>`,
			esc(t.BlockedReason))
	}
}

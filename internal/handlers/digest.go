package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/digest"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type DigestHandler struct {
	sessions  *scs.SessionManager
	tmpl      *templates.Templates
	generator *digest.Generator
}

func NewDigestHandler(sessions *scs.SessionManager, tmpl *templates.Templates, generator *digest.Generator) *DigestHandler {
	return &DigestHandler{sessions: sessions, tmpl: tmpl, generator: generator}
}

func (h *DigestHandler) Generate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	// Check if there's already an active job
	active, err := h.generator.ActiveJobForUser(user.ID)
	if err == nil && active != nil {
		http.Redirect(w, r, fmt.Sprintf("/digest/generating/%d", active.ID), http.StatusSeeOther)
		return
	}

	jobID, err := h.generator.CreateJob(user.ID)
	if err != nil {
		h.sessions.Put(r.Context(), "flash", "Failed to start generation: "+err.Error())
		today := time.Now().Format("2006-01-02")
		http.Redirect(w, r, "/digest/"+today, http.StatusSeeOther)
		return
	}

	go h.generator.GenerateAsync(jobID, user.ID)

	http.Redirect(w, r, fmt.Sprintf("/digest/generating/%d", jobID), http.StatusSeeOther)
}

func (h *DigestHandler) GeneratingPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	jobID := r.PathValue("jobID")

	var id int64
	fmt.Sscan(jobID, &id)

	job, err := h.generator.GetJob(id)
	if err != nil || job.UserID != user.ID {
		today := time.Now().Format("2006-01-02")
		http.Redirect(w, r, "/digest/"+today, http.StatusSeeOther)
		return
	}

	if job.Status == "complete" && job.DigestID.Valid {
		today := time.Now().Format("2006-01-02")
		h.sessions.Put(r.Context(), "flash", "Digest generated successfully.")
		http.Redirect(w, r, fmt.Sprintf("/digest/%s/%d", today, job.DigestID.Int64), http.StatusSeeOther)
		return
	}

	if job.Status == "failed" {
		h.sessions.Put(r.Context(), "flash", "Generation failed: "+job.Step)
		today := time.Now().Format("2006-01-02")
		http.Redirect(w, r, "/digest/"+today, http.StatusSeeOther)
		return
	}

	h.tmpl.Render(w, "generating", map[string]any{
		"Title":   "Generating Digest",
		"User":    user,
		"Job":     job,
		"JobID":   id,
		"Percent": jobPercent(job),
	})
}

func (h *DigestHandler) GeneratingStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	jobID := r.PathValue("jobID")

	var id int64
	fmt.Sscan(jobID, &id)

	job, err := h.generator.GetJob(id)
	if err != nil || job.UserID != user.ID {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="text-muted italic">Job not found.</div>`)
		return
	}

	today := time.Now().Format("2006-01-02")

	if job.Status == "complete" && job.DigestID.Valid {
		h.sessions.Put(r.Context(), "flash", "Digest generated successfully.")
		w.Header().Set("HX-Redirect", fmt.Sprintf("/digest/%s/%d", today, job.DigestID.Int64))
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, " ")
		return
	}

	if job.Status == "failed" {
		h.sessions.Put(r.Context(), "flash", "Generation failed: "+job.Step)
		w.Header().Set("HX-Redirect", fmt.Sprintf("/digest/%s", today))
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, " ")
		return
	}

	h.tmpl.RenderPartial(w, "generating_status", map[string]any{
		"Job":     job,
		"JobID":   id,
		"Percent": jobPercent(job),
	})
}

func jobPercent(job *digest.JobStatus) int {
	if job.TotalBatches == 0 {
		return 0
	}
	p := (job.CurrentBatch * 100) / job.TotalBatches
	if p > 100 {
		p = 100
	}
	return p
}

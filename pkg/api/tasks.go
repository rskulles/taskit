package api

import (
	"net/http"

	"github.com/rskulles/taskit/pkg/core"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	requirementID, ok := pathID(r, "requirementID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid requirementID")
		return
	}
	tasks, err := s.store.ListTasks(r.Context(), requirementID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	requirementID, ok := pathID(r, "requirementID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid requirementID")
		return
	}
	t, err := decode[core.Task](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.RequirementID = requirementID
	created, err := s.store.CreateTask(r.Context(), t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := decode[core.Task](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.ID = id
	updated, err := s.store.UpdateTask(r.Context(), t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

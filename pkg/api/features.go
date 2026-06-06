package api

import (
	"net/http"

	"github.com/rskulles/taskit/pkg/core"
)

func (s *Server) listFeatures(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(r, "projectID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid projectID")
		return
	}
	features, err := s.store.ListFeatures(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, features)
}

func (s *Server) createFeature(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(r, "projectID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid projectID")
		return
	}
	f, err := decode[core.Feature](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f.ProjectID = projectID
	created, err := s.store.CreateFeature(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getFeature(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, err := s.store.GetFeature(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) updateFeature(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, err := decode[core.Feature](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f.ID = id
	updated, err := s.store.UpdateFeature(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteFeature(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteFeature(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

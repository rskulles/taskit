package api

import (
	"net/http"

	"github.com/rskulles/taskit/pkg/core"
)

func (s *Server) listRequirements(w http.ResponseWriter, r *http.Request) {
	featureID, ok := pathID(r, "featureID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid featureID")
		return
	}
	reqs, err := s.store.ListRequirements(r.Context(), featureID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reqs)
}

func (s *Server) createRequirement(w http.ResponseWriter, r *http.Request) {
	featureID, ok := pathID(r, "featureID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid featureID")
		return
	}
	req, err := decode[core.Requirement](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.FeatureID = featureID
	created, err := s.store.CreateRequirement(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getRequirement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	req, err := s.store.GetRequirement(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *Server) updateRequirement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	req, err := decode[core.Requirement](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = id
	updated, err := s.store.UpdateRequirement(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteRequirement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteRequirement(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

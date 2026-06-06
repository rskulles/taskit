package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rskulles/taskit/pkg/core"
)

type Server struct {
	store  core.Store
	mux    *http.ServeMux
}

func NewServer(store core.Store) *Server {
	s := &Server{store: store, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /projects", s.listProjects)
	s.mux.HandleFunc("POST /projects", s.createProject)
	s.mux.HandleFunc("GET /projects/{id}", s.getProject)
	s.mux.HandleFunc("PUT /projects/{id}", s.updateProject)
	s.mux.HandleFunc("DELETE /projects/{id}", s.deleteProject)

	s.mux.HandleFunc("GET /projects/{projectID}/features", s.listFeatures)
	s.mux.HandleFunc("POST /projects/{projectID}/features", s.createFeature)
	s.mux.HandleFunc("GET /features/{id}", s.getFeature)
	s.mux.HandleFunc("PUT /features/{id}", s.updateFeature)
	s.mux.HandleFunc("DELETE /features/{id}", s.deleteFeature)

	s.mux.HandleFunc("GET /features/{featureID}/requirements", s.listRequirements)
	s.mux.HandleFunc("POST /features/{featureID}/requirements", s.createRequirement)
	s.mux.HandleFunc("GET /requirements/{id}", s.getRequirement)
	s.mux.HandleFunc("PUT /requirements/{id}", s.updateRequirement)
	s.mux.HandleFunc("DELETE /requirements/{id}", s.deleteRequirement)

	s.mux.HandleFunc("GET /requirements/{requirementID}/tasks", s.listTasks)
	s.mux.HandleFunc("POST /requirements/{requirementID}/tasks", s.createTask)
	s.mux.HandleFunc("GET /tasks/{id}", s.getTask)
	s.mux.HandleFunc("PUT /tasks/{id}", s.updateTask)
	s.mux.HandleFunc("DELETE /tasks/{id}", s.deleteTask)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func pathID(r *http.Request, key string) (int64, bool) {
	v := r.PathValue(key)
	id, err := strconv.ParseInt(v, 10, 64)
	return id, err == nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

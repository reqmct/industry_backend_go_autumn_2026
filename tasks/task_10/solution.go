package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"uuid"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task := Task{
		ID:        uuid.New().String(),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[task.ID] = task

	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]

	return task, ok
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Task
	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}

	return result
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done == done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()
	r.tasks[id] = task

	return task, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	const prefix = "/tasks/"
	if strings.HasPrefix(r.URL.Path, prefix) {
		id := r.URL.Path[len(prefix):]
		if strings.Contains(id, "/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, id)
		case http.MethodPatch:
			h.handlePatch(w, r, id)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

type createReq struct {
	Title *string `json:"title"`
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createReq
	err := decodeStrictJSON(r.Body, &req)
	if err != nil || req.Title == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*req.Title)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	if values, ok := r.URL.Query()["done"]; ok {
		if len(values) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch values[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	list := h.repo.List(done)
	if list == nil {
		list = []Task{}
	}
	sort.Slice(list, func(i, j int) bool {
		return !list[i].UpdatedAt.Equal(list[j].UpdatedAt) &&
			list[i].UpdatedAt.After(list[j].UpdatedAt) ||
			list[i].ID < list[j].ID
	})

	writeJSON(w, http.StatusOK, list)
}

type patchReq struct {
	Done *bool `json:"done"`
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req patchReq
	err := decodeStrictJSON(r.Body, &req)
	if err != nil || req.Done == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *req.Done)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(v)
	if err != nil {
		return err
	}

	var raw json.RawMessage
	err = decoder.Decode(&raw)
	if errors.Is(err, io.EOF) {
		return nil
	}

	return errors.New("invalid json body")
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidTitle):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type QuickstartEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type ListQuickstartsResponse struct {
	Quickstarts []QuickstartEntry `json:"quickstarts"`
}

type QuickstartContentResponse struct {
	Name  string `json:"name"`
	Files []File `json:"files"`
}

func (h *Handler) ListQuickstarts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.repoMgr.PullLatest(); err != nil {
		logrus.WithError(err).Error("Failed to pull latest")
		writeError(w, http.StatusInternalServerError, "failed to pull latest changes")
		return
	}

	dirs, err := h.repoMgr.ListDirectories(h.quickstartsDirPath)
	if err != nil {
		logrus.WithError(err).Error("Failed to list quickstart directories")
		writeError(w, http.StatusInternalServerError, "failed to list quickstarts")
		return
	}

	quickstarts := make([]QuickstartEntry, 0, len(dirs))
	for _, dir := range dirs {
		entry := QuickstartEntry{Name: dir}

		contentPath := filepath.Join(h.quickstartsDirPath, dir, dir+".yml")
		content, err := h.repoMgr.ReadFile(contentPath)
		if err != nil {
			contentPath = filepath.Join(h.quickstartsDirPath, dir, dir+".yaml")
			content, err = h.repoMgr.ReadFile(contentPath)
		}
		if err == nil {
			entry.DisplayName = extractDisplayName(content)
		}

		if entry.DisplayName == "" {
			entry.DisplayName = dir
		}

		quickstarts = append(quickstarts, entry)
	}

	json.NewEncoder(w).Encode(ListQuickstartsResponse{Quickstarts: quickstarts})
}

func (h *Handler) GetQuickstartContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "quickstart name is required")
		return
	}
	if containsTraversal(name) {
		writeError(w, http.StatusBadRequest, "name contains invalid path segment")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.repoMgr.PullLatest(); err != nil {
		logrus.WithError(err).Error("Failed to pull latest")
		writeError(w, http.StatusInternalServerError, "failed to pull latest changes")
		return
	}

	dirPath := filepath.Join(h.quickstartsDirPath, name)
	fileNames, err := h.repoMgr.ListFiles(dirPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "quickstart not found")
		return
	}

	files := make([]File, 0, len(fileNames))
	for _, fileName := range fileNames {
		content, err := h.repoMgr.ReadFile(filepath.Join(dirPath, fileName))
		if err != nil {
			logrus.WithError(err).WithField("file", fileName).Warn("Failed to read file")
			continue
		}
		files = append(files, File{Name: fileName, Content: content})
	}

	if len(files) == 0 {
		writeError(w, http.StatusNotFound, "no files found for quickstart")
		return
	}

	json.NewEncoder(w).Encode(QuickstartContentResponse{Name: name, Files: files})
}

func extractDisplayName(yamlContent string) string {
	var doc struct {
		Spec struct {
			DisplayName string `yaml:"displayName"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		return ""
	}
	return doc.Spec.DisplayName
}

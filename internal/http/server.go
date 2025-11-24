package http

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"leveltalk/internal/dialogs"
)

// Server wires HTTP routing for LevelTalk.
type Server struct {
	logger     *slog.Logger
	dialogs    *dialogs.Service
	templates  *template.Template
	staticFS   http.FileSystem
	languages  []string
	cefrLevels []string
}

// NewServer constructs a chi router implementing http.Handler.
func NewServer(logger *slog.Logger, service *dialogs.Service, templates *template.Template, staticFS http.FileSystem) http.Handler {
	srv := &Server{
		logger:     logger,
		dialogs:    service,
		templates:  templates,
		staticFS:   staticFS,
		languages:  []string{"ru", "en", "es", "fi", "de", "fr"},
		cefrLevels: []string{"A1", "A2", "B1", "B2", "C1", "C2"},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(srv.staticFS)))

	r.Get("/", srv.handleIndex)
	r.Post("/dialogs", srv.handleCreateDialog)
	r.Get("/dialogs/search", srv.handleSearch)
	r.Get("/dialogs/{id}", srv.handleDetail)
	r.Get("/dialogs/download/text", srv.handleDownloadText)
	r.Get("/dialogs/download/audio", srv.handleDownloadAudio)

	return r
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dialogsList, err := s.dialogs.SearchDialogs(ctx, dialogs.DialogFilter{Limit: 10})
	if err != nil {
		s.serverError(w, err)
		return
	}

	// Build query params for download links (empty for initial page)
	queryParams := ""

	payload := map[string]any{
		"Languages":   s.languages,
		"CEFRLevels":  s.cefrLevels,
		"Dialogs":     dialogsList,
		"QueryParams": queryParams,
	}
	s.renderPage(w, "LevelTalk — multilingual dialogs", "index.html", payload)
}

func (s *Server) handleCreateDialog(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.clientError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	input := dialogs.CreateDialogInput{
		InputLanguage:  r.FormValue("input_language"),
		DialogLanguage: r.FormValue("dialog_language"),
		CEFRLevel:      r.FormValue("cefr_level"),
		InputWords:     parseWords(r.FormValue("input_words")),
	}

	if _, err := s.dialogs.CreateDialog(r.Context(), input); err != nil {
		s.serverError(w, err)
		return
	}

	s.renderDialogList(w, r)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	s.renderDialogList(w, r)
}

func (s *Server) renderDialogList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := dialogs.DialogFilter{
		Limit: 20,
	}

	if v := strings.TrimSpace(r.FormValue("input_language")); v != "" {
		filter.InputLanguage = &v
	}
	if v := strings.TrimSpace(r.FormValue("dialog_language")); v != "" {
		filter.DialogLanguage = &v
	}
	if v := strings.TrimSpace(r.FormValue("cefr_level")); v != "" {
		filter.CEFRLevel = &v
	}

	results, err := s.dialogs.SearchDialogs(ctx, filter)
	if err != nil {
		s.serverError(w, err)
		return
	}

	// Build query params for download links
	queryParams := s.buildQueryParams(r)

	s.renderPartial(w, "dialogs_list.html", map[string]any{
		"Dialogs":     results,
		"QueryParams": queryParams,
	})
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	dialogID, err := uuid.Parse(idParam)
	if err != nil {
		s.clientError(w, http.StatusBadRequest, "invalid dialog id")
		return
	}

	dlg, err := s.dialogs.GetDialog(r.Context(), dialogID)
	if err != nil {
		if errors.Is(err, dialogs.ErrNotFound) {
			s.clientError(w, http.StatusNotFound, "dialog not found")
			return
		}
		s.serverError(w, err)
		return
	}

	s.renderPage(w, "LevelTalk — dialog detail", "dialog_detail.html", map[string]any{
		"Dialog": dlg,
	})
}

type pageView struct {
	Title string
	Body  template.HTML
}

func (s *Server) renderPage(w http.ResponseWriter, title, contentTemplate string, payload any) {
	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, contentTemplate, payload); err != nil {
		s.logger.Error("render template failed", slog.String("template", contentTemplate), slog.String("error", err.Error()))
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	data := pageView{
		Title: title,
		Body:  template.HTML(body.String()),
	}
	s.executeTemplate(w, "base.html", data)
}

func (s *Server) renderPartial(w http.ResponseWriter, templateName string, data any) {
	s.executeTemplate(w, templateName, data)
}

func (s *Server) executeTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Error("render template failed", slog.String("template", name), slog.String("error", err.Error()))
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	s.logger.Error("request error", slog.String("error", err.Error()))
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (s *Server) clientError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

func (s *Server) handleDownloadText(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var dialogsList []dialogs.Dialog
	var err error
	
	// Check if specific IDs are provided
	selectedIDs := r.URL.Query()["id"]
	if len(selectedIDs) > 0 {
		// Fetch specific dialogs by ID
		dialogsList = make([]dialogs.Dialog, 0, len(selectedIDs))
		for _, idStr := range selectedIDs {
			id, parseErr := uuid.Parse(idStr)
			if parseErr != nil {
				s.logger.Warn("invalid dialog id in download request", slog.String("id", idStr), slog.String("error", parseErr.Error()))
				continue
			}
			dlg, fetchErr := s.dialogs.GetDialog(ctx, id)
			if fetchErr != nil {
				s.logger.Warn("failed to fetch dialog", slog.String("id", idStr), slog.String("error", fetchErr.Error()))
				continue
			}
			dialogsList = append(dialogsList, dlg)
		}
	} else {
		// Use filter-based search
		filter := s.buildFilterFromRequest(r)
		dialogsList, err = s.dialogs.SearchDialogs(ctx, filter)
		if err != nil {
			s.serverError(w, err)
			return
		}
	}

	if len(dialogsList) == 0 {
		s.clientError(w, http.StatusNotFound, "no dialogs found")
		return
	}

	var buf bytes.Buffer
	buf.WriteString("LevelTalk Dialog Export\n")
	buf.WriteString("=======================\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC822)))
	buf.WriteString(fmt.Sprintf("Total dialogs: %d\n\n", len(dialogsList)))

	for i, dlg := range dialogsList {
		// Use title if available, otherwise generate from metadata
		dialogName := dlg.Title
		if dialogName == "" {
			dialogName = fmt.Sprintf("%s→%s %s",
				strings.ToUpper(dlg.InputLanguage),
				strings.ToUpper(dlg.DialogLanguage),
				dlg.CEFRLevel,
			)
			if len(dlg.InputWords) > 0 {
				firstWord := strings.TrimSpace(dlg.InputWords[0])
				if len(firstWord) > 15 {
					firstWord = firstWord[:15] + "..."
				}
				dialogName = fmt.Sprintf("%s - %s", dialogName, firstWord)
			}
		}

		buf.WriteString(fmt.Sprintf("Dialog %d: %s\n", i+1, dialogName))
		buf.WriteString(strings.Repeat("-", 40) + "\n")
		buf.WriteString(fmt.Sprintf("ID: %s\n", dlg.ID.String()))
		buf.WriteString(fmt.Sprintf("Input Language: %s\n", dlg.InputLanguage))
		buf.WriteString(fmt.Sprintf("Dialog Language: %s\n", dlg.DialogLanguage))
		buf.WriteString(fmt.Sprintf("CEFR Level: %s\n", dlg.CEFRLevel))
		buf.WriteString(fmt.Sprintf("Created: %s\n", dlg.CreatedAt.Format(time.RFC822)))

		if len(dlg.InputWords) > 0 {
			buf.WriteString("\nVocabulary:\n")
			for _, word := range dlg.InputWords {
				if trans, ok := dlg.Translations[word]; ok && trans != "" {
					buf.WriteString(fmt.Sprintf("  %s → %s\n", word, trans))
				} else {
					buf.WriteString(fmt.Sprintf("  %s\n", word))
				}
			}
		}

		buf.WriteString("\nDialog:\n")
		for _, turn := range dlg.Turns {
			buf.WriteString(fmt.Sprintf("%s: %s\n", turn.Speaker, turn.Text))
		}
		buf.WriteString("\n\n")
	}

	filename := fmt.Sprintf("leveltalk-dialogs-%s.txt", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(buf.Bytes())
}

func (s *Server) handleDownloadAudio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var dialogsList []dialogs.Dialog
	var err error
	
	// Check if specific IDs are provided
	selectedIDs := r.URL.Query()["id"]
	if len(selectedIDs) > 0 {
		// Fetch specific dialogs by ID
		dialogsList = make([]dialogs.Dialog, 0, len(selectedIDs))
		for _, idStr := range selectedIDs {
			id, parseErr := uuid.Parse(idStr)
			if parseErr != nil {
				s.logger.Warn("invalid dialog id in download request", slog.String("id", idStr), slog.String("error", parseErr.Error()))
				continue
			}
			dlg, fetchErr := s.dialogs.GetDialog(ctx, id)
			if fetchErr != nil {
				s.logger.Warn("failed to fetch dialog", slog.String("id", idStr), slog.String("error", fetchErr.Error()))
				continue
			}
			dialogsList = append(dialogsList, dlg)
		}
	} else {
		// Use filter-based search
		filter := s.buildFilterFromRequest(r)
		dialogsList, err = s.dialogs.SearchDialogs(ctx, filter)
		if err != nil {
			s.serverError(w, err)
			return
		}
	}

	if len(dialogsList) == 0 {
		s.clientError(w, http.StatusNotFound, "no dialogs found")
		return
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	audioCount := 0
	for _, dlg := range dialogsList {
		for _, turn := range dlg.Turns {
			if turn.AudioURL == "" {
				continue
			}

			// Extract base64 data from data URL
			var audioData []byte
			if strings.HasPrefix(turn.AudioURL, "data:audio/") {
				// Format: data:audio/mpeg;base64,<data>
				parts := strings.SplitN(turn.AudioURL, ",", 2)
				if len(parts) == 2 {
					var err error
					audioData, err = base64.StdEncoding.DecodeString(parts[1])
					if err != nil {
						s.logger.Warn("failed to decode audio data",
							slog.String("dialog_id", dlg.ID.String()),
							slog.String("turn_id", turn.ID.String()),
							slog.String("error", err.Error()),
						)
						continue
					}
				}
			} else {
				// Skip non-data URLs (like placeholder.mp3)
				continue
			}

			if len(audioData) == 0 {
				continue
			}

			// Create folder structure: dialog_name/turn_position-speaker.mp3
			// Use title if available, otherwise generate from metadata
			var dialogFolder string
			if dlg.Title != "" {
				dialogFolder = sanitizeFilename(dlg.Title)
			} else {
				dialogFolder = sanitizeFilename(fmt.Sprintf("%s-%s-%s",
					strings.ToUpper(dlg.InputLanguage),
					strings.ToUpper(dlg.DialogLanguage),
					dlg.CEFRLevel,
				))
				if len(dlg.InputWords) > 0 {
					firstWord := sanitizeFilename(strings.TrimSpace(dlg.InputWords[0]))
					if len(firstWord) > 20 {
						firstWord = firstWord[:20]
					}
					dialogFolder = fmt.Sprintf("%s-%s", dialogFolder, firstWord)
				}
			}

			filename := fmt.Sprintf("%s/%02d-%s.mp3",
				dialogFolder,
				turn.Position,
				strings.ReplaceAll(turn.Speaker, " ", "_"),
			)

			file, err := zipWriter.Create(filename)
			if err != nil {
				s.logger.Error("failed to create zip entry", slog.String("error", err.Error()))
				continue
			}

			if _, err := file.Write(audioData); err != nil {
				s.logger.Error("failed to write audio to zip", slog.String("error", err.Error()))
				continue
			}

			audioCount++
		}
	}

	if err := zipWriter.Close(); err != nil {
		s.serverError(w, fmt.Errorf("close zip: %w", err))
		return
	}

	if audioCount == 0 {
		s.clientError(w, http.StatusNotFound, "no audio files found in dialogs")
		return
	}

	filename := fmt.Sprintf("leveltalk-audio-%s.zip", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(zipBuf.Bytes())
}

func (s *Server) buildQueryParams(r *http.Request) string {
	var params []string
	if v := strings.TrimSpace(r.FormValue("input_language")); v != "" {
		params = append(params, fmt.Sprintf("input_language=%s", v))
	}
	if v := strings.TrimSpace(r.FormValue("dialog_language")); v != "" {
		params = append(params, fmt.Sprintf("dialog_language=%s", v))
	}
	if v := strings.TrimSpace(r.FormValue("cefr_level")); v != "" {
		params = append(params, fmt.Sprintf("cefr_level=%s", v))
	}
	if len(params) > 0 {
		return strings.Join(params, "&")
	}
	return ""
}

func (s *Server) buildFilterFromRequest(r *http.Request) dialogs.DialogFilter {
	filter := dialogs.DialogFilter{
		Limit: 1000, // Allow more for downloads
	}

	// Try query params first (for download links), then form values (for search)
	var inputLang, dialogLang, cefr string
	if v := r.URL.Query().Get("input_language"); v != "" {
		inputLang = strings.TrimSpace(v)
	} else if v := r.FormValue("input_language"); v != "" {
		inputLang = strings.TrimSpace(v)
	}

	if v := r.URL.Query().Get("dialog_language"); v != "" {
		dialogLang = strings.TrimSpace(v)
	} else if v := r.FormValue("dialog_language"); v != "" {
		dialogLang = strings.TrimSpace(v)
	}

	if v := r.URL.Query().Get("cefr_level"); v != "" {
		cefr = strings.TrimSpace(v)
	} else if v := r.FormValue("cefr_level"); v != "" {
		cefr = strings.TrimSpace(v)
	}

	if inputLang != "" {
		filter.InputLanguage = &inputLang
	}
	if dialogLang != "" {
		filter.DialogLanguage = &dialogLang
	}
	if cefr != "" {
		filter.CEFRLevel = &cefr
	}

	return filter
}

func sanitizeFilename(name string) string {
	// Remove/replace characters that are problematic in filenames
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

func parseWords(raw string) []string {
	// First try splitting by comma (preferred format)
	if strings.Contains(raw, ",") {
		fields := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		result := make([]string, 0, len(fields))
		for _, field := range fields {
			trimmed := strings.TrimSpace(field)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	// If no commas, split by whitespace
	fields := strings.Fields(raw)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

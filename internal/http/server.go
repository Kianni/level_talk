package http

import (
	"bytes"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

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

	return r
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dialogsList, err := s.dialogs.SearchDialogs(ctx, dialogs.DialogFilter{Limit: 10})
	if err != nil {
		s.serverError(w, err)
		return
	}

	payload := map[string]any{
		"Languages":  s.languages,
		"CEFRLevels": s.cefrLevels,
		"Dialogs":    dialogsList,
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

	s.renderPartial(w, "dialogs_list.html", map[string]any{
		"Dialogs": results,
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

func parseWords(raw string) []string {
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

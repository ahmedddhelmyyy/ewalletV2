package handler

import (
	"html/template"
	"net/http"
	"time"
)

type Renderer struct {
	templates *template.Template
}

type PayPageData struct {
	RedirectToken string
	OrderID       string
	Amount        string
	Currency      string
	ExpiresAt     time.Time
	Status        string
	ErrorMessage  string
	MerchantName  string
	FinalState    bool
	UserID        string
}

func NewRenderer(templatePath string) (*Renderer, error) {
	tmpl, err := template.ParseGlob(templatePath + "/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: tmpl}, nil
}

func (r *Renderer) Execute(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}
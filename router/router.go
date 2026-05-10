///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/router/router.go
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"       // ← add
	_ "github.com/ewallet/docs"                        // ← add (swag generated)
	"github.com/ewallet/handler"
	"github.com/ewallet/middleware"
)

// Dependencies bundles all HTTP handlers needed to register routes.
type Dependencies struct {
	AuthHandler        *handler.AuthHandler
	WalletHandler      *handler.WalletHandler
	TransactionHandler *handler.TransactionHandler
	BillHandler        *handler.BillHandler
	ExpenseHandler     *handler.ExpenseHandler
	JWTSecret          string
}

// New creates and returns the fully configured chi router.
func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger)

	// ── Swagger UI ─────────────────────────────────────────────────────────────
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// ── API v1 ─────────────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", deps.AuthHandler.Register)
			r.Post("/login", deps.AuthHandler.Login)
			r.Post("/refresh", deps.AuthHandler.RefreshToken)
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.JWTSecret))
				r.Post("/logout", deps.AuthHandler.Logout)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.JWTSecret))

			r.Get("/wallet", deps.WalletHandler.GetWallet)

			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", deps.TransactionHandler.GetHistory)
				r.Post("/send", deps.TransactionHandler.Send)
				r.Post("/top-up", deps.TransactionHandler.TopUp)
				r.Post("/withdraw", deps.TransactionHandler.Withdraw)
				r.Get("/{transaction_id}", deps.TransactionHandler.GetByID)
			})

			r.Route("/bills", func(r chi.Router) {
				r.Get("/", deps.BillHandler.List)
				r.Post("/", deps.BillHandler.Create)
				r.Get("/{bill_id}", deps.BillHandler.GetByID)
				r.Post("/{bill_id}/pay", deps.BillHandler.Pay)
				r.Delete("/{bill_id}", deps.BillHandler.Delete)
			})

			r.Route("/expenses", func(r chi.Router) {
				r.Get("/summary", deps.ExpenseHandler.GetSummary)
				r.Get("/flow", deps.ExpenseHandler.GetFlow)
			})
		})
	})

	return r
}
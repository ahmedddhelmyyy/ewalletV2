package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger" // ← add
	_ "github.com/ewallet/docs"                  // ← add (swag generated)
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
	MerchantHandler    *handler.MerchantHandler
	PayHandler         *handler.PayHandler
	JWTSecret          string
}

// New creates and returns the fully configured chi router.
func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:45678"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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

		r.Route("/merchant", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuthenticateMerchant)
				r.Post("/transactions", deps.MerchantHandler.CreateTransaction)
				r.Get("/transactions/{transaction_id}", deps.MerchantHandler.GetTransaction)
			})
		})
	})

	r.Route("/pay", func(r chi.Router) {
		r.Get("/{redirect_token}", deps.PayHandler.Show)
		r.Post("/{redirect_token}/confirm", deps.PayHandler.Confirm)
		r.Post("/{redirect_token}/cancel", deps.PayHandler.Cancel)
	})

	return r
}
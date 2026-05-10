///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "github.com/ewallet/docs" // swag generated — import side-effect

	"github.com/ewallet/config"
	"github.com/ewallet/handler"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
	"github.com/ewallet/router"
	"github.com/ewallet/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title           eWallet API
// @version         1.0
// @description     Production-ready eWallet backend — auth, wallets, transactions, bills, expense tracking.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Backend Team
// @contact.email  backend@ewallet.com

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and your access token. Example: "Bearer eyJhbG..."
func main() {
	// ── Load config ────────────────────────────────────────────────────────────
	cfg := config.Load()

	// ── Connect to PostgreSQL via GORM ─────────────────────────────────────────
	logLevel := logger.Silent
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("FATAL: failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("FATAL: failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// ── Auto-migrate GORM models ───────────────────────────────────────────────
	// GORM will CREATE / ALTER tables to match the current model definitions.
	log.Println("Running database migrations...")
	if err := db.AutoMigrate(
		&model.User{},
		&model.Wallet{},
		&model.Transaction{},
		&model.Bill{},
		&model.RefreshToken{},
	); err != nil {
		log.Fatalf("FATAL: auto-migrate failed: %v", err)
	}
	log.Println("Migrations complete.")

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo    := repository.NewUserRepository(db)
	walletRepo  := repository.NewWalletRepository(db)
	txRepo      := repository.NewTransactionRepository(db)
	billRepo    := repository.NewBillRepository(db)
	tokenRepo   := repository.NewRefreshTokenRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc    := service.NewAuthService(db, cfg, userRepo, walletRepo, tokenRepo)
	walletSvc  := service.NewWalletService(walletRepo)
	txSvc      := service.NewTransactionService(db, walletRepo, txRepo, userRepo)
	billSvc    := service.NewBillService(db, billRepo, walletRepo, txRepo)
	expenseSvc := service.NewExpenseService(walletRepo, txRepo)

	// ── Handlers ──────────────────F────────────────────────────────────────────
	authHandler    := handler.NewAuthHandler(authSvc)
	walletHandler  := handler.NewWalletHandler(walletSvc)
	txHandler      := handler.NewTransactionHandler(txSvc)
	billHandler    := handler.NewBillHandler(billSvc)
	expenseHandler := handler.NewExpenseHandler(expenseSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.New(router.Dependencies{
		AuthHandler:        authHandler,
		WalletHandler:      walletHandler,
		TransactionHandler: txHandler,
		BillHandler:        billHandler,
		ExpenseHandler:     expenseHandler,
		JWTSecret:          cfg.JWTSecret,
	})

	// ── Start server ──────────────────────────────────────────────────────────
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("eWallet API listening on http://localhost%s/api/v1", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("FATAL: server failed: %v", err)
	}
}

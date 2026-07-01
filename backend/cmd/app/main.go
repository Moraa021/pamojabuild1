package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"

	authHandler "pamojabuild1/backend/internal/auth/delivery/http"
	authRepo "pamojabuild1/backend/internal/auth/repository"
	authService "pamojabuild1/backend/internal/auth/service"

	volunteerHandler "pamojabuild1/backend/internal/volunteer/delivery/http"
	volunteerRepo "pamojabuild1/backend/internal/volunteer/repository"
	volunteerService "pamojabuild1/backend/internal/volunteer/service"

	escrowHandler "pamojabuild1/backend/internal/escrow/delivery/http"
	escrowRepo "pamojabuild1/backend/internal/escrow/repository"
	escrowService "pamojabuild1/backend/internal/escrow/service"

	ledgerHandler "pamojabuild1/backend/internal/ledger/delivery/http"
	ledgerRepo "pamojabuild1/backend/internal/ledger/repository"
	ledgerService "pamojabuild1/backend/internal/ledger/service"

	lightningHandler "pamojabuild1/backend/internal/lightning/delivery/http"
	lightningRepo "pamojabuild1/backend/internal/lightning/repository"
	lightningService "pamojabuild1/backend/internal/lightning/service"

	taskHandler "pamojabuild1/backend/internal/task/delivery/http"
	taskRepo "pamojabuild1/backend/internal/task/repository"
	taskService "pamojabuild1/backend/internal/task/service"

	trusteeHandler "pamojabuild1/backend/internal/trustee/delivery/http"
	trusteeRepo "pamojabuild1/backend/internal/trustee/repository"
	trusteeService "pamojabuild1/backend/internal/trustee/service"
)

func runMigrations(db *sql.DB, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var sqlFiles []string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)
	for _, f := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		log.Printf("Running migration: %s", f)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", f, err)
		}
	}
	log.Println("Migrations complete")
	return nil
}

func main() {
	cfg := config.Load()

	db, err := config.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := runMigrations(db, "db/migrations"); err != nil {
		log.Printf("Warning: migration error: %v", err)
	}

	eventBus := events.NewEventBus()

	authRepo := authRepo.NewAuthRepository(db)
	authSvc := authService.NewAuthService(authRepo, cfg.JWTSecret)
	authH := authHandler.NewAuthHandler(authSvc)

	profileRepo := volunteerRepo.NewProfileRepository(db)
	applicationRepo := volunteerRepo.NewApplicationRepository(db)
	submissionRepo := volunteerRepo.NewSubmissionRepository(db)
	paymentRepo := volunteerRepo.NewPaymentRepository(db)

	volunteerSvc := volunteerService.NewVolunteerService(profileRepo)
	applicationSvc := volunteerService.NewApplicationService(applicationRepo)
	submissionSvc := volunteerService.NewSubmissionService(submissionRepo, applicationRepo)
	reputationSvc := volunteerService.NewReputationService(profileRepo, applicationRepo, submissionRepo, paymentRepo)
	volunteerH := volunteerHandler.NewVolunteerHandler(volunteerSvc, applicationSvc, submissionSvc, reputationSvc)

	taskRepo := taskRepo.NewTaskRepository(db)
	taskSvc := taskService.NewTaskService(taskRepo, eventBus)
	taskH := taskHandler.NewTaskHandler(taskSvc)

	trusteeRepo := trusteeRepo.NewTrusteeRepository(db)
	trusteeSvc := trusteeService.NewTrusteeService(trusteeRepo, trusteeRepo)
	trusteeH := trusteeHandler.NewTrusteeHandler(trusteeSvc)

	lightningRepo := lightningRepo.NewLightningRepository(db)
	lightningSvc := lightningService.NewLightningService(lightningRepo, cfg)
	lightningH := lightningHandler.NewLightningHandler(lightningSvc)

	ledgerRepo := ledgerRepo.NewLedgerRepository(db)
	ledgerSvc := ledgerService.NewLedgerService(ledgerRepo, cfg.ServerSecret)
	ledgerH := ledgerHandler.NewLedgerHandler(ledgerSvc)

	escrowRepo := escrowRepo.NewEscrowRepository(db)
	escrowSvc := escrowService.NewEscrowService(escrowRepo, trusteeRepo, ledgerRepo)
	escrowH := escrowHandler.NewEscrowHandler(escrowSvc)

	eventBus.Subscribe(events.PaymentSettled, func(event events.Event) {
		payload := event.Payload.(events.PaymentSettledPayload)
		ctx := context.Background()
		ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "INBOUND_DONATION", payload.AmountSats, payload.PaymentHash)
	})

	eventBus.Subscribe(events.ThresholdReached, func(event events.Event) {
		payload := event.Payload.(events.ThresholdReachedPayload)
		ctx := context.Background()
		escrowSvc.FinalizeAndBroadcastPayout(ctx, payload.TaskSlug)
	})

	eventBus.Subscribe(events.FinancialStateChanged, func(event events.Event) {
		payload := event.Payload.(events.FinancialStateChangedPayload)
		log.Printf("Financial state changed for %s: %s -> %s", payload.TaskSlug, payload.OldState, payload.NewState)
	})

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/signin", authH.SignIn)
		}

		protected := api.Group("")
		protected.Use(authHandler.AuthMiddleware(authSvc))
		{
			protected.POST("/auth/signout", authH.SignOut)

			tasks := protected.Group("/tasks")
			{
				tasks.GET("", taskH.ListTasks)
				tasks.POST("", taskH.CreateTask)
				tasks.GET("/:slug", taskH.GetTask)
				tasks.POST("/:slug/apply", volunteerH.ApplyForTask)
				tasks.POST("/:slug/submissions", volunteerH.SubmitWork)
				tasks.POST("/:slug/trustees", trusteeH.RegisterTrusteeKeys)
				tasks.POST("/:slug/donate", lightningH.RequestDonationInvoice)
			}

			volunteers := protected.Group("/volunteers")
			{
				volunteers.GET("/profile", volunteerH.GetProfile)
				volunteers.PUT("/profile", volunteerH.UpdateProfile)
				volunteers.GET("/applications", volunteerH.GetApplications)
				volunteers.GET("/submissions", volunteerH.GetSubmissions)
				volunteers.GET("/payments", volunteerH.GetPayments)
				volunteers.PUT("/payment-profile", volunteerH.UpdatePaymentProfile)
				volunteers.GET("/reputation", volunteerH.GetReputation)
			}

			trustees := protected.Group("/trustees")
			{
				trustees.GET("/payouts/:slug", escrowH.GetPayoutReviewManifest)
				trustees.POST("/payouts/:slug/sign", escrowH.SubmitCoSignatures)
			}

			ledger := protected.Group("/ledger")
			{
				ledger.GET("/tasks/:slug", ledgerH.GetTaskBalance)
				ledger.GET("/tasks/:slug/verify", ledgerH.VerifyChainIntegrity)
			}
		}
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

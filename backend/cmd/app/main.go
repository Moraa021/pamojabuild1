package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"

	// Auth
	authDomain "pamojabuild1/backend/internal/auth"
	authHandler "pamojabuild1/backend/internal/auth/delivery/http"
	authRepo "pamojabuild1/backend/internal/auth/repository"
	authService "pamojabuild1/backend/internal/auth/service"

	// Volunteer
	volunteerHandler "pamojabuild1/backend/internal/volunteer/delivery/http"
	volunteerRepo "pamojabuild1/backend/internal/volunteer/repository"
	volunteerService "pamojabuild1/backend/internal/volunteer/service"

	// Existing packages
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

func main() {
	// Load config
	cfg := config.Load()

	// Initialize database
	db, err := config.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize event bus
	eventBus := events.NewEventBus()

	// Initialize repositories
	authRepo := authRepo.NewAuthRepository(db)
	volunteerRepo := volunteerRepo.NewVolunteerRepository(db)
	taskRepo := taskRepo.NewTaskRepository(db)
	trusteeRepo := trusteeRepo.NewTrusteeRepository(db)
	lightningRepo := lightningRepo.NewLightningRepository(db)
	ledgerRepo := ledgerRepo.NewLedgerRepository(db)
	escrowRepo := escrowRepo.NewEscrowRepository(db)

	// Initialize services
	authSvc := authService.NewAuthService(authRepo, cfg.JWTSecret)
	volunteerSvc := volunteerService.NewVolunteerService(volunteerRepo)
	applicationSvc := volunteerService.NewApplicationService(volunteerRepo)
	submissionSvc := volunteerService.NewSubmissionService(volunteerRepo, volunteerRepo)
	reputationSvc := volunteerService.NewReputationService(volunteerRepo, volunteerRepo, volunteerRepo, volunteerRepo)
	taskSvc := taskService.NewTaskService(taskRepo, eventBus)
	trusteeSvc := trusteeService.NewTrusteeService(trusteeRepo)
	lightningSvc := lightningService.NewLightningService(lightningRepo, cfg)
	ledgerSvc := ledgerService.NewLedgerService(ledgerRepo, cfg.ServerSecret)
	escrowSvc := escrowService.NewEscrowService(escrowRepo, trusteeRepo, ledgerRepo)

	// Initialize handlers
	authH := authHandler.NewAuthHandler(authSvc)
	volunteerH := volunteerHandler.NewVolunteerHandler(volunteerSvc, applicationSvc, submissionSvc, reputationSvc)
	taskH := taskHandler.NewTaskHandler(taskSvc)
	trusteeH := trusteeHandler.NewTrusteeHandler(trusteeSvc)
	lightningH := lightningHandler.NewLightningHandler(lightningSvc)
	ledgerH := ledgerHandler.NewLedgerHandler(ledgerSvc)
	escrowH := escrowHandler.NewEscrowHandler(escrowSvc)

	// Subscribe to events
	eventBus.Subscribe(events.PaymentSettled, func(event events.Event) {
		// Handle payment settlement
		payload := event.Payload.(events.PaymentSettledPayload)
		ledgerSvc.RecordValidatedTransaction(nil, payload.TaskSlug, "INBOUND_DONATION", payload.AmountSats, payload.PaymentHash)
	})

	eventBus.Subscribe(events.ThresholdReached, func(event events.Event) {
		// Handle multi-sig threshold
		payload := event.Payload.(events.ThresholdReachedPayload)
		escrowSvc.FinalizeAndBroadcastPayout(nil, payload.TaskSlug)
	})

	// Setup router
	router := gin.Default()

	// Public routes
	api := router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/signin", authH.SignIn)
			auth.POST("/signout", authH.SignOut)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(authHandler.AuthMiddleware(authSvc))
		{
			// Task routes
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

			// Volunteer routes
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

			// Trustee routes
			trustees := protected.Group("/trustees")
			{
				trustees.GET("/payouts/:slug", escrowH.GetPayoutReviewManifest)
				trustees.POST("/payouts/:slug/sign", escrowH.SubmitCoSignatures)
			}

			// Ledger routes
			ledger := protected.Group("/ledger")
			{
				ledger.GET("/tasks/:slug", ledgerH.GetTaskBalance)
				ledger.GET("/tasks/:slug/verify", ledgerH.VerifyChainIntegrity)
			}
		}
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
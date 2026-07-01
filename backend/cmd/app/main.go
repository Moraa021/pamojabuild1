package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"

	// Auth
	authHandler "pamojabuild1/backend/internal/auth/delivery/http"
	authRepo "pamojabuild1/backend/internal/auth/repository"
	authService "pamojabuild1/backend/internal/auth/service"

	// Volunteer
	volunteerHandler "pamojabuild1/backend/internal/volunteer/delivery/http"
	volunteerRepo "pamojabuild1/backend/internal/volunteer/repository"
	volunteerService "pamojabuild1/backend/internal/volunteer/service"

	// Escrow
	escrowHandler "pamojabuild1/backend/internal/escrow/delivery/http"
	escrowRepo "pamojabuild1/backend/internal/escrow/repository"
	escrowService "pamojabuild1/backend/internal/escrow/service"

	// Ledger
	ledgerHandler "pamojabuild1/backend/internal/ledger/delivery/http"
	ledgerRepo "pamojabuild1/backend/internal/ledger/repository"
	ledgerService "pamojabuild1/backend/internal/ledger/service"

	// Lightning
	lightningHandler "pamojabuild1/backend/internal/lightning/delivery/http"
	lightningRepo "pamojabuild1/backend/internal/lightning/repository"
	lightningService "pamojabuild1/backend/internal/lightning/service"

	// Task
	taskHandler "pamojabuild1/backend/internal/task/delivery/http"
	taskRepo "pamojabuild1/backend/internal/task/repository"
	taskService "pamojabuild1/backend/internal/task/service"

	// Trustee
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
	trusteeSvc := trusteeService.NewTrusteeService(trusteeRepo, trusteeRepo)
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
		payload, ok := event.Payload.(events.PaymentSettledPayload)
		if !ok {
			log.Println("Invalid payload type for PaymentSettled event")
			return
		}
		ctx := context.Background()
		if err := ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "INBOUND_DONATION", payload.AmountSats, payload.PaymentHash); err != nil {
			log.Printf("Failed to record transaction: %v", err)
		}
	})

	eventBus.Subscribe(events.ThresholdReached, func(event events.Event) {
		payload, ok := event.Payload.(events.ThresholdReachedPayload)
		if !ok {
			log.Println("Invalid payload type for ThresholdReached event")
			return
		}
		ctx := context.Background()
		if err := escrowSvc.FinalizeAndBroadcastPayout(ctx, payload.TaskSlug); err != nil {
			log.Printf("Failed to finalize payout: %v", err)
		}
	})

	eventBus.Subscribe(events.FinancialStateChanged, func(event events.Event) {
		payload, ok := event.Payload.(events.FinancialStateChangedPayload)
		if !ok {
			log.Println("Invalid payload type for FinancialStateChanged event")
			return
		}
		log.Printf("Financial state changed for %s: %s -> %s", payload.TaskSlug, payload.OldState, payload.NewState)
	})

	// Setup router
	router := gin.Default()

	// CORS middleware
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

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/signin", authH.SignIn)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(authHandler.AuthMiddleware(authSvc))
		{
			// Auth routes (protected)
			protected.POST("/auth/signout", authH.SignOut)

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
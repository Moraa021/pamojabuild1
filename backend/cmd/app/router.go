package main

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/gin-gonic/gin"

    "pamojabuild1/backend/internal/config"
    "pamojabuild1/backend/internal/events"
    "pamojabuild1/backend/internal/middleware"

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

func NewRouter(db *sql.DB, cfg *config.Config) *gin.Engine {
    eventBus := events.NewEventBus()

    authRepo := authRepo.NewAuthRepository(db)
    authSvc := authService.NewAuthService(authRepo, cfg.JWTSecret)
    authH := authHandler.NewAuthHandler(authSvc)

    profileRepo := volunteerRepo.NewProfileRepository(db)
    applicationRepo := volunteerRepo.NewApplicationRepository(db)
    submissionRepo := volunteerRepo.NewSubmissionRepository(db)
    paymentRepo := volunteerRepo.NewPaymentRepository(db)

    volunteerSvc := volunteerService.NewVolunteerService(profileRepo)
    applicationSvc := volunteerService.NewApplicationService(applicationRepo, eventBus)
    submissionSvc := volunteerService.NewSubmissionService(submissionRepo, applicationRepo, eventBus)
    reputationSvc := volunteerService.NewReputationService(profileRepo, applicationRepo, submissionRepo, paymentRepo)
    volunteerH := volunteerHandler.NewVolunteerHandler(volunteerSvc, applicationSvc, submissionSvc, reputationSvc)

    taskRepo := taskRepo.NewTaskRepository(db)
    taskSvc := taskService.NewTaskService(taskRepo, eventBus)
    taskH := taskHandler.NewTaskHandler(taskSvc)

    trusteeRepo := trusteeRepo.NewTrusteeRepository(db)
    trusteeSvc := trusteeService.NewTrusteeService(trusteeRepo, trusteeRepo)
    trusteeH := trusteeHandler.NewTrusteeHandler(trusteeSvc)

    lightningRepo := lightningRepo.NewLightningRepository(db)
    lightningSvc := lightningService.NewLightningService(lightningRepo, cfg, eventBus)
    lightningH := lightningHandler.NewLightningHandler(lightningSvc)

    ledgerRepo := ledgerRepo.NewLedgerRepository(db)
    ledgerSvc := ledgerService.NewLedgerService(ledgerRepo, cfg.ServerSecret)
    ledgerH := ledgerHandler.NewLedgerHandler(ledgerSvc)

    escrowRepo := escrowRepo.NewEscrowRepository(db)
    escrowSvc := escrowService.NewEscrowService(escrowRepo, trusteeRepo, ledgerRepo, eventBus)
    escrowH := escrowHandler.NewEscrowHandler(escrowSvc)

    eventBus.Subscribe(events.PaymentSettled, func(event events.Event) {
        payload := event.Payload.(events.PaymentSettledPayload)
        ctx := context.Background()
        ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "INBOUND_DONATION", payload.AmountSats, payload.PaymentHash)
    })

    eventBus.Subscribe(events.TaskCreated, func(event events.Event) {
        payload := event.Payload.(events.TaskCreatedPayload)
        ctx := context.Background()
        ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "TASK_CREATED", 0, payload.TaskSlug)
    })

    eventBus.Subscribe(events.ApplicationSubmitted, func(event events.Event) {
        payload := event.Payload.(events.ApplicationSubmittedPayload)
        ctx := context.Background()
        ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "APPLICATION_SUBMITTED", 0, fmt.Sprintf("volunteer-%d", payload.VolunteerID))
    })

    eventBus.Subscribe(events.SubmissionCreated, func(event events.Event) {
        payload := event.Payload.(events.SubmissionCreatedPayload)
        ctx := context.Background()
        if err := taskSvc.TransitionVolunteerStatus(ctx, payload.TaskSlug, "submitted"); err != nil {
            fmt.Printf("failed to transition task status on submission: %v\n", err)
        }
        ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "SUBMISSION_CREATED", 0, fmt.Sprintf("volunteer-%d", payload.VolunteerID))
    })

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

    eventBus.Subscribe(events.TaskStatusChanged, func(event events.Event) {
        payload := event.Payload.(events.TaskStatusChangedPayload)
        if payload.NewStatus == "completed" {
            ctx := context.Background()
            ledgerSvc.RecordValidatedTransaction(ctx, payload.TaskSlug, "TASK_STATUS_COMPLETED", 0, payload.TaskSlug)
        }
    })

    eventBus.Subscribe(events.FinancialStateChanged, func(event events.Event) {
        payload := event.Payload.(events.FinancialStateChangedPayload)
        ctx := context.Background()
        if payload.NewState == "LIQUIDATING" || payload.NewState == "READY_FOR_PAYOUT" {
            escrowSvc.FinalizeAndBroadcastPayout(ctx, payload.TaskSlug)
        }
    })

    router := gin.Default()
    router.Use(middleware.ErrorHandler(), middleware.RateLimiter(), middleware.ValidationMiddleware())
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
                tasks.GET(":slug", taskH.GetTask)
                tasks.POST(":slug/apply", volunteerH.ApplyForTask)
                tasks.POST(":slug/submissions", volunteerH.SubmitWork)
                tasks.POST(":slug/trustees", trusteeH.RegisterTrusteeKeys)
                tasks.POST(":slug/donate", lightningH.RequestDonationInvoice)
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

    return router
}

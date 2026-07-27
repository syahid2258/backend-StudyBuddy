package routes

import (
	"github.com/goravel/framework/contracts/http"
	routecontract "github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/support"

	"goravel/app/facades"
	"goravel/app/http/controllers"
	"goravel/app/http/middleware"
)

func Web() {
	// ─────────────────────────────────────────────
	// PUBLIC ROUTES (No Authentication Required)
	// ─────────────────────────────────────────────

	// Welcome / Landing Page
	facades.Route().Get("/", func(ctx http.Context) http.Response {
		return ctx.Response().View().Make("welcome.tmpl", map[string]any{
			"version": support.Version,
		})
	})

	// Login Page
	facades.Route().Get("/login", func(ctx http.Context) http.Response {
		return ctx.Response().View().Make("login.tmpl", map[string]any{})
	})

	// Serve static files
	facades.Route().Static("public", "./public")

	// ─────────────────────────────────────────────
	// GLOBAL SECURITY HEADERS (FINDING-003 FIX)
	// Applied to all routes via middleware.
	// ─────────────────────────────────────────────
	securityHeadersMiddleware := middleware.NewSecurityHeaders()
	facades.Route().GlobalMiddleware(securityHeadersMiddleware)

	// ─────────────────────────────────────────────
	// PUBLIC API ROUTES (Auth Endpoints)
	// Rate-limited to prevent brute force (FINDING-007 FIX)
	// ─────────────────────────────────────────────
	authController := controllers.NewAuthController()

	authorRateLimiter := middleware.AuthLimiter
	facades.Route().Middleware(authorRateLimiter).Post("/api/login", authController.Login)
	facades.Route().Middleware(authorRateLimiter).Post("/api/register", authController.Register)

	// ─────────────────────────────────────────────
	// PROTECTED ROUTES (Authentication Required)
	// All routes in this group require a valid JWT cookie.
	// ─────────────────────────────────────────────
	authMiddleware := middleware.NewAuth()

	facades.Route().Middleware(authMiddleware).Group(func(router routecontract.Router) {
		homeController := controllers.NewHomeController()
		pageController := controllers.NewPageController()
		topicController := controllers.NewTopicController()
		profileController := controllers.NewProfileController()
		aiController := controllers.NewAIController()
		diagnosticAIController := controllers.NewAiController()
		moduleController := controllers.NewModuleController()
		flashcardController := controllers.NewFlashcardController()
		examController := controllers.NewExamController()
		adminController := controllers.NewAdminController()
		wsController := controllers.NewWebSocketController()
		activeRecallController := controllers.NewActiveRecallController()
		pomodoroController := controllers.NewPomodoroController()

		// ── Page Routes ──
		router.Get("/home", homeController.Index)
		router.Get("/addTopic", pageController.AddTopic)
		router.Get("/generate-topic", pageController.GenerateTopic)
		router.Get("/materi", pageController.Materi)
		router.Get("/modules", pageController.Modules)
		router.Get("/active-recall", pageController.ActiveRecall)
		router.Get("/exam", pageController.Exam)
		router.Get("/tanya-ai", aiController.Show)
		router.Get("/tanyaAI", aiController.Show) // backward compat alias
		router.Get("/profile", profileController.Show)
		router.Get("/pomodoro", pageController.Pomodoro)
		router.Get("/flashcard", pageController.Flashcard)
		

		// ── Logout ──
		router.Post("/api/logout", authController.Logout)

		// ── API: Topics / Projects ──
		router.Get("/api/topics", topicController.Index)
		router.Post("/api/topics", topicController.Store)
		router.Put("/api/topics/{id}", topicController.Update)
		router.Delete("/api/topics/{id}", topicController.Destroy)
		router.Post("/api/topics/generate", topicController.GenerateSyllabus)
		router.Get("/api/topics/{id}/modules", topicController.Modules)
		
		// "?"? API: Pomodoro "?"?
		router.Post("/api/pomodoro/log", pomodoroController.LogSession)

		// ── API: Modules / AI / Flashcards / Exams ──
		router.Get("/api/modules/{id}/content", moduleController.Content)
		router.Post("/api/modules/{id}/evaluate", moduleController.Evaluate)
		router.Post("/api/active-recall/{id}/submit", activeRecallController.Submit)
		router.Get("/api/modules/{id}/chat", moduleController.ChatHistory)
		router.Post("/api/modules/{id}/chat", moduleController.Chat)
		router.Post("/api/modules/{id}/complete", moduleController.Complete)
		router.Post("/api/modules/{id}/regenerate", moduleController.Regenerate)
		router.Get("/api/flashcards/due", flashcardController.Due)
		router.Post("/api/flashcards/{id}/review", flashcardController.Review)
		router.Post("/api/topics/{id}/flashcards/generate", flashcardController.Generate)
		router.Delete("/api/topics/{id}/flashcards", flashcardController.Delete)
		router.Get("/api/topics/{id}/exams", examController.GetExams)
		router.Get("/api/exams/{id}", examController.GetExam)
		router.Post("/api/topics/{id}/exam/generate", examController.GenerateExam)
		router.Post("/api/exams/{id}/submit", examController.Submit)
		router.Get("/api/ai/pool-status", diagnosticAIController.PoolStatus)
		router.Get("/api/ai/chat", aiController.ChatHistory)
		router.Post("/api/ai/chat", aiController.Chat)

		// ── API: User Profile ──
		router.Get("/api/profile/me", profileController.GetProfile)
		router.Put("/api/profile/me", profileController.UpdateProfile)

		apiKeyController := controllers.NewApiKeyController()
		
		// ── Admin Routes (Protected by Admin Middleware) ──
		adminMiddleware := middleware.Admin()
		router.Middleware(adminMiddleware).Group(func(adminRouter routecontract.Router) {
			// NOTE: Server-side admin simulation pages (/admin/chat, /admin/exam, etc.)
			// have been REMOVED — they are superseded by the React frontend (AdminAIMonitor)
			// and exposed dangerous AI interfaces accessible to any admin credential holder.
			adminRouter.Get("/admin/dashboard", adminController.Dashboard)
			adminRouter.Post("/api/admin/requests/{id}/claim", adminController.ClaimRequest)
			adminRouter.Post("/api/admin/send-chat", adminController.RespondRequest)
			adminRouter.Post("/api/admin/evaluate-active-recall", adminController.EvaluateActiveRecall)
			adminRouter.Get("/api/admin/users", adminController.GetUsers)
			adminRouter.Post("/api/admin/users", adminController.CreateUser)
			adminRouter.Get("/api/admin/logs", adminController.GetLogs)

			// API Key Management Routes
			adminRouter.Get("/api/admin/api-keys", apiKeyController.Index)
			adminRouter.Post("/api/admin/api-keys", apiKeyController.Store)
			adminRouter.Put("/api/admin/api-keys/{id}", apiKeyController.Update)
			adminRouter.Delete("/api/admin/api-keys/{id}", apiKeyController.Destroy)
		})

		// ── WebSocket Endpoint — requires authentication (FINDING-002 FIX) ──
		router.Get("/ws", wsController.HandleWS)
	})
}


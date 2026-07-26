package controllers

import (
	"encoding/json"
	"strconv"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

// PageController handles rendering of protected static/dynamic pages.
type PageController struct{}

func NewPageController() *PageController {
	return &PageController{}
}

// AddTopic renders the add topic page.
func (c *PageController) AddTopic(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}
	return ctx.Response().View().Make("addTopic.tmpl", map[string]any{
		"user": user,
	})
}

// GenerateTopic renders the topic generation page.
func (c *PageController) GenerateTopic(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	reqIDStr := ctx.Request().Query("req_id")
	var reqID int
	if reqIDStr != "" {
		reqID, _ = strconv.Atoi(reqIDStr)
	}

	var latestProject models.Project
	var modules []models.Module
	var latestReq models.AdminRequest

	if userID != nil {
		if reqID > 0 {
			facades.Orm().Query().Where("user_id", userID).Where("id", reqID).First(&latestReq)
		} else {
			// Fallback: Ambil status request terakhir untuk generate topic
			facades.Orm().Query().Where("user_id", userID).Where("type", "generate_topic").OrderBy("created_at", "desc").First(&latestReq)
		}

		if latestReq.ID != 0 {
			// Parse payload to get project title
			var reqPayload map[string]any
			json.Unmarshal([]byte(latestReq.Payload), &reqPayload)
			projectTitle, ok := reqPayload["title"].(string)
			
			if ok && projectTitle != "" {
				// Cari project berdasarkan title dari payload
				facades.Orm().Query().Where("user_id", userID).Where("title", projectTitle).First(&latestProject)
				
				if latestProject.ID != 0 {
					facades.Orm().Query().Where("project_id", latestProject.ID).OrderBy("modules.order", "asc").Get(&modules)
				}
			}
		}
	}

	return ctx.Response().View().Make("generate-topic.tmpl", map[string]any{
		"user":          user,
		"project":       latestProject,
		"modules":       modules,
		"requestStatus": latestReq.Status, // "pending", "processing", "completed"
		"reqID":         latestReq.ID,
	})
}

// Materi renders the study material page.
// Optionally loads a specific project if project_id is provided.
func (c *PageController) Materi(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	// Optional: load project context
	var project models.Project
	projectIDStr := ctx.Request().Query("project_id")
	if projectIDStr != "" {
		if projectID, err := strconv.Atoi(projectIDStr); err == nil {
			facades.Orm().Query().Find(&project, projectID)
		}
	} else if userID != nil {
		// Fallback to the user's latest project
		facades.Orm().Query().Where("user_id", userID).OrderBy("created_at", "desc").First(&project)
	}

	// Load modules for the project
	var modules []models.Module
	if project.ID != 0 {
		facades.Orm().Query().Where("project_id", project.ID).OrderBy("modules.order", "asc").Get(&modules)
	}

	// Determine active module based on module_id query param
	var activeModule models.Module
	moduleIDStr := ctx.Request().Query("module_id")
	if moduleIDStr != "" {
		if moduleID, err := strconv.Atoi(moduleIDStr); err == nil {
			for _, m := range modules {
				if m.ID == uint(moduleID) {
					activeModule = m
					break
				}
			}
		}
	}
	// Fallback to first module if not found
	if activeModule.ID == 0 && len(modules) > 0 {
		activeModule = modules[0]
	}

	return ctx.Response().View().Make("materi.tmpl", map[string]any{
		"user":    user,
		"project": project,
		"modules": modules,
		"activeModule": activeModule,
	})
}

// Modules renders the learning path / modules page.
func (c *PageController) Modules(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	var project models.Project
	projectIDStr := ctx.Request().Query("project_id")
	if projectIDStr != "" {
		if projectID, err := strconv.Atoi(projectIDStr); err == nil {
			facades.Orm().Query().Find(&project, projectID)
		}
	} else if userID != nil {
		// Fallback to the user's latest project
		facades.Orm().Query().Where("user_id", userID).OrderBy("created_at", "desc").First(&project)
	}

	// Load modules for the project
	var modules []models.Module
	if project.ID != 0 {
		err := facades.Orm().Query().Where("project_id", project.ID).OrderBy("modules.order", "asc").Get(&modules)
		if err != nil {
			facades.Log().Errorf("Error fetching modules: %v", err)
		}
	}

	// Check if exam exists for this project
	var exam models.Exam
	isExamUnlocked := false
	if project.ID != 0 {
		facades.Orm().Query().Where("project_id", project.ID).First(&exam)
		if exam.ID != 0 {
			isExamUnlocked = true
		}
	}

	facades.Log().Infof("Project ID: %d, Modules count: %d", project.ID, len(modules))

	return ctx.Response().View().Make("modules.tmpl", map[string]any{
		"user":           user,
		"project":        project,
		"modules":        modules,
		"isExamUnlocked": isExamUnlocked,
	})
}

// ActiveRecall renders the active recall / Feynman page.
func (c *PageController) ActiveRecall(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	moduleIDStr := ctx.Request().Query("module_id")
	var moduleID int
	if moduleIDStr != "" {
		moduleID, _ = strconv.Atoi(moduleIDStr)
	}

	var activeRecall models.ActiveRecall
	var module models.Module
	var nextModule models.Module

	if moduleID == 0 {
		// Fallback: If no module_id is provided, try to find the user's latest project and its first module
		var project models.Project
		if userID != nil {
			facades.Orm().Query().Where("user_id", userID).OrderBy("created_at", "desc").First(&project)
			if project.ID != 0 {
				facades.Orm().Query().Where("project_id", project.ID).OrderBy("modules.order", "asc").First(&module)
				moduleID = int(module.ID)
			}
		}
	}

	if moduleID != 0 {
		facades.Orm().Query().Find(&module, moduleID)
		facades.Orm().Query().Where("module_id", moduleID).First(&activeRecall)
		
		if activeRecall.ID == 0 {
			activeRecall = models.ActiveRecall{
				ModuleID: uint(moduleID),
				Question: "Jelaskan dengan bahasamu sendiri mengenai konsep utama pada modul ini.",
			}
		}

		facades.Orm().Query().Where("project_id", module.ProjectID).Where("modules.order", module.Order+1).First(&nextModule)
	}

	return ctx.Response().View().Make("active-recall.tmpl", map[string]any{
		"user": user,
		"module": module,
		"activeRecall": activeRecall,
		"nextModule": nextModule,
	})
}

// Exam renders the exam page.
func (c *PageController) Exam(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	projectIDStr := ctx.Request().Query("project_id")
	var projectID int
	if projectIDStr != "" {
		projectID, _ = strconv.Atoi(projectIDStr)
	}

	var project models.Project
	if projectID == 0 {
		if userID != nil {
			facades.Orm().Query().Where("user_id", userID).OrderBy("created_at", "desc").First(&project)
			projectID = int(project.ID)
		}
	} else {
		facades.Orm().Query().Find(&project, projectID)
	}

	var exam models.Exam
	if projectID != 0 {
		facades.Orm().Query().Where("project_id", projectID).First(&exam)
	}

	// Unmarshal and remove correct_answer
	var safeQuestions []map[string]any
	if exam.ID != 0 && exam.Questions != "" {
		var rawQuestions []map[string]any
		json.Unmarshal([]byte(exam.Questions), &rawQuestions)
		for _, q := range rawQuestions {
			delete(q, "correct_answer")
			safeQuestions = append(safeQuestions, q)
		}
	} else {
		safeQuestions = []map[string]any{}
	}

	questionsJSON, _ := json.Marshal(safeQuestions)

	return ctx.Response().View().Make("exam.tmpl", map[string]any{
		"user":          user,
		"project":       project,
		"exam":          exam,
		"questionsJSON": string(questionsJSON),
	})
}

// Pomodoro renders the Pomodoro timer page.
func (c *PageController) Pomodoro(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}
	return ctx.Response().View().Make("pomodoro.tmpl", map[string]any{
		"user": user,
	})
}

// Flashcard renders the flashcard / spaced repetition page.
func (c *PageController) Flashcard(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")
	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	// Load flashcards for the user
	var flashcards []models.Flashcard
	if userID != nil {
		var projects []models.Project
		facades.Orm().Query().Where("user_id", userID).Get(&projects)
		if len(projects) > 0 {
			var projectIDs []any
			for _, p := range projects {
				projectIDs = append(projectIDs, p.ID)
			}
			facades.Orm().Query().WhereIn("project_id", projectIDs).Get(&flashcards)
		}
	}

	return ctx.Response().View().Make("flashcard.tmpl", map[string]any{
		"user":       user,
		"flashcards": flashcards,
	})
}

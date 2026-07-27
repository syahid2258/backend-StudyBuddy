package controllers

import (
	"encoding/json"
	stdhttp "net/http"
	"strconv"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

type AdminController struct {
}

func NewAdminController() *AdminController {
	return &AdminController{}
}

func (c *AdminController) Dashboard(ctx http.Context) http.Response {
	var requests []models.AdminRequest
	facades.Orm().Query().With("User").Where("status", "pending").OrderByDesc("created_at").Get(&requests)
	return ctx.Response().View().Make("admin-requests.tmpl", map[string]any{
		"requests": requests,
		"pendingCount": len(requests),
	})
}

func (c *AdminController) Chat(ctx http.Context) http.Response {
	reqIDStr := ctx.Request().Query("req_id")
	if reqIDStr == "" {
		return ctx.Response().String(400, "req_id is required")
	}
	
	var req models.AdminRequest
	if err := facades.Orm().Query().With("User").Find(&req, reqIDStr); err != nil {
		return ctx.Response().String(404, "Request not found")
	}

	var history []models.AdminRequest
	facades.Orm().Query().Where("user_id", req.UserID).Where("type", "tanya_ai").OrderBy("created_at").Get(&history)
	historyJSON, _ := json.Marshal(history)

	var payload map[string]any
	json.Unmarshal([]byte(req.Payload), &payload)

	userMessage, _ := payload["message"].(string)

	return ctx.Response().View().Make("admin-ai-chat.tmpl", map[string]any{
		"req": req,
		"userMessage": userMessage,
		"historyJSON": string(historyJSON),
	})
}

func (c *AdminController) Exam(ctx http.Context) http.Response {
	reqIDStr := ctx.Request().Query("req_id")
	var req models.AdminRequest
	
	if reqIDStr != "" {
		reqID, _ := strconv.Atoi(reqIDStr)
		facades.Orm().Query().With("User").Find(&req, reqID)
	}

	return ctx.Response().View().Make("admin-ai-exam.tmpl", map[string]any{
		"req": req,
	})
}

func (c *AdminController) Flashcard(ctx http.Context) http.Response {
	reqIDStr := ctx.Request().Query("req_id")
	var req models.AdminRequest
	
	if reqIDStr != "" {
		reqID, _ := strconv.Atoi(reqIDStr)
		facades.Orm().Query().With("User").Find(&req, reqID)
	}

	return ctx.Response().View().Make("admin-ai-flashcard.tmpl", map[string]any{
		"req": req,
	})
}

func (c *AdminController) ActiveRecall(ctx http.Context) http.Response {
	reqIDStr := ctx.Request().Query("req_id")
	var req models.AdminRequest
	
	if reqIDStr != "" {
		reqID, _ := strconv.Atoi(reqIDStr)
		facades.Orm().Query().With("User").Find(&req, reqID)
	}

	return ctx.Response().View().Make("admin-ai-active-recall.tmpl", map[string]any{
		"req": req,
	})
}

func (c *AdminController) AI(ctx http.Context) http.Response {
	return ctx.Response().View().Make("admin-ai.tmpl", map[string]any{})
}

func (c *AdminController) ClaimRequest(ctx http.Context) http.Response {
	id := ctx.Request().Route("id")
	// Find request by id, update status to 'processing', assign AdminID
	// In a real app, we get AdminID from Auth.
	var req models.AdminRequest
	if err := facades.Orm().Query().Where("id", id).FirstOrFail(&req); err != nil {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Request not found"})
	}

	req.Status = "processing"
	facades.Orm().Query().Save(&req)
	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Request claimed"})
}

func (c *AdminController) RespondRequest(ctx http.Context) http.Response {
	var payload struct {
		RequestID     uint   `json:"request_id"`
		Message       string `json:"message"`
		Type          string `json:"type"` // e.g. "chat_reply", "regenerate_subtopic", "generate_topic", "finish_generate_topic"
		ModuleTitle   string `json:"module_title"`
		ModuleContent string `json:"module_content"`
		ModuleType    string `json:"module_type"`
		ModuleID      int    `json:"module_id"`
	}

	if err := ctx.Request().Bind(&payload); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Invalid payload"})
	}

	// Fetch the AdminRequest
	var req models.AdminRequest
	if err := facades.Orm().Query().Where("id", payload.RequestID).FirstOrFail(&req); err != nil {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Request not found"})
	}

	var project models.Project

	// For generate_topic, we store to projects, modules, and active_recalls
	if payload.Type == "generate_topic" {
		var reqPayload map[string]any
		json.Unmarshal([]byte(req.Payload), &reqPayload)
		projectTitle, _ := reqPayload["title"].(string)

		// Find or create Project
		facades.Orm().Query().Where("user_id = ?", req.UserID).Where("title = ?", projectTitle).OrderByDesc("id").First(&project)
		if project.ID == 0 {
			project = models.Project{
				UserID: &req.UserID,
				Title:  projectTitle,
				Total:  0,
			}
			facades.Orm().Query().Create(&project)
		}

		if payload.ModuleType == "active_recall" {
			// Find the module by Order == payload.ModuleID
			var module models.Module
			facades.Orm().Query().Where("project_id", project.ID).Where("modules.order", payload.ModuleID).First(&module)
			
			if module.ID != 0 {
				activeRecall := models.ActiveRecall{
					ModuleID: module.ID,
					Question: payload.ModuleContent,
				}
				facades.Orm().Query().Create(&activeRecall)
			}
		} else if payload.ModuleType == "exam" {
			exam := models.Exam{
				ProjectID: project.ID,
				Questions: payload.ModuleContent,
			}
			facades.Orm().Query().Create(&exam)
		} else {
			// Update total for new module
			project.Total++
			facades.Orm().Query().Save(&project)

			// Create Module
			contentBlocks := models.ContentBlockList{
				{Type: "paragraph", Text: payload.ModuleContent},
			}
			module := models.Module{
				ProjectID:     project.ID,
				Title:         payload.ModuleTitle,
				Order:         project.Total, // which should match payload.ModuleID if sent in order
				IsLocked:      project.Total > 1,
				Status:        "not_started",
				ContentBlocks: &contentBlocks,
			}
			facades.Orm().Query().Create(&module)
		}
	} else if payload.Type == "finish_generate_topic" {
		// Also fetch project to get ID
		var reqPayload map[string]any
		json.Unmarshal([]byte(req.Payload), &reqPayload)
		projectTitle, _ := reqPayload["title"].(string)
		facades.Orm().Query().Where("user_id = ?", req.UserID).Where("title = ?", projectTitle).OrderByDesc("id").First(&project)
	} else if payload.Type == "regenerate_subtopic" {
		if req.SubTopicID != nil {
			if payload.ModuleType == "materi" {
				var module models.Module
				if err := facades.Orm().Query().Find(&module, *req.SubTopicID); err == nil && module.ID != 0 {
					module.Title = payload.ModuleTitle
					contentBlocks := models.ContentBlockList{
						{Type: "paragraph", Text: payload.ModuleContent},
					}
					module.ContentBlocks = &contentBlocks
					facades.Orm().Query().Save(&module)
				}
			} else if payload.ModuleType == "active_recall" {
				var activeRecall models.ActiveRecall
				facades.Orm().Query().Where("module_id", *req.SubTopicID).First(&activeRecall)
				if activeRecall.ID != 0 {
					activeRecall.Question = payload.ModuleContent
					facades.Orm().Query().Save(&activeRecall)
				} else {
					activeRecall = models.ActiveRecall{
						ModuleID: uint(*req.SubTopicID),
						Question: payload.ModuleContent,
					}
					facades.Orm().Query().Create(&activeRecall)
				}
			}
		}
	}
	var exam models.Exam
	if payload.Type == "generate_exam" {
		// Admin is returning questions for the exam.
		var reqPayload map[string]any
		json.Unmarshal([]byte(req.Payload), &reqPayload)
		projectIDFloat, _ := reqPayload["project_id"].(float64)
		projectID := uint(projectIDFloat)

		questionTypesRaw, _ := reqPayload["question_types"].([]interface{})
		var questionTypes []string
		for _, q := range questionTypesRaw {
			questionTypes = append(questionTypes, q.(string))
		}
		questionTypesJSON, _ := json.Marshal(questionTypes)

		exam = models.Exam{
			ProjectID:     projectID,
			Title:         payload.ModuleTitle,
			QuestionTypes: string(questionTypesJSON),
			Questions:     payload.ModuleContent,
		}
		facades.Orm().Query().Create(&exam)
	} else if payload.Type == "generate_flashcards" {
		var reqPayload map[string]any
		json.Unmarshal([]byte(req.Payload), &reqPayload)
		projectIDFloat, _ := reqPayload["project_id"].(float64)
		projectID := uint(projectIDFloat)

		// Hapus flashcard lama untuk project ini
		facades.Orm().Query().Where("project_id", projectID).Delete(&models.Flashcard{})

		// Parse flashcards dari Admin (dari payload.ModuleContent)
		var flashcardsData []map[string]string
		if err := json.Unmarshal([]byte(payload.ModuleContent), &flashcardsData); err == nil {
			for _, fcData := range flashcardsData {
				fc := models.Flashcard{
					ProjectID:    projectID,
					FrontText:    fcData["front_text"],
					BackText:     fcData["back_text"],
					EaseFactor:   2.5,
					IntervalDays: 0,
				}
				facades.Orm().Query().Create(&fc)
			}
		}
	}
	
	if payload.Type == "finish_generate_topic" || payload.Type == "chat_reply" || payload.Type == "regenerate_subtopic" || payload.Type == "tanya_ai" || payload.Type == "generate_exam" || payload.Type == "generate_flashcards" {
		if req.Response == "" {
			req.Status = "completed"
			req.Response = payload.Message
			facades.Orm().Query().Save(&req)
		} else {
			newReq := models.AdminRequest{
				UserID: req.UserID,
				Type: "tanya_ai",
				Payload: "{}",
				Response: payload.Message,
				Status: "completed",
			}
			facades.Orm().Query().Create(&newReq)
		}
	}

	// Build response based on Type
	var responseMap map[string]any = map[string]any{
		"type":          payload.Type,
		"message":       payload.Message,
		"module_title":  payload.ModuleTitle,
		"project_id":    project.ID,
		"project_title": project.Title,
		"req_id":        payload.RequestID,
	}

	if payload.Type == "generate_exam" {
		responseMap["exam_id"] = exam.ID
		var rawQuestions []map[string]any
		json.Unmarshal([]byte(payload.ModuleContent), &rawQuestions)
		
		// Remove correct_answer from broadcast to prevent cheating
		for i := range rawQuestions {
			delete(rawQuestions[i], "correct_answer")
		}
		responseMap["questions"] = rawQuestions
	}

	// Broadcast response to user via WebSocket
	responseBytes, _ := json.Marshal(responseMap)
	
	// Convert UserID and SubTopicID to string
	importStrConv := strconv.Itoa(int(req.UserID))
	subTopicStr := "global"
	if req.SubTopicID != nil {
		subTopicStr = strconv.Itoa(int(*req.SubTopicID))
	} else if payload.Type == "tanya_ai" || payload.Type == "generate_flashcards" {
		subTopicStr = "chat_user"
	}

	err := BroadcastMessage(importStrConv, subTopicStr, responseBytes)
	if err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to broadcast"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Response sent"})
}

// GetUsers handles GET /api/admin/users — returns all users with their project count.
func (c *AdminController) GetUsers(ctx http.Context) http.Response {
	var users []models.User
	if err := facades.Orm().Query().OrderByDesc("created_at").Get(&users); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, http.Json{
			"status":  "error",
			"message": "Gagal mengambil data user: " + err.Error(),
		})
	}

	// Build response with project counts
	type UserWithStats struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		CreatedAt   any    `json:"created_at"`
		ProjectCount int64 `json:"project_count"`
	}

	result := make([]UserWithStats, 0, len(users))
	for _, u := range users {
		var count int64
		count, _ = facades.Orm().Query().Model(&models.Project{}).Where("user_id", u.ID).Count()
		result = append(result, UserWithStats{
			ID:           u.ID,
			Name:         u.Name,
			Email:        u.Email,
			Role:         u.Role,
			CreatedAt:    u.CreatedAt,
			ProjectCount: count,
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, http.Json{
		"status": "success",
		"data":   result,
		"total":  len(result),
	})
}

// CreateUser handles POST /api/admin/users — creates a new user.
func (c *AdminController) CreateUser(ctx http.Context) http.Response {
	name := ctx.Request().Input("name")
	email := ctx.Request().Input("email")
	password := ctx.Request().Input("password")
	role := ctx.Request().Input("role")

	if name == "" || email == "" || password == "" {
		return ctx.Response().Json(stdhttp.StatusBadRequest, http.Json{
			"status":  "error",
			"message": "Nama, email, dan password wajib diisi.",
		})
	}

	if role == "" {
		role = "user"
	}

	var existingUser models.User
	if err := facades.Orm().Query().Where("email", email).First(&existingUser); err == nil && existingUser.ID > 0 {
		return ctx.Response().Json(stdhttp.StatusBadRequest, http.Json{
			"status":  "error",
			"message": "Email sudah digunakan.",
		})
	}

	hashedPassword, err := facades.Hash().Make(password)
	if err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, http.Json{
			"status":  "error",
			"message": "Gagal mengenkripsi password.",
		})
	}

	newUser := models.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     role,
	}

	if err := facades.Orm().Query().Create(&newUser); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, http.Json{
			"status":  "error",
			"message": "Gagal membuat user: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusCreated, http.Json{
		"status":  "success",
		"message": "User berhasil dibuat",
	})
}

// GetLogs handles GET /api/admin/logs — returns recent AI activity logs.
func (c *AdminController) GetLogs(ctx http.Context) http.Response {
	var requests []models.AdminRequest
	if err := facades.Orm().Query().With("User").OrderByDesc("created_at").Limit(50).Get(&requests); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, http.Json{
			"status":  "error",
			"message": "Gagal mengambil log aktivitas: " + err.Error(),
		})
	}

	type LogEntry struct {
		ID        uint   `json:"id"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		UserEmail string `json:"user_email"`
		CreatedAt any    `json:"created_at"`
	}

	logs := make([]LogEntry, 0, len(requests))
	for _, r := range requests {
		userEmail := ""
		if r.User != nil {
			userEmail = r.User.Email
		}
		logs = append(logs, LogEntry{
			ID:        r.ID,
			Type:      r.Type,
			Status:    r.Status,
			UserEmail: userEmail,
			CreatedAt: r.CreatedAt,
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, http.Json{
		"status": "success",
		"data":   logs,
	})
}

func (c *AdminController) EvaluateActiveRecall(ctx http.Context) http.Response {
	var payload struct {
		RequestID      uint                  `json:"request_id"`
		ActiveRecallID uint                  `json:"active_recall_id"`
		Score          int                   `json:"score"`
		Feedback       string                `json:"feedback"`
		Evaluations    models.EvaluationList `json:"evaluations"`
	}

	if err := ctx.Request().Bind(&payload); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Invalid payload"})
	}

	// Fetch AdminRequest
	var req models.AdminRequest
	if err := facades.Orm().Query().Where("id", payload.RequestID).FirstOrFail(&req); err != nil {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Request not found"})
	}

	// Update ActiveRecall
	var activeRecall models.ActiveRecall
	if err := facades.Orm().Query().Where("id", payload.ActiveRecallID).First(&activeRecall); err == nil && activeRecall.ID != 0 {
		activeRecall.Score = &payload.Score
		activeRecall.Feedback = payload.Feedback
		activeRecall.Evaluations = &payload.Evaluations
		facades.Orm().Query().Save(&activeRecall)

		// Unlocking logic if passed
		if payload.Score >= 80 {
			var module models.Module
			if err := facades.Orm().Query().Find(&module, activeRecall.ModuleID); err == nil && module.ID != 0 {
				module.Status = "mastered"
				facades.Orm().Query().Save(&module)

				var nextModule models.Module
				if err := facades.Orm().Query().Where("project_id", module.ProjectID).Where("`order`", module.Order+1).First(&nextModule); err == nil && nextModule.ID != 0 {
					nextModule.IsLocked = false
					nextModule.Status = "not_started"
					facades.Orm().Query().Save(&nextModule)
				}

				var project models.Project
				if err := facades.Orm().Query().Find(&project, module.ProjectID); err == nil && project.ID != 0 {
					project.Completed++
					facades.Orm().Query().Save(&project)
				}
			}
		}
	}

	// Update Request status
	req.Status = "completed"
	req.Response = "Evaluated"
	facades.Orm().Query().Save(&req)

	// Broadcast
	responseBytes, _ := json.Marshal(map[string]any{
		"type":             "active_recall_evaluated",
		"active_recall_id": payload.ActiveRecallID,
		"score":            payload.Score,
		"feedback":         payload.Feedback,
		"evaluations":      payload.Evaluations,
		"req_id":           payload.RequestID,
	})

	importStrConv := strconv.Itoa(int(req.UserID))
	subTopicStr := "global"
	if req.SubTopicID != nil {
		subTopicStr = strconv.Itoa(int(*req.SubTopicID))
	}

	BroadcastMessage(importStrConv, subTopicStr, responseBytes)

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Active recall evaluated successfully"})
}

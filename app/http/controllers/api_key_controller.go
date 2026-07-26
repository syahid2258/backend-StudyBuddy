package controllers

import (
	"goravel/app/models"
	"goravel/app/services/ai"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

type ApiKeyController struct {
}

func NewApiKeyController() *ApiKeyController {
	return &ApiKeyController{}
}

// Index fetches all API keys combined with their real-time usage stats.
func (c *ApiKeyController) Index(ctx http.Context) http.Response {
	var keys []models.ApiKey
	if err := facades.Orm().Query().Order("id desc").Get(&keys); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal mengambil data API Keys",
		})
	}

	// Fetch real-time pool status
	poolStatus, _ := ai.GetPoolStatus(ctx.Context())
	statusMap := make(map[uint]ai.PoolStatusEntry)
	for _, entry := range poolStatus {
		statusMap[entry.ID] = entry
	}

	type KeyResponse struct {
		models.ApiKey
		UsageRPM  int    `json:"usage_rpm"`
		UsageTPM  int    `json:"usage_tpm"`
		UsageRPD  int    `json:"usage_rpd"`
		IsCooling    bool   `json:"is_cooling"`
		StatusStr    string `json:"status_str"` 
		TotalReqs    int    `json:"total_requests"`
		TotalTks     int    `json:"total_tokens"`
		IsCurrentKey bool   `json:"is_current_key"`
	}

	var response []KeyResponse
	for _, k := range keys {
		resp := KeyResponse{ApiKey: k}
		
		if !k.IsActive {
			resp.StatusStr = "Tidak Aktif"
		} else if !k.IsValid {
			resp.StatusStr = "Invalid"
		} else if stat, exists := statusMap[k.ID]; exists {
			resp.UsageRPM = stat.UsageRPM
			resp.UsageTPM = stat.UsageTPM
			resp.UsageRPD = stat.UsageRPD
			resp.TotalReqs = stat.TotalReqs
			resp.TotalTks = stat.TotalTks
			resp.IsCurrentKey = stat.IsCurrent
			if len(stat.Cooldowns) > 0 {
				resp.IsCooling = true
				resp.StatusStr = "Rate Limited"
			} else {
				resp.StatusStr = "Aktif"
			}
		} else {
			resp.StatusStr = "Aktif"
		}

		// Also mask the key if not fully masked
		resp.Key = maskAPIKeyStr(k.Key)
		response = append(response, resp)
	}

	return ctx.Response().Json(http.StatusOK, response)
}

// Store adds a new API Key.
func (c *ApiKeyController) Store(ctx http.Context) http.Response {
	name := ctx.Request().Input("name")
	keyVal := ctx.Request().Input("key")

	if name == "" || keyVal == "" {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Name and Key are required"})
	}

	newKey := models.ApiKey{
		Name:     name,
		Key:      keyVal,
		IsActive: true,
		IsValid:  true,
	}

	if err := facades.Orm().Query().Create(&newKey); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan API Key (mungkin duplikat)"})
	}

	// Reload the pool
	ai.ReloadKeys(ctx.Context())

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "API Key berhasil ditambahkan"})
}

// Update modifies an API Key (e.g., toggle active/inactive).
func (c *ApiKeyController) Update(ctx http.Context) http.Response {
	id := ctx.Request().Route("id")
	isActive := ctx.Request().InputBool("is_active")

	var key models.ApiKey
	if err := facades.Orm().Query().Where("id", id).First(&key); err != nil || key.ID == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "API Key tidak ditemukan"})
	}

	key.IsActive = isActive
	// If re-activated, we assume it's valid again so we can retry
	if isActive {
		key.IsValid = true 
	}
	
	if err := facades.Orm().Query().Save(&key); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal mengupdate API Key"})
	}

	// Reload the pool
	ai.ReloadKeys(ctx.Context())

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "API Key berhasil diupdate"})
}

// Destroy removes an API Key.
func (c *ApiKeyController) Destroy(ctx http.Context) http.Response {
	id := ctx.Request().Route("id")

	if _, err := facades.Orm().Query().Where("id", id).Delete(&models.ApiKey{}); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menghapus API Key"})
	}

	// Reload the pool
	ai.ReloadKeys(ctx.Context())

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "API Key berhasil dihapus"})
}

func maskAPIKeyStr(key string) string {
	if len(key) <= 10 {
		return "***"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"

	"goravel/app/facades"
	appmodels "goravel/app/models"

	"github.com/goravel/framework/database/orm"
)

// keyState menyimpan kondisi satu API key: client yang sudah diinisialisasi,
// dan sampai kapan key ini "didinginkan" (cooldown) untuk model tertentu
// setelah baru saja kena error retryable (rate limit, server error, dsb).
type keyState struct {
	id        uint
	name      string
	apiKey    string
	client    *genai.Client
	cooldowns map[string]time.Time

	// Usage tracking (In-Memory)
	usageRPM int
	usageTPM int
	usageRPD int

	totalRequests int
	totalTokens   int

	rpmResetAt time.Time
	rpdResetAt time.Time
}

// keyPool mengelola rotasi otomatis antar beberapa API key Gemini
type keyPool struct {
	mu      sync.Mutex
	keys    []*keyState
	nextIdx int
	loaded  bool
}

var (
	pool    = &keyPool{}
	poolErr error
)

const (
	cooldownDuration = 60 * time.Second
	limitRPM         = 5       // Disesuaikan dengan batas Gemini Flash (5 RPM)
	limitTPM         = 250000  // Disesuaikan dengan batas Gemini Flash (250K TPM)
	limitRPD         = 20      // Disesuaikan dengan batas Gemini Flash (20 RPD)
)

func getPool(ctx context.Context) (*keyPool, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.loaded {
		return pool, poolErr
	}
	return reloadKeysInternal(ctx)
}

// ReloadKeys reloads the keys from the database. Called by Admin controller when keys are updated.
func ReloadKeys(ctx context.Context) (*keyPool, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return reloadKeysInternal(ctx)
}

func reloadKeysInternal(ctx context.Context) (*keyPool, error) {
	var apiKeys []appmodels.ApiKey
	// Load active and valid keys
	err := facades.Orm().Query().Where("is_active = ?", true).Where("is_valid = ?", true).Get(&apiKeys)
	if err != nil {
		poolErr = fmt.Errorf("gagal memuat API keys dari database: %w", err)
		pool.loaded = true
		return pool, poolErr
	}

	var rawKeys []string
	// Fallback to .env if DB is empty
	if len(apiKeys) == 0 {
		multiRaw := facades.Config().GetString("ai.gemini_api_keys")
		if strings.TrimSpace(multiRaw) != "" {
			for _, k := range strings.Split(multiRaw, ",") {
				k = strings.TrimSpace(k)
				if k != "" {
					rawKeys = append(rawKeys, k)
				}
			}
		}

		if len(rawKeys) == 0 {
			single := facades.Config().GetString("ai.gemini_api_key")
			if strings.TrimSpace(single) != "" {
				rawKeys = []string{single}
			}
		}

		// Convert rawKeys to dummy ApiKey objects
		for i, k := range rawKeys {
			apiKeys = append(apiKeys, appmodels.ApiKey{
				Model:    orm.Model{ID: uint(i + 1)},
				Name:     fmt.Sprintf("Env Key %d", i+1),
				Key:      k,
				IsActive: true,
				IsValid:  true,
			})
		}
	}

	if len(apiKeys) == 0 {
		poolErr = fmt.Errorf("tidak ada API key Gemini yang aktif dan valid")
		pool.loaded = true
		return pool, poolErr
	}

	// Keep old usage if possible
	oldKeys := make(map[uint]*keyState)
	for _, ks := range pool.keys {
		oldKeys[ks.id] = ks
	}

	var newKeys []*keyState
	now := time.Now()
	for _, ak := range apiKeys {
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  ak.Key,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			facades.Log().Errorf("gagal inisialisasi client untuk API key ID %d: %v", ak.ID, err)
			continue
		}
		
		ks := &keyState{
			id:         ak.ID,
			name:       ak.Name,
			apiKey:     ak.Key,
			client:     client,
			cooldowns:  make(map[string]time.Time),
			rpmResetAt: now.Add(time.Minute),
			rpdResetAt: now.Add(24 * time.Hour),
			totalRequests: ak.TotalRequests,
			totalTokens:   ak.TotalTokens,
		}

		// Migrate old usage if it existed
		if old, exists := oldKeys[ak.ID]; exists {
			ks.usageRPM = old.usageRPM
			ks.usageTPM = old.usageTPM
			ks.usageRPD = old.usageRPD
			ks.rpmResetAt = old.rpmResetAt
			ks.rpdResetAt = old.rpdResetAt
			ks.cooldowns = old.cooldowns
		}

		newKeys = append(newKeys, ks)
	}

	if len(newKeys) == 0 {
		poolErr = fmt.Errorf("semua API key gagal diinisialisasi")
	} else {
		poolErr = nil
	}

	pool.keys = newKeys
	if len(newKeys) > 0 && pool.nextIdx < len(newKeys) {
		// Pertahankan giliran nextIdx saat reload agar tidak kembali ke Key 1 (0) saat tidak ada req / reload
	} else {
		pool.nextIdx = 0
	}
	pool.loaded = true

	return pool, poolErr
}

// markKeyInvalid marks a key as invalid in DB after 401/403
func markKeyInvalid(ctx context.Context, id uint) {
	var key appmodels.ApiKey
	if err := facades.Orm().Query().Where("id", id).First(&key); err == nil && key.ID != 0 {
		key.IsValid = false
		facades.Orm().Query().Save(&key)
	}
}

// isRetryableAPIError menentukan apakah error dari Gemini API layak untuk
// mencoba kombinasi key/model lain
func isRetryableAPIError(err error) bool {
	var apiErr *genai.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Code == 429: // rate limit / quota habis untuk key+model ini
			return true
		case apiErr.Code == 401 || apiErr.Code == 403: // key tidak valid/dicabut/tidak punya akses
			return true
		case apiErr.Code >= 500: // error di sisi server Google, coba key/model lain
			return true
		default:
			return false
		}
	}
	return true
}

func (ks *keyState) checkAndResetUsage() {
	now := time.Now()
	if now.After(ks.rpmResetAt) {
		ks.usageRPM = 0
		ks.usageTPM = 0
		ks.rpmResetAt = now.Add(time.Minute)
	}
	if now.After(ks.rpdResetAt) {
		ks.usageRPD = 0
		ks.rpdResetAt = now.Add(24 * time.Hour)
	}
}

// generateContentWithFailover memanggil GenerateContent lewat key pool,
// otomatis rotate ke API key lain atau model fallback lain kalau kombinasi
// yang sedang dicoba kena error retryable.
func generateContentWithFailover(ctx context.Context, models []string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	p, err := getPool(ctx)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("tidak ada model yang diberikan")
	}

	p.mu.Lock()
	totalKeys := len(p.keys)
	startIdx := p.nextIdx
	p.mu.Unlock()

	var lastErr error
	var attemptsMade int

	for _, model := range models {
		for attempt := 0; attempt < totalKeys; attempt++ {
			p.mu.Lock()
			idx := (startIdx + attempt) % totalKeys
			ks := p.keys[idx]
			ks.checkAndResetUsage()
			
			// Custom soft limit check (optional, but good to avoid hitting hard 429)
			skip := ks.cooldowns[model].After(time.Now()) || ks.usageRPM >= limitRPM || ks.usageRPD >= limitRPD || ks.usageTPM >= limitTPM
			p.mu.Unlock()

			if skip {
				continue
			}
			attemptsMade++

			// Track request sent
			p.mu.Lock()
			ks.usageRPM++
			ks.usageRPD++
			ks.totalRequests++
			p.mu.Unlock()

			result, err := ks.client.Models.GenerateContent(ctx, model, contents, config)
			
			var addedTokens int
			if result != nil && result.UsageMetadata != nil {
				addedTokens = int(result.UsageMetadata.TotalTokenCount)
				p.mu.Lock()
				ks.usageTPM += addedTokens
				ks.totalTokens += addedTokens
				p.mu.Unlock()
			}

			// Background DB update
			go func(keyID uint, reqs int, tokens int) {
				var key appmodels.ApiKey
				if err := facades.Orm().Query().Where("id", keyID).First(&key); err == nil && key.ID != 0 {
					key.TotalRequests += reqs
					key.TotalTokens += tokens
					facades.Orm().Query().Save(&key)
				}
			}(ks.id, 1, addedTokens)

			if err == nil {
				p.mu.Lock()
				p.nextIdx = (idx + 1) % totalKeys // Lanjut ke giliran API Key berikutnya secara merata dan berurutan (Round-Robin)
				p.mu.Unlock()
				return result, nil
			}

			lastErr = err
			
			// Identify if it's 401/403 to mark as invalid globally
			var apiErr *genai.APIError
			if errors.As(err, &apiErr) && (apiErr.Code == 401 || apiErr.Code == 403) {
				markKeyInvalid(ctx, ks.id)
			}

			if !isRetryableAPIError(err) {
				return nil, err
			}

			p.mu.Lock()
			ks.cooldowns[model] = time.Now().Add(cooldownDuration)
			p.mu.Unlock()
		}
	}

	if attemptsMade == 0 {
		return nil, fmt.Errorf("semua %d API key sedang cooldown atau over limit, coba lagi dalam beberapa saat", totalKeys)
	}
	return nil, fmt.Errorf("semua API key dan model gagal (error terakhir: %w)", lastErr)
}

// PoolStatusEntry merepresentasikan status satu key
type PoolStatusEntry struct {
	ID         uint                 `json:"id"`
	Name       string               `json:"name"`
	KeyPreview string               `json:"key_preview"`
	Cooldowns  map[string]time.Time `json:"cooldowns,omitempty"`
	UsageRPM   int                  `json:"usage_rpm"`
	UsageTPM   int                  `json:"usage_tpm"`
	UsageRPD   int                  `json:"usage_rpd"`
	TotalReqs  int                  `json:"total_requests"`
	TotalTks   int                  `json:"total_tokens"`
	IsCurrent  bool                 `json:"is_current_key"`
}

// GetPoolStatus mengembalikan status seluruh key
func GetPoolStatus(ctx context.Context) ([]PoolStatusEntry, error) {
	p, err := getPool(ctx)
	if err != nil && p == nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	var entries []PoolStatusEntry
	if p.keys == nil {
		return entries, nil
	}
	for i, ks := range p.keys {
		ks.checkAndResetUsage()
		entry := PoolStatusEntry{
			ID:         ks.id,
			Name:       ks.name,
			KeyPreview: maskAPIKey(ks.apiKey),
			Cooldowns:  make(map[string]time.Time),
			UsageRPM:   ks.usageRPM,
			UsageTPM:   ks.usageTPM,
			UsageRPD:   ks.usageRPD,
			TotalReqs:  ks.totalRequests,
			TotalTks:   ks.totalTokens,
			IsCurrent:  (i == p.nextIdx),
		}
		for model, till := range ks.cooldowns {
			if till.After(now) {
				entry.Cooldowns[model] = till
			}
		}
		if len(entry.Cooldowns) == 0 {
			entry.Cooldowns = nil
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func maskAPIKey(key string) string {
	if len(key) <= 6 {
		return "***"
	}
	return "..." + key[len(key)-6:]
}

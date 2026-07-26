package models

import (
	"github.com/goravel/framework/database/orm"
)

type ApiKey struct {
	orm.Model
	Name     string `gorm:"column:name" json:"name"`
	Key      string `gorm:"column:key;unique" json:"key"`
	IsActive      bool   `gorm:"column:is_active;default:true" json:"is_active"`
	IsValid       bool   `gorm:"column:is_valid;default:true" json:"is_valid"`
	TotalRequests int    `gorm:"column:total_requests;default:0" json:"total_requests"`
	TotalTokens   int    `gorm:"column:total_tokens;default:0" json:"total_tokens"`
}

func (a *ApiKey) TableName() string {
	return "api_keys"
}

package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/goravel/framework/database/orm"
)

type ActiveRecall struct {
	orm.Model
	ModuleID    uint            `json:"module_id"`
	Question    string          `json:"question"`
	Answer      string          `json:"answer"`
	Score       *int            `json:"score"`
	Feedback    string          `json:"feedback"`
	Evaluations *EvaluationList `gorm:"type:json;serializer:json" json:"evaluations"`
}

type EvaluationItem struct {
	IsSuccess bool   `json:"is_success"`
	Text      string `json:"text"`
}

type EvaluationList []EvaluationItem

func (e *EvaluationList) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (e *EvaluationList) Scan(value any) error {
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, e)
}

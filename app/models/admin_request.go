package models

import (
	"github.com/goravel/framework/database/orm"
)

type AdminRequest struct {
	orm.Model
	UserID     uint   `json:"user_id"`
	User       *User  `json:"user"`
	SubTopicID *uint  `json:"sub_topic_id"`
	Type       string `json:"type"` // chat, module, evaluation, generate_topic
	Payload    string `json:"payload"`
	Status     string `json:"status"` // pending, processing, completed, failed
	AdminID    *uint  `json:"admin_id"`
	Response   string `json:"response"`
}

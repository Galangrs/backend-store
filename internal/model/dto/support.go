package dto

import (
	"time"

	"portolio-backend/configs/constants"
)

type RequestCreateTicket struct {
	Subject string `json:"subject" binding:"required,min=10,max=150"`
	Message string `json:"message" binding:"required,min=20,max=1000"`
}

type SupportTicketResponse struct {
	ID                uint                        `json:"id"`
	UserID            uint                        `json:"user_id"`
	Subject           string                      `json:"subject"`
	Status            constants.SupportTicketStatus `json:"status"`
	AssignedAdminID   *uint                       `json:"assigned_admin_id,omitempty"`
	AssignedAdminName string                      `json:"assigned_admin_name,omitempty"`
	QueueNumber       uint                        `json:"queue_number"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

type SupportMessageResponse struct {
	ID          uint                      `json:"id"`
	TicketID    uint                      `json:"ticket_id"`
	SenderID    uint                      `json:"sender_id"`
	SenderName  string                    `json:"sender_name"`
	IsAdmin     bool                      `json:"is_admin"`
	Content     string                    `json:"content"`
	MessageType constants.ChatMessageType `json:"message_type"`
	FileURL     string                    `json:"file_url,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
}

type RequestReplyTicket struct {
	Content     string                    `json:"content" binding:"required_without=FileURL,max=1000"`
	MessageType constants.ChatMessageType `json:"message_type" binding:"required,oneof=text image file"`
	FileURL string `json:"file_url,omitempty" binding:"omitempty,url,required_if=MessageType image required_if=MessageType file"`
}
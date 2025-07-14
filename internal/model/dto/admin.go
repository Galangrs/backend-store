package dto

import (
	"portolio-backend/configs/constants"
	"time"
)

type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}


type RequestSuspendUser struct {
	Reason string `json:"reason" binding:"required,min=10,max=255"`
}

type RequestBanUser struct {
	DurationHours int    `json:"duration_hours" binding:"required,gt=0,lte=8760"`
	Reason        string `json:"reason" binding:"required,min=10,max=255"`
}

type RequestPatchTransactionStatus struct {
	Status string `json:"status" binding:"required,oneof=pending waiting_owner waiting_users success cancel"`
	Reason string `json:"reason,omitempty"`
}

type UserDetailResponse struct {
	ID              uint                 `json:"id"`
	FullName        string               `json:"full_name"`
	Email           string               `json:"email"`
	Role            constants.UserRole   `json:"role"`
	Balance         uint                 `json:"balance"`
	Status          constants.UserStatus `json:"status"`
	BanUntil        *int64               `json:"ban_until,omitempty"`
	BanReason       string               `json:"ban_reason,omitempty"`
	PenaltyWarnings uint                 `json:"penalty_warnings"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	DeletedAt       *time.Time           `json:"deleted_at,omitempty"`
}

type GetUsersResponse struct {
	TotalRecords int64                `json:"total_records"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
	Users        []UserDetailResponse `json:"users"`
}

type GetProductsAdminResponse struct {
	TotalRecords int64                     `json:"total_records"`
	Page         int                       `json:"page"`
	Limit        int                       `json:"limit"`
	Products     []GetProductsRequestAdmin `json:"products"`
}

type TransactionDetailResponse struct {
	ID            uint                        `json:"id"`
	ProductID     uint                        `json:"product_id"`
	UserID        uint                        `json:"user_id"`
	Quantity      uint                        `json:"quantity"`
	TotalPrice    uint                        `json:"total_price"`
	GovtTax       uint                        `json:"govt_tax"`
	EcommerceTax  uint                        `json:"ecommerce_tax"`
	Status        constants.TransactionStatus `json:"status"`
	IsSolved      bool                        `json:"is_solved"`
	ReceiptStatus constants.ReceiptStatus     `json:"receipt_status"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
	Product       ProductDetailResponse       `json:"product,omitempty"` // Re-use ProductDetailResponse from shop DTO
	User          UserDetailResponse          `json:"user,omitempty"`    // Re-use UserDetailResponse
}

type GetTransactionsResponse struct {
	TotalRecords int64                       `json:"total_records"`
	Page         int                         `json:"page"`
	Limit        int                         `json:"limit"`
	Transactions []TransactionDetailResponse `json:"transactions"`
}

type BalanceHistoryDetailResponse struct {
	ID           uint                    `json:"id"`
	UserID       uint                    `json:"user_id"`
	Description  string                  `json:"description"`
	Amount       int                     `json:"amount"`
	LastBalance  uint                    `json:"last_balance"`
	FinalBalance uint                    `json:"final_balance"`
	Status       constants.BalanceStatus `json:"status"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	User         UserDetailResponse      `json:"user,omitempty"`
}

type GetBalanceHistoriesResponse struct {
	TotalRecords int64                          `json:"total_records"`
	Page         int                            `json:"page"`
	Limit        int                            `json:"limit"`
	Histories    []BalanceHistoryDetailResponse `json:"histories"`
}

type GetTopUpWithdrawLogsResponse struct {
	TotalRecords int64                          `json:"total_records"`
	Page         int                            `json:"page"`
	Limit        int                            `json:"limit"`
	Logs         []BalanceHistoryDetailResponse `json:"logs"`
}

type GetSupportTicketsResponse struct {
	TotalRecords int64                   `json:"total_records"`
	Page         int                     `json:"page"`
	Limit        int                     `json:"limit"`
	Tickets      []SupportTicketResponse `json:"tickets"`
}

type GetSupportTicketMessagesResponse struct {
	TicketID  uint                     `json:"ticket_id"`
	Subject   string                   `json:"subject"`
	Status    constants.SupportTicketStatus `json:"status"`
	Messages  []SupportMessageResponse `json:"messages"`
}
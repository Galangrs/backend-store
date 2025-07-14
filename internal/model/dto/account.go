package dto

import "time"

type RequestPostRegister struct {
	FullName string `json:"full_name" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"` // Validasi kompleks di handler
}

type RequestPostLogin struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RequestTopUp struct {
	Amount uint `json:"amount" binding:"required,gt=0,lte=10000000"` // Max 10 juta
}

type RequestWithdraw struct {
	Amount uint `json:"amount" binding:"required,gte=10000,lte=10000000"` 
}

type RequestPatchAccount struct {
	FullName    string `json:"full_name,omitempty" binding:"omitempty,min=3,max=100"`
	Email       string `json:"email,omitempty" binding:"omitempty,email"`
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty" binding:"omitempty,min=8"`
}

type PostRegisterResponse struct {
	Fullname  string    `json:"full_name"`
	Email     string    `json:"email"`
	Balance   uint      `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type BalanceHistoryResponse struct {
	ID           uint   `json:"id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Description  string `json:"description"`
	Amount       int    `json:"amount"`
	LastBalance  uint   `json:"last_balance"`
	FinalBalance uint   `json:"final_balance"`
	Status       string `json:"status"`
}

type GetBalanceResponse struct {
	Balance   uint                     `json:"balance"`
	Histories []BalanceHistoryResponse `json:"histories"`
}

type PostTopUpResponse struct {
	Message      string                 `json:"message"`
	FullName     string                 `json:"full_name"`
	NewBalance   uint                   `json:"new_balance"`
	BalanceEntry BalanceHistoryResponse `json:"balance_entry"`
}

type PostWithdrawResponse struct {
	Message      string                 `json:"message"`
	FullName     string                 `json:"full_name"`
	NewBalance   uint                   `json:"new_balance"`
	BalanceEntry BalanceHistoryResponse `json:"balance_entry"`
}
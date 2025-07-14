package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/internal/handler"
	"portolio-backend/internal/middlewares"
	"portolio-backend/internal/util"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB, hub *util.Hub) {
	accountHandler := handler.NewAccountHandler(db)
	adminHandler := handler.NewAdminHandler(db)
	shopHandler := handler.NewShopHandler(db)
	supportHandler := handler.NewSupportHandler(db)
	uploadHandler := handler.NewUploadHandler(db)
	websocketHandler := handler.NewWebsocketHandler(db, hub)
	mediaHandler := handler.NewMediaHandler(db) // Inisialisasi handler media baru

	api := r.Group("/api")
	{
		account := api.Group("/account")
		{
			account.POST("/register", accountHandler.PostRegisterRequest)
			account.POST("/login", accountHandler.PostLoginRequest)

			account.Use(middlewares.JWTMiddleware())
			account.Use(middlewares.Authorize(db))
			account.Use(middlewares.CheckUserStatus(db))

			account.GET("/balance", accountHandler.GetBalanceRequest)
			account.POST("/topup", accountHandler.PostTopUpBalance)
			account.POST("/withdraw", accountHandler.PostWithDrawBalance)
			account.PATCH("/", accountHandler.PatchAccount)
		}

		shop := api.Group("/shop")
		{
			shop.Use(middlewares.CheckRole())
			shop.GET("/products", shopHandler.GetProductsRequest)

			shop.Use(middlewares.JWTMiddleware())
			shop.Use(middlewares.Authorize(db))
			shop.Use(middlewares.CheckUserStatus(db))

			shop.GET("/orders", shopHandler.GetCartHandler)
			shop.POST("/purchase", shopHandler.PostPurchaseProduct)
			shop.POST("/transactions/cancel", shopHandler.CancelTransaction)
			shop.POST("/transactions/confirm-receipt", shopHandler.ConfirmTransactionByUser)

			shop.POST("/products", shopHandler.PostProductsRequest)
			shop.PUT("/products/:id", shopHandler.PutProductsRequest)
			shop.DELETE("/products/:id", shopHandler.DeleteProductsRequest)
			shop.PATCH("/products/:id/visibility", shopHandler.PatchProductVisibility)
			shop.GET("/products/orders", shopHandler.GetOwnerProductOrders)
			shop.POST("/products/orders/confirm-shipment", shopHandler.ConfirmTransactionByOwner)

			shop.POST("/support/tickets", supportHandler.CreateSupportTicket)
			shop.GET("/support/tickets", supportHandler.GetUserSupportTickets)
			shop.GET("/support/tickets/:id/messages", supportHandler.GetUserSupportTicketMessages)
			shop.POST("/support/tickets/:id/reply", supportHandler.ReplySupportTicket)
			shop.PATCH("/support/tickets/:id/cancel", supportHandler.CancelSupportTicket)
		}

		adminAPI := r.Group("/api/admin")
		{
			adminAPI.POST("/login", adminHandler.PostAdminLogin)

			adminAPI.Use(middlewares.JWTMiddleware())
			adminAPI.Use(middlewares.AdminAuth(db))
			adminAPI.Use(middlewares.AuthorizeAdmin(db))

			adminAPI.GET("/dashboard", func(c *gin.Context) {
				util.RespondJSON(c, 200, gin.H{"message": "Selamat datang di Dashboard Admin!"})
			})

			adminAPI.GET("/users", adminHandler.GetUsers)
			adminAPI.PATCH("/users/:id/suspend", adminHandler.SuspendUser)
			adminAPI.PATCH("/users/:id/ban", adminHandler.BanUser)
			adminAPI.PATCH("/users/:id/unban", adminHandler.UnbanUser)
			adminAPI.DELETE("/users/:id", adminHandler.DeleteUser)

			adminAPI.GET("/products", adminHandler.GetProductsAdmin)
			adminAPI.PATCH("/products/:id", adminHandler.PatchProductAdmin)

			adminAPI.GET("/transactions", adminHandler.GetTransactions)
			adminAPI.PATCH("/transactions/:id/status", adminHandler.PatchTransactionStatus)

			adminAPI.GET("/balances/history", adminHandler.GetBalanceHistories)
			adminAPI.GET("/balances/topup-withdraw-logs", adminHandler.GetTopUpWithdrawLogs)

			adminAPI.GET("/support/tickets", adminHandler.GetSupportTickets)
			adminAPI.GET("/support/tickets/:id/messages", adminHandler.GetSupportTicketMessages)
			adminAPI.POST("/support/tickets/:id/claim", adminHandler.ClaimSupportTicket)
			adminAPI.POST("/support/tickets/:id/reply", adminHandler.ReplySupportTicket)
		}

		uploadAPI := r.Group("/api/upload")
		{
			uploadAPI.Use(middlewares.JWTMiddleware())
			uploadAPI.Use(middlewares.Authorize(db))
			uploadAPI.Use(middlewares.CheckUserStatus(db))

			uploadAPI.POST("/image", uploadHandler.UploadImage)
			uploadAPI.POST("/file", uploadHandler.UploadFile)
		}
	}

	wsAPI := r.Group("/ws")
	{
		wsAPI.Use(middlewares.JWTMiddleware())
		wsAPI.Use(middlewares.Authorize(db))
		wsAPI.Use(middlewares.CheckUserStatus(db))

		wsAPI.GET("/chat/:transaction_id", websocketHandler.ChatHandler)
		wsAPI.GET("/notifications", websocketHandler.NotificationHandler)
	}

	// Rute media terproteksi (chat dan support)
	protectedMedia := r.Group("/media")
	{
		protectedMedia.Use(middlewares.JWTMiddleware())
		protectedMedia.Use(middlewares.Authorize(db))
		protectedMedia.Use(middlewares.CheckUserStatus(db))
		// Rute ini akan menangani /media/chat/:filename dan /media/support/:filename
		protectedMedia.GET("/:mediaType/:filename", mediaHandler.ServeProtectedMedia)
	}
}
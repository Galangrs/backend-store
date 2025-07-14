package main

import (
	"log"
	"fmt"
	"time"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"portolio-backend/configs"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/routes"
	"portolio-backend/internal/seed"
	"portolio-backend/internal/util"
)

func main() {
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"*"},
		AllowMethods: []string{"*"},
        ExposeHeaders:    []string{"Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	configDB := configs.GetDBConfig()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=disable password=%s client_encoding=UTF8",
		configDB.Host, configDB.Port, configDB.User, configDB.DBName, configDB.Password,
	)
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Fatalf("❌ Gagal terhubung ke database: %v", err)
	}

	err = dbConn.AutoMigrate(
		&db.User{},
		&db.Product{},
		&db.ProductImage{},
		&db.TransactionHistory{},
		&db.BalanceHistory{},
		&db.Review{},
		&db.Notification{},
		&db.ChatMessage{},
		&db.SupportTicket{},
		&db.SupportMessage{},
		&db.AdminSession{},
		&db.AdminLog{},
	)
	if err != nil {
		log.Fatalf("❌ Gagal melakukan auto migrate: %v", err)
	}

	mediaDirs := []string{"media/products", "media/chat", "media/support", "media/general", "media/temp"}
	for _, dir := range mediaDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatalf("❌ Gagal membuat direktori media '%s': %v", dir, err)
			}
		}
	}

	util.WebsocketHub = util.NewHub()
	go util.WebsocketHub.Run()

	seed.Shop(dbConn)

	routes.SetupRoutes(r, dbConn, util.WebsocketHub)

	// Hapus baris ini: r.Static("/media", "./media")

	// Tambahkan ini untuk akses publik ke gambar produk
	r.Static("api/media/products", "./media/products")

	port := configs.GetEnv("PORT", "8080")
	log.Printf("🚀 Server berjalan di port :%s", port)
	r.Run(":" + port)
}
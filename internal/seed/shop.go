package seed

import (
	"fmt"
	"log"
	"math/rand"

	"portolio-backend/internal/model/db"
	"portolio-backend/configs/constants"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Shop(dbConn *gorm.DB) {
	var count int64
	dbConn.Model(&db.User{}).Count(&count)
	if count > 0 {
		fmt.Println("🔁 Seeding skipped: data already exists.")
		return
	}

	// --- Hashing Passwords ---
	adminPass, err := bcrypt.GenerateFromPassword([]byte("@admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("❌ Failed to hash admin password:", err)
	}

	userPass, err := bcrypt.GenerateFromPassword([]byte("@users123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("❌ Failed to hash user password:", err)
	}

	// --- Create Admin User ---
	adminUser := db.User{
		FullName: "Super Admin",
		Email:    "admin@admin.com",
		Password: string(adminPass),
		Role:     constants.RoleAdmin,
		Balance:  1000000, // Lebih banyak saldo untuk admin
		Status:   constants.UserStatusActive,
	}
	if err := dbConn.Create(&adminUser).Error; err != nil {
		log.Fatal("❌ Failed to seed admin user:", err)
	}
	fmt.Println("✅ Admin user created.")

	// --- Create Initial Users ---
	var users []db.User
	initialUser1 := db.User{
		FullName: "Budi Santoso",
		Email:    "budi@user.com",
		Password: string(userPass),
		Role:     constants.RoleUser,
		Balance:  500000,
		Status:   constants.UserStatusActive,
	}
	users = append(users, initialUser1)

	initialUser2 := db.User{
		FullName: "Citra Dewi",
		Email:    "citra@user.com",
		Password: string(userPass),
		Role:     constants.RoleUser,
		Balance:  350000,
		Status:   constants.UserStatusActive,
	}
	users = append(users, initialUser2)

	// User with suspended status
	suspendedUser := db.User{
		FullName: "Doni Suspend",
		Email:    "doni@suspended.com",
		Password: string(userPass),
		Role:     constants.RoleUser,
		Balance:  10000,
		Status:   constants.UserStatusSuspended,
		BanReason: "Melanggar kebijakan penggunaan berulang kali.",
	}
	users = append(users, suspendedUser)

	// User with penalty warnings
	penaltyUser := db.User{
		FullName: "Eka Peringatan",
		Email:    "eka@penalty.com",
		Password: string(userPass),
		Role:     constants.RoleUser,
		Balance:  200000,
		Status:   constants.UserStatusActive,
		PenaltyWarnings: 2, // Mendekati batas penalti
	}
	users = append(users, penaltyUser)

	// Create 10 more varied users
	for i := 1; i <= 10; i++ {
		user := db.User{
			FullName: fmt.Sprintf("Pengguna %d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Password: string(userPass),
			Role:     constants.RoleUser,
			Balance:  uint(rand.Intn(500000) + 50000), // Saldo acak antara 50rb - 550rb
			Status:   constants.UserStatusActive,
		}
		users = append(users, user)
	}

	for _, user := range users {
		if err := dbConn.Create(&user).Error; err != nil {
			log.Fatalf("❌ Failed to seed user %s: %v", user.Email, err)
		}
		fmt.Printf("✅ User %s created.\n", user.Email)
	}

	// --- Products for Initial Users ---
	var products []db.Product

	// Products for Budi Santoso (initialUser1)
	products = append(products, db.Product{
		Title:      "Smartphone Terbaru X1",
		Price:      3500000,
		Stock:      15,
		UserID:     initialUser1.ID,
		Categories: "Elektronik,Gadget",
		Visibility: constants.ProductVisibilityAll,
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image1.png"},
			{ImageURL: "/media/products/dummy_image2.png"},
		},
	})
	products = append(products, db.Product{
		Title:      "Laptop Gaming Pro",
		Price:      12000000,
		Stock:      5,
		UserID:     initialUser1.ID,
		Categories: "Elektronik,Komputer,Gaming",
		Visibility: constants.ProductVisibilityAll,
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image3.png"},
		},
	})

	// Products for Citra Dewi (initialUser2)
	products = append(products, db.Product{
		Title:      "Buku Fiksi Fantasi: Dunia Paralel",
		Price:      95000,
		Stock:      50,
		UserID:     initialUser2.ID,
		Categories: "Buku,Fiksi",
		Visibility: constants.ProductVisibilityAll,
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image4.png"},
		},
	})
	products = append(products, db.Product{
		Title:      "Set Peralatan Dapur Lengkap",
		Price:      750000,
		Stock:      10,
		UserID:     initialUser2.ID,
		Categories: "Rumah Tangga,Dapur",
		Visibility: constants.ProductVisibilityOwnerAdmin, // Hanya terlihat oleh pemilik dan admin
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image1.png"},
		},
	})

	// Product for Doni Suspend (suspendedUser) - should be owner_admin due to user status
	products = append(products, db.Product{
		Title:      "Kamera DSLR Profesional",
		Price:      8000000,
		Stock:      3,
		UserID:     suspendedUser.ID,
		Categories: "Elektronik,Fotografi",
		Visibility: constants.ProductVisibilityOwnerAdmin, // Otomatis jadi owner_admin karena user suspended
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image2.png"},
		},
	})

	// Product for Eka Peringatan (penaltyUser)
	products = append(products, db.Product{
		Title:      "Headphone Noise Cancelling",
		Price:      1200000,
		Stock:      20,
		UserID:     penaltyUser.ID,
		Categories: "Elektronik,Audio",
		Visibility: constants.ProductVisibilityAll,
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image3.png"},
		},
	})

	// --- Products for the 10 new users (each has 1 product) ---
	productTitles := []string{
		"Smartwatch Canggih", "Drone Mini Lipat", "Speaker Bluetooth Portabel",
		"Meja Kerja Ergonomis", "Kursi Gaming Premium", "Monitor Ultrawide 27 Inci",
		"Keyboard Mekanikal RGB", "Mouse Gaming Presisi", "Webcam Full HD", "Router Wi-Fi 6",
	}
	productCategories := []string{
		"Elektronik,Gadget", "Elektronik,Hobi", "Elektronik,Audio",
		"Perabotan,Kantor", "Perabotan,Gaming", "Elektronik,Komputer",
		"Komputer,Aksesoris", "Komputer,Aksesoris", "Komputer,Aksesoris", "Jaringan,Elektronik",
	}
	productPrices := []uint{
		1500000, 2000000, 450000,
		800000, 2500000, 3000000,
		700000, 300000, 250000, 900000,
	}
	productStocks := []uint{
		10, 8, 25,
		5, 3, 7,
		12, 18, 30, 9,
	}

	for i := 0; i < 10; i++ {
		user := users[4+i] // Start from the 5th user (index 4) in the 'users' slice
		products = append(products, db.Product{
			Title:      productTitles[i],
			Price:      productPrices[i],
			Stock:      productStocks[i],
			UserID:     user.ID,
			Categories: productCategories[i],
			Visibility: constants.ProductVisibilityAll,
			IsActive:   true,
			Images: []db.ProductImage{
				{ImageURL: fmt.Sprintf("/media/products/dummy_image%d.png", (i%4)+1)}, // Cycle through dummy images
			},
		})
	}

	// Add a product with 0 stock
	products = append(products, db.Product{
		Title:      "Kopi Arabika Premium (Stok Habis)",
		Price:      75000,
		Stock:      0, // Stok 0
		UserID:     initialUser1.ID,
		Categories: "Makanan,Minuman",
		Visibility: constants.ProductVisibilityOwnerAdmin, // Otomatis owner_admin jika stok 0
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image4.png"},
		},
	})

	// Add a product with admin_only visibility
	products = append(products, db.Product{
		Title:      "Prototype Gadget Rahasia",
		Price:      99999999, // Harga sangat tinggi
		Stock:      1,
		UserID:     adminUser.ID, // Dimiliki oleh admin
		Categories: "Elektronik,Prototype",
		Visibility: constants.ProductVisibilityAdminOnly, // Hanya terlihat oleh admin
		IsActive:   true,
		Images: []db.ProductImage{
			{ImageURL: "/media/products/dummy_image1.png"},
		},
	})

	for _, product := range products {
		if err := dbConn.Create(&product).Error; err != nil {
			log.Fatalf("❌ Failed to seed product %s: %v", product.Title, err)
		}
		fmt.Printf("✅ Product '%s' created.\n", product.Title)
	}

	// --- Reviews ---
	var reviews []db.Review
	reviews = append(reviews, db.Review{
		UserID:    initialUser2.ID,
		ProductID: products[0].ID, // Smartphone Terbaru X1
		Rating:    5,
		Comment:   "Sangat puas dengan smartphone ini! Cepat dan kameranya bagus.",
	})
	reviews = append(reviews, db.Review{
		UserID:    initialUser1.ID,
		ProductID: products[2].ID, // Buku Fiksi Fantasi
		Rating:    4,
		Comment:   "Ceritanya menarik, tapi pengiriman agak lambat.",
	})
	reviews = append(reviews, db.Review{
		UserID:    users[4].ID, // Pengguna 1
		ProductID: products[0].ID, // Smartphone Terbaru X1
		Rating:    4,
		Comment:   "Harga sesuai kualitas, recommended!",
	})
	reviews = append(reviews, db.Review{
		UserID:    initialUser2.ID,
		ProductID: products[5].ID, // Headphone Noise Cancelling
		Rating:    5,
		Comment:   "Suara jernih, noise cancellingnya bekerja dengan baik.",
	})

	for _, review := range reviews {
		if err := dbConn.Create(&review).Error; err != nil {
			log.Fatalf("❌ Failed to seed review: %v", err)
		}
		fmt.Printf("✅ Review for product %d created.\n", review.ProductID)
	}

	// --- Transactions ---
	var transactions []db.TransactionHistory

	// Transaction 1: Pending (Budi buys from Citra)
	transactions = append(transactions, db.TransactionHistory{
		ProductID:    products[2].ID, // Buku Fiksi Fantasi
		UserID:       initialUser1.ID,
		Quantity:     1,
		TotalPrice:   products[2].Price,
		GovtTax:      uint(float64(products[2].Price) * constants.DefaultGovtTaxPercent),
		EcommerceTax: uint(float64(products[2].Price) * constants.DefaultEcommerceTaxPercent),
		Status:       constants.TrxStatusPending,
		IsSolved:     false,
		ReceiptStatus: constants.ReceiptPendingProcess,
	})

	// Transaction 2: Waiting User (Citra buys from Budi, Budi confirmed shipment)
	transactions = append(transactions, db.TransactionHistory{
		ProductID:    products[0].ID, // Smartphone Terbaru X1
		UserID:       initialUser2.ID,
		Quantity:     1,
		TotalPrice:   products[0].Price,
		GovtTax:      uint(float64(products[0].Price) * constants.DefaultGovtTaxPercent),
		EcommerceTax: uint(float64(products[0].Price) * constants.DefaultEcommerceTaxPercent),
		Status:       constants.TrxStatusWaitingUser,
		IsSolved:     false,
		ReceiptStatus: constants.ReceiptPendingProcess,
	})

	// Transaction 3: Success (Pengguna 1 buys from Budi)
	transactions = append(transactions, db.TransactionHistory{
		ProductID:    products[1].ID, // Laptop Gaming Pro
		UserID:       users[4].ID, // Pengguna 1
		Quantity:     1,
		TotalPrice:   products[1].Price,
		GovtTax:      uint(float64(products[1].Price) * constants.DefaultGovtTaxPercent),
		EcommerceTax: uint(float64(products[1].Price) * constants.DefaultEcommerceTaxPercent),
		Status:       constants.TrxStatusSuccess,
		IsSolved:     true,
		ReceiptStatus: constants.ReceiptCompleted,
	})

	// Transaction 4: Cancelled (Pengguna 2 buys from Citra, then cancelled)
	transactions = append(transactions, db.TransactionHistory{
		ProductID:    products[2].ID, // Buku Fiksi Fantasi
		UserID:       users[5].ID, // Pengguna 2
		Quantity:     1,
		TotalPrice:   products[2].Price,
		GovtTax:      uint(float64(products[2].Price) * constants.DefaultGovtTaxPercent),
		EcommerceTax: uint(float64(products[2].Price) * constants.DefaultEcommerceTaxPercent),
		Status:       constants.TrxStatusCancel,
		IsSolved:     true,
		ReceiptStatus: constants.ReceiptCanceled,
	})

	for _, trx := range transactions {
		if err := dbConn.Create(&trx).Error; err != nil {
			log.Fatalf("❌ Failed to seed transaction: %v", err)
		}
		fmt.Printf("✅ Transaction %d created.\n", trx.ID)
	}

	fmt.Println("🎉 Database seeded successfully with varied data!")
}
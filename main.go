package main

import (
	"database/sql"
	"log"
	"time"

	"inventory/handler"
	"inventory/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func initDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS skus (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		sku_code     TEXT    NOT NULL UNIQUE,
		name         TEXT    NOT NULL,
		total_qty    INTEGER NOT NULL DEFAULT 0,
		locked_qty   INTEGER NOT NULL DEFAULT 0,
		available_qty INTEGER NOT NULL DEFAULT 0,
		created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS orders (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		order_no   TEXT    NOT NULL UNIQUE,
		status     TEXT    NOT NULL DEFAULT 'created',
		expire_at  DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id INTEGER NOT NULL REFERENCES orders(id),
		sku_id   INTEGER NOT NULL REFERENCES skus(id),
		qty     INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS inventory_locks (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id   INTEGER NOT NULL REFERENCES orders(id),
		sku_id     INTEGER NOT NULL REFERENCES skus(id),
		qty        INTEGER NOT NULL,
		status     TEXT    NOT NULL DEFAULT 'locked',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		released_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS proxy_rules (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL UNIQUE,
		listen_port INTEGER NOT NULL,
		target_host TEXT    NOT NULL,
		target_port INTEGER NOT NULL,
		protocol    TEXT    NOT NULL DEFAULT 'tcp',
		enabled     INTEGER NOT NULL DEFAULT 1,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	db, err := initDB("./inventory.db")
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer db.Close()

	invSvc := service.NewInventoryService(db)
	invH := handler.NewInventoryHandler(invSvc)

	proxySvc := service.NewProxyService(db)
	proxyH := handler.NewProxyHandler(proxySvc)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		log.Println("expired order release job started, interval=1min")
		for range ticker.C {
			result, err := invSvc.ReleaseExpiredOrders()
			if err != nil {
				log.Printf("release expired orders failed: %v", err)
				continue
			}
			if result.ReleasedOrderCount > 0 {
				log.Printf("released expired orders: orders=%d, locks=%d", result.ReleasedOrderCount, result.ReleasedLockCount)
			}
		}
	}()

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		api.POST("/skus", invH.CreateSKU)
		api.GET("/skus/:sku_code", invH.GetSKU)
		api.POST("/orders", invH.CreateOrder)
		api.POST("/orders/complete", invH.CompleteOrder)
		api.POST("/orders/cancel", invH.CancelOrder)
		api.POST("/orders/release-expired", invH.ReleaseExpiredOrders)

		proxy := api.Group("/proxy")
		{
			proxy.POST("", proxyH.CreateRule)
			proxy.GET("", proxyH.ListRules)
			proxy.GET("/:id", proxyH.GetRule)
			proxy.GET("/port/:port", proxyH.GetRuleByPort)
			proxy.PUT("/:id", proxyH.UpdateRule)
			proxy.DELETE("/:id", proxyH.DeleteRule)
			proxy.POST("/:id/enable", proxyH.EnableRule)
			proxy.POST("/:id/disable", proxyH.DisableRule)
			proxy.POST("/port/enable", proxyH.EnableRuleByPort)
			proxy.POST("/port/disable", proxyH.DisableRuleByPort)
		}
	}

	log.Println("server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

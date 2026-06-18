package model

import "time"

type SKU struct {
	ID        int64
	SkuCode   string
	Name      string
	TotalQty  int
	LockedQty int
	AvailableQty int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Order struct {
	ID        int64
	OrderNo   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ID      int64
	OrderID int64
	SkuID   int64
	Qty     int
}

type InventoryLock struct {
	ID        int64
	OrderID   int64
	SkuID     int64
	Qty       int
	Status    string
	CreatedAt time.Time
	ReleasedAt *time.Time
}

const (
	OrderStatusCreated   = "created"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"

	LockStatusLocked   = "locked"
	LockStatusReleased = "released"
)

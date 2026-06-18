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
	ExpireAt  time.Time
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

type ProxyRule struct {
	ID         int64
	Name       string
	ListenPort int
	TargetHost string
	TargetPort int
	Protocol   string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const (
	OrderStatusCreated   = "created"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
	OrderStatusExpired   = "expired"

	LockStatusLocked   = "locked"
	LockStatusReleased = "released"

	DefaultOrderTTLMinutes = 30

	ProxyProtocolTCP = "tcp"
	ProxyProtocolUDP = "udp"
)

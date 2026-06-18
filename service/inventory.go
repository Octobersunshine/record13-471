package service

import (
	"database/sql"
	"errors"
	"fmt"

	"inventory/model"
)

type InventoryService struct {
	db *sql.DB
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
}

type OrderRequest struct {
	OrderNo string             `json:"order_no"`
	Items   []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	SkuCode string `json:"sku_code"`
	Qty     int    `json:"qty"`
}

func (s *InventoryService) CreateOrder(req OrderRequest) (*model.Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var order model.Order
	err = tx.QueryRow(
		"INSERT INTO orders (order_no, status) VALUES (?, ?) RETURNING id, order_no, status, created_at, updated_at",
		req.OrderNo, model.OrderStatusCreated,
	).Scan(&order.ID, &order.OrderNo, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	for _, item := range req.Items {
		var sku model.SKU
		err := tx.QueryRow(
			"SELECT id, sku_code, name, total_qty, locked_qty, available_qty FROM skus WHERE sku_code = ?",
			item.SkuCode,
		).Scan(&sku.ID, &sku.SkuCode, &sku.Name, &sku.TotalQty, &sku.LockedQty, &sku.AvailableQty)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("sku not found: %s", item.SkuCode)
			}
			return nil, fmt.Errorf("query sku: %w", err)
		}

		if sku.AvailableQty < item.Qty {
			return nil, fmt.Errorf("insufficient stock for sku %s: available=%d, required=%d", item.SkuCode, sku.AvailableQty, item.Qty)
		}

		_, err = tx.Exec(
			"UPDATE skus SET locked_qty = locked_qty + ?, available_qty = available_qty - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			item.Qty, item.Qty, sku.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("lock stock: %w", err)
		}

		_, err = tx.Exec(
			"INSERT INTO order_items (order_id, sku_id, qty) VALUES (?, ?, ?)",
			order.ID, sku.ID, item.Qty,
		)
		if err != nil {
			return nil, fmt.Errorf("insert order item: %w", err)
		}

		_, err = tx.Exec(
			"INSERT INTO inventory_locks (order_id, sku_id, qty, status) VALUES (?, ?, ?, ?)",
			order.ID, sku.ID, item.Qty, model.LockStatusLocked,
		)
		if err != nil {
			return nil, fmt.Errorf("insert inventory lock: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &order, nil
}

func (s *InventoryService) CompleteOrder(orderNo string) (*model.Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var order model.Order
	err = tx.QueryRow(
		"SELECT id, order_no, status, created_at, updated_at FROM orders WHERE order_no = ?",
		orderNo,
	).Scan(&order.ID, &order.OrderNo, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order not found: %s", orderNo)
		}
		return nil, fmt.Errorf("query order: %w", err)
	}

	if order.Status != model.OrderStatusCreated {
		return nil, fmt.Errorf("order %s cannot be completed: current status=%s", orderNo, order.Status)
	}

	rows, err := tx.Query(
		"SELECT il.id, il.sku_id, il.qty FROM inventory_locks il WHERE il.order_id = ? AND il.status = ?",
		order.ID, model.LockStatusLocked,
	)
	if err != nil {
		return nil, fmt.Errorf("query locks: %w", err)
	}
	defer rows.Close()

	type lockInfo struct {
		ID    int64
		SkuID int64
		Qty   int
	}
	var locks []lockInfo
	for rows.Next() {
		var l lockInfo
		if err := rows.Scan(&l.ID, &l.SkuID, &l.Qty); err != nil {
			return nil, fmt.Errorf("scan lock: %w", err)
		}
		locks = append(locks, l)
	}

	for _, l := range locks {
		_, err = tx.Exec(
			"UPDATE skus SET locked_qty = locked_qty - ?, total_qty = total_qty - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			l.Qty, l.Qty, l.SkuID,
		)
		if err != nil {
			return nil, fmt.Errorf("release stock for sku_id=%d: %w", l.SkuID, err)
		}

		_, err = tx.Exec(
			"UPDATE inventory_locks SET status = ?, released_at = CURRENT_TIMESTAMP WHERE id = ?",
			model.LockStatusReleased, l.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("update lock status: %w", err)
		}
	}

	_, err = tx.Exec(
		"UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		model.OrderStatusCompleted, order.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	order.Status = model.OrderStatusCompleted
	return &order, nil
}

func (s *InventoryService) CancelOrder(orderNo string) (*model.Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var order model.Order
	err = tx.QueryRow(
		"SELECT id, order_no, status, created_at, updated_at FROM orders WHERE order_no = ?",
		orderNo,
	).Scan(&order.ID, &order.OrderNo, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order not found: %s", orderNo)
		}
		return nil, fmt.Errorf("query order: %w", err)
	}

	if order.Status != model.OrderStatusCreated {
		return nil, fmt.Errorf("order %s cannot be cancelled: current status=%s", orderNo, order.Status)
	}

	rows, err := tx.Query(
		"SELECT il.id, il.sku_id, il.qty FROM inventory_locks il WHERE il.order_id = ? AND il.status = ?",
		order.ID, model.LockStatusLocked,
	)
	if err != nil {
		return nil, fmt.Errorf("query locks: %w", err)
	}
	defer rows.Close()

	type lockInfo struct {
		ID    int64
		SkuID int64
		Qty   int
	}
	var locks []lockInfo
	for rows.Next() {
		var l lockInfo
		if err := rows.Scan(&l.ID, &l.SkuID, &l.Qty); err != nil {
			return nil, fmt.Errorf("scan lock: %w", err)
		}
		locks = append(locks, l)
	}

	for _, l := range locks {
		_, err = tx.Exec(
			"UPDATE skus SET locked_qty = locked_qty - ?, available_qty = available_qty + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			l.Qty, l.Qty, l.SkuID,
		)
		if err != nil {
			return nil, fmt.Errorf("restore stock for sku_id=%d: %w", l.SkuID, err)
		}

		_, err = tx.Exec(
			"UPDATE inventory_locks SET status = ?, released_at = CURRENT_TIMESTAMP WHERE id = ?",
			model.LockStatusReleased, l.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("update lock status: %w", err)
		}
	}

	_, err = tx.Exec(
		"UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		model.OrderStatusCancelled, order.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	order.Status = model.OrderStatusCancelled
	return &order, nil
}

func (s *InventoryService) GetSKU(skuCode string) (*model.SKU, error) {
	var sku model.SKU
	err := s.db.QueryRow(
		"SELECT id, sku_code, name, total_qty, locked_qty, available_qty, created_at, updated_at FROM skus WHERE sku_code = ?",
		skuCode,
	).Scan(&sku.ID, &sku.SkuCode, &sku.Name, &sku.TotalQty, &sku.LockedQty, &sku.AvailableQty, &sku.CreatedAt, &sku.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("sku not found: %s", skuCode)
		}
		return nil, fmt.Errorf("query sku: %w", err)
	}
	return &sku, nil
}

func (s *InventoryService) CreateSKU(skuCode, name string, totalQty int) (*model.SKU, error) {
	var sku model.SKU
	err := s.db.QueryRow(
		"INSERT INTO skus (sku_code, name, total_qty, locked_qty, available_qty) VALUES (?, ?, ?, 0, ?) RETURNING id, sku_code, name, total_qty, locked_qty, available_qty, created_at, updated_at",
		skuCode, name, totalQty, totalQty,
	).Scan(&sku.ID, &sku.SkuCode, &sku.Name, &sku.TotalQty, &sku.LockedQty, &sku.AvailableQty, &sku.CreatedAt, &sku.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert sku: %w", err)
	}
	return &sku, nil
}

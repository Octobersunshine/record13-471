package handler

import (
	"net/http"

	"inventory/service"

	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	svc *service.InventoryService
}

func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

type response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response{Code: 0, Message: "ok", Data: data})
}

func fail(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, response{Code: -1, Message: msg})
}

type createSKUReq struct {
	SkuCode  string `json:"sku_code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	TotalQty int    `json:"total_qty" binding:"required,gt=0"`
}

func (h *InventoryHandler) CreateSKU(c *gin.Context) {
	var req createSKUReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	sku, err := h.svc.CreateSKU(req.SkuCode, req.Name, req.TotalQty)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, sku)
}

func (h *InventoryHandler) GetSKU(c *gin.Context) {
	skuCode := c.Param("sku_code")
	sku, err := h.svc.GetSKU(skuCode)
	if err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}
	ok(c, sku)
}

func (h *InventoryHandler) CreateOrder(c *gin.Context) {
	var req service.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.svc.CreateOrder(req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, order)
}

type orderNoReq struct {
	OrderNo string `json:"order_no" binding:"required"`
}

func (h *InventoryHandler) CompleteOrder(c *gin.Context) {
	var req orderNoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.svc.CompleteOrder(req.OrderNo)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, order)
}

func (h *InventoryHandler) CancelOrder(c *gin.Context) {
	var req orderNoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.svc.CancelOrder(req.OrderNo)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, order)
}

func (h *InventoryHandler) ReleaseExpiredOrders(c *gin.Context) {
	result, err := h.svc.ReleaseExpiredOrders()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, result)
}

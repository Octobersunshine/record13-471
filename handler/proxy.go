package handler

import (
	"net/http"
	"strconv"

	"inventory/service"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	svc *service.ProxyService
}

func NewProxyHandler(svc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{svc: svc}
}

func (h *ProxyHandler) CreateRule(c *gin.Context) {
	var req service.CreateProxyRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.CreateRule(req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) GetRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	rule, err := h.svc.GetRule(id)
	if err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) GetRuleByPort(c *gin.Context) {
	port, err := strconv.Atoi(c.Param("port"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid port")
		return
	}
	rule, err := h.svc.GetRuleByPort(port)
	if err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) ListRules(c *gin.Context) {
	rules, err := h.svc.ListRules()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rules)
}

func (h *ProxyHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req service.UpdateProxyRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.UpdateRule(id, req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteRule(id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *ProxyHandler) EnableRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	rule, err := h.svc.EnableRule(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) DisableRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	rule, err := h.svc.DisableRule(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

type toggleByPortReq struct {
	Port int `json:"port" binding:"required,gt=0,lte=65535"`
}

func (h *ProxyHandler) EnableRuleByPort(c *gin.Context) {
	var req toggleByPortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.EnableRuleByPort(req.Port)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

func (h *ProxyHandler) DisableRuleByPort(c *gin.Context) {
	var req toggleByPortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.DisableRuleByPort(req.Port)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rule)
}

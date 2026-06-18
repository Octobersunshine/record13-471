package service

import (
	"database/sql"
	"errors"
	"fmt"

	"inventory/model"
)

type ProxyService struct {
	db *sql.DB
}

func NewProxyService(db *sql.DB) *ProxyService {
	return &ProxyService{db: db}
}

type CreateProxyRuleReq struct {
	Name       string `json:"name" binding:"required"`
	ListenPort int    `json:"listen_port" binding:"required,gt=0,lte=65535"`
	TargetHost string `json:"target_host" binding:"required"`
	TargetPort int    `json:"target_port" binding:"required,gt=0,lte=65535"`
	Protocol   string `json:"protocol" binding:"required,oneof=tcp udp"`
}

type UpdateProxyRuleReq struct {
	Name       string `json:"name"`
	ListenPort int    `json:"listen_port" binding:"omitempty,gt=0,lte=65535"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port" binding:"omitempty,gt=0,lte=65535"`
	Protocol   string `json:"protocol" binding:"omitempty,oneof=tcp udp"`
}

func (s *ProxyService) CreateRule(req CreateProxyRuleReq) (*model.ProxyRule, error) {
	var rule model.ProxyRule
	enabled := 1
	err := s.db.QueryRow(
		"INSERT INTO proxy_rules (name, listen_port, target_host, target_port, protocol, enabled) VALUES (?, ?, ?, ?, ?, ?) RETURNING id, name, listen_port, target_host, target_port, protocol, enabled, created_at, updated_at",
		req.Name, req.ListenPort, req.TargetHost, req.TargetPort, req.Protocol, enabled,
	).Scan(&rule.ID, &rule.Name, &rule.ListenPort, &rule.TargetHost, &rule.TargetPort, &rule.Protocol, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert proxy rule: %w", err)
	}
	return &rule, nil
}

func (s *ProxyService) GetRule(id int64) (*model.ProxyRule, error) {
	var rule model.ProxyRule
	err := s.db.QueryRow(
		"SELECT id, name, listen_port, target_host, target_port, protocol, enabled, created_at, updated_at FROM proxy_rules WHERE id = ?",
		id,
	).Scan(&rule.ID, &rule.Name, &rule.ListenPort, &rule.TargetHost, &rule.TargetPort, &rule.Protocol, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("proxy rule not found: id=%d", id)
		}
		return nil, fmt.Errorf("query proxy rule: %w", err)
	}
	return &rule, nil
}

func (s *ProxyService) GetRuleByPort(port int) (*model.ProxyRule, error) {
	var rule model.ProxyRule
	err := s.db.QueryRow(
		"SELECT id, name, listen_port, target_host, target_port, protocol, enabled, created_at, updated_at FROM proxy_rules WHERE listen_port = ?",
		port,
	).Scan(&rule.ID, &rule.Name, &rule.ListenPort, &rule.TargetHost, &rule.TargetPort, &rule.Protocol, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("proxy rule not found: port=%d", port)
		}
		return nil, fmt.Errorf("query proxy rule by port: %w", err)
	}
	return &rule, nil
}

func (s *ProxyService) ListRules() ([]model.ProxyRule, error) {
	rows, err := s.db.Query(
		"SELECT id, name, listen_port, target_host, target_port, protocol, enabled, created_at, updated_at FROM proxy_rules ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query proxy rules: %w", err)
	}
	defer rows.Close()

	var rules []model.ProxyRule
	for rows.Next() {
		var r model.ProxyRule
		if err := rows.Scan(&r.ID, &r.Name, &r.ListenPort, &r.TargetHost, &r.TargetPort, &r.Protocol, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan proxy rule: %w", err)
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return rules, nil
}

func (s *ProxyService) UpdateRule(id int64, req UpdateProxyRuleReq) (*model.ProxyRule, error) {
	existing, err := s.GetRule(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.ListenPort > 0 {
		existing.ListenPort = req.ListenPort
	}
	if req.TargetHost != "" {
		existing.TargetHost = req.TargetHost
	}
	if req.TargetPort > 0 {
		existing.TargetPort = req.TargetPort
	}
	if req.Protocol != "" {
		existing.Protocol = req.Protocol
	}

	_, err = s.db.Exec(
		"UPDATE proxy_rules SET name = ?, listen_port = ?, target_host = ?, target_port = ?, protocol = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		existing.Name, existing.ListenPort, existing.TargetHost, existing.TargetPort, existing.Protocol, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update proxy rule: %w", err)
	}

	return s.GetRule(id)
}

func (s *ProxyService) DeleteRule(id int64) error {
	result, err := s.db.Exec("DELETE FROM proxy_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete proxy rule: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("proxy rule not found: id=%d", id)
	}
	return nil
}

func (s *ProxyService) EnableRule(id int64) (*model.ProxyRule, error) {
	_, err := s.GetRule(id)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(
		"UPDATE proxy_rules SET enabled = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("enable proxy rule: %w", err)
	}
	return s.GetRule(id)
}

func (s *ProxyService) DisableRule(id int64) (*model.ProxyRule, error) {
	_, err := s.GetRule(id)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(
		"UPDATE proxy_rules SET enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("disable proxy rule: %w", err)
	}
	return s.GetRule(id)
}

func (s *ProxyService) EnableRuleByPort(port int) (*model.ProxyRule, error) {
	rule, err := s.GetRuleByPort(port)
	if err != nil {
		return nil, err
	}
	return s.EnableRule(rule.ID)
}

func (s *ProxyService) DisableRuleByPort(port int) (*model.ProxyRule, error) {
	rule, err := s.GetRuleByPort(port)
	if err != nil {
		return nil, err
	}
	return s.DisableRule(rule.ID)
}

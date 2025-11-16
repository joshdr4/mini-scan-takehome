package scan_manager

import (
	"context"
	"errors"
	"fmt"
)

type ScanResult struct {
	IP          string
	Port        uint32
	Service     string
	Timestamp   int64
	Response    string
	DataVersion int
}

type Repository interface {
	Put(ctx context.Context, result *ScanResult) error
}

type ScanManagerConfig struct {
	Repo Repository
}

type scanManager struct {
	repo Repository
}

func NewScanManager(cfg *ScanManagerConfig) (*scanManager, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if cfg.Repo == nil {
		return nil, errors.New("repository is nil")
	}

	manager := &scanManager{
		repo: cfg.Repo,
	}

	return manager, nil
}

func (m *scanManager) PutScan(ctx context.Context, result *ScanResult) error {
	if err := m.repo.Put(ctx, result); err != nil {
		return fmt.Errorf("failed to put scan: %w", err)
	}
	fmt.Printf("scan result stored: %+v\n", result)

	return nil
}

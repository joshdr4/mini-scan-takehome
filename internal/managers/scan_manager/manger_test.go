package scan_manager

import (
	"context"
	"errors"
	"testing"
)

// MockRepository for testing
type MockRepository struct {
	ShouldFail bool
}

func (m *MockRepository) Put(ctx context.Context, result *ScanResult) error {
	if m.ShouldFail {
		return errors.New("repository error")
	}
	return nil
}

func TestNewScanManager(t *testing.T) {
	t.Run("should return error if config is nil", func(t *testing.T) {
		_, err := NewScanManager(nil)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("should return error if repo is nil", func(t *testing.T) {
		_, err := NewScanManager(&ScanManagerConfig{})
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("should return a new manager", func(t *testing.T) {
		mockRepo := &MockRepository{}

		_, err := NewScanManager(&ScanManagerConfig{
			Repo: mockRepo,
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPutScan(t *testing.T) {
	t.Run("should successfully put scan", func(t *testing.T) {
		mockRepo := &MockRepository{ShouldFail: false}
		manager, _ := NewScanManager(&ScanManagerConfig{
			Repo: mockRepo,
		})

		result := &ScanResult{
			IP:          "192.168.1.1",
			Port:        80,
			Service:     "http",
			Timestamp:   1234567890,
			Response:    "test response",
			DataVersion: 1,
		}

		err := manager.PutScan(context.Background(), result)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("should fail when repository fails", func(t *testing.T) {
		mockRepo := &MockRepository{ShouldFail: true}
		manager, _ := NewScanManager(&ScanManagerConfig{
			Repo: mockRepo,
		})

		result := &ScanResult{
			IP:          "192.168.1.1",
			Port:        80,
			Service:     "http",
			Timestamp:   1234567890,
			Response:    "test response",
			DataVersion: 1,
		}

		err := manager.PutScan(context.Background(), result)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}

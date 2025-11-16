package serializer

import (
	"encoding/base64"
	"testing"

	"github.com/censys/scan-takehome/pkg/scanning"
)

func TestParseScanMessage_V1(t *testing.T) {
	response := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(response))

	jsonData := `{
		"ip": "192.168.1.1",
		"port": 80,
		"service": "http",
		"timestamp": 1234567890,
		"data_version": 1,
		"data": {
			"response_bytes_utf8": "` + encoded + `"
		}
	}`

	result, err := ParseScanMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseScanMessage failed: %v", err)
	}

	if result.IP != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", result.IP)
	}

	if result.Port != 80 {
		t.Errorf("Expected Port 80, got %d", result.Port)
	}

	if result.Service != "http" {
		t.Errorf("Expected Service http, got %s", result.Service)
	}

	if result.Timestamp != 1234567890 {
		t.Errorf("Expected Timestamp 1234567890, got %d", result.Timestamp)
	}

	if result.DataVersion != scanning.V1 {
		t.Errorf("Expected DataVersion V1, got %d", result.DataVersion)
	}

	if result.Response != response {
		t.Errorf("Expected Response '%s', got '%s'", response, result.Response)
	}
}

func TestParseScanMessage_V2(t *testing.T) {
	// V2 format with plain string response
	response := "service response"

	jsonData := `{
		"ip": "10.0.0.1",
		"port": 443,
		"service": "https",
		"timestamp": 9876543210,
		"data_version": 2,
		"data": {
			"response_str": "` + response + `"
		}
	}`

	result, err := ParseScanMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseScanMessage failed: %v", err)
	}

	if result.IP != "10.0.0.1" {
		t.Errorf("Expected IP 10.0.0.1, got %s", result.IP)
	}

	if result.Port != 443 {
		t.Errorf("Expected Port 443, got %d", result.Port)
	}

	if result.Service != "https" {
		t.Errorf("Expected Service https, got %s", result.Service)
	}

	if result.Timestamp != 9876543210 {
		t.Errorf("Expected Timestamp 9876543210, got %d", result.Timestamp)
	}

	if result.DataVersion != scanning.V2 {
		t.Errorf("Expected DataVersion V2, got %d", result.DataVersion)
	}

	if result.Response != response {
		t.Errorf("Expected Response '%s', got '%s'", response, result.Response)
	}
}

func TestParseScanMessage_InvalidJSON(t *testing.T) {
	jsonData := `{"invalid json`

	_, err := ParseScanMessage([]byte(jsonData))

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

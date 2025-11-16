package serializer

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/censys/scan-takehome/internal/managers/scan_manager"
	"github.com/censys/scan-takehome/pkg/scanning"
)

// ParseScanMessage deserializes and normalizes scan data from raw bytes
// Handles both V1 (base64 encoded) and V2 (plain string) data formats
func ParseScanMessage(data []byte) (*scan_manager.ScanResult, error) {
	var scan scanning.Scan
	if err := json.Unmarshal(data, &scan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scan data: %w", err)
	}

	if scan.Data == nil {
		return nil, errors.New("scan data is nil")
	}

	var response string

	dataBytes, err := json.Marshal(scan.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	switch scan.DataVersion {
	case scanning.V1:
		var v1 scanning.V1Data
		if err := json.Unmarshal(dataBytes, &v1); err != nil {
			return nil, fmt.Errorf("failed to unmarshal V1 data: %w", err)
		}
		response = string(v1.ResponseBytesUtf8)

	case scanning.V2:
		var v2 scanning.V2Data
		if err := json.Unmarshal(dataBytes, &v2); err != nil {
			return nil, fmt.Errorf("failed to unmarshal V2 data: %w", err)
		}
		response = v2.ResponseStr

	default:
		return nil, fmt.Errorf("unknown data version: %d", scan.DataVersion)
	}

	result := &scan_manager.ScanResult{
		IP:          scan.Ip,
		Port:        scan.Port,
		Service:     scan.Service,
		Timestamp:   scan.Timestamp,
		Response:    response,
		DataVersion: scan.DataVersion,
	}

	return result, nil
}

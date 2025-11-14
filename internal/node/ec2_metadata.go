package node

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GetEC2InstanceMetadata retrieves metadata from EC2 instance metadata service
func GetEC2InstanceMetadata() (instanceID string, privateIP string, err error) {
	// Try to get from environment variables first (for local testing)
	if id := os.Getenv("NODE_ID"); id != "" {
		instanceID = id
	} else {
		// Get instance ID from EC2 metadata
		instanceID, err = getMetadata("instance-id")
		if err != nil {
			return "", "", fmt.Errorf("failed to get instance ID: %w", err)
		}
	}

	// Get private IP
	if ip := os.Getenv("NODE_PRIVATE_IP"); ip != "" {
		privateIP = ip
	} else {
		privateIP, err = getMetadata("local-ipv4")
		if err != nil {
			return "", "", fmt.Errorf("failed to get private IP: %w", err)
		}
	}

	return instanceID, privateIP, nil
}

func getMetadata(path string) (string, error) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("http://169.254.169.254/latest/meta-data/%s", path)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetNodePort gets the port from environment or defaults to 8080
func GetNodePort() int {
	if portStr := os.Getenv("NODE_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return 8080
}

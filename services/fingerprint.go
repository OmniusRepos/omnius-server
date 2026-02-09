package services

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const machineIDFile = "data/.machine-id"

// GetMachineFingerprint returns a stable fingerprint for this machine.
// It persists a UUID to data/.machine-id and hashes it with the hostname.
func GetMachineFingerprint() (string, error) {
	machineID, err := getOrCreateMachineID()
	if err != nil {
		return "", fmt.Errorf("failed to get machine ID: %w", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	raw := machineID + ":" + hostname
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash), nil
}

func getOrCreateMachineID() (string, error) {
	dir := filepath.Dir(machineIDFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	data, err := os.ReadFile(machineIDFile)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	}

	id := uuid.New().String()
	if err := os.WriteFile(machineIDFile, []byte(id), 0600); err != nil {
		return "", fmt.Errorf("failed to write machine ID: %w", err)
	}
	log.Printf("[License] Generated machine ID: %s", id)
	return id, nil
}

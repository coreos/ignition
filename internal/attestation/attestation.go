// Copyright 2025 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package attestation

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
)

const (
	TPMDir      = "/var/tpm"
	AKPath      = "/var/tpm/ak.pub"
	AKCtxPath   = "/var/tpm/ak.ctx"
	AKRegisterd = "/var/tpm/ak.registerd"
	AKHandle    = "0x81010002"
	EKHandle    = "0x81010001"
)

func HandleAttestation(logger *log.Logger, cfg *types.Config, platformName string, needNetPath string) error {
	if !util.NilOrEmpty(cfg.Attestation.AttestationKey.Registration.Url) {
		// Generate and persist the AK
		if err := GenerateAndPersistAK(logger); err != nil {
			return err
		}

		attestationKeyBytes, err := os.ReadFile(AKPath)
		if err != nil {
			return err
		}
		attestationKey := string(attestationKeyBytes)

		// Check if the neednet file exists to determine our retry behavior
		_, needNetErr := os.Stat(needNetPath)
		needNetExists := (needNetErr == nil)
		if needNetExists {
			logger.Info("neednet file exists, network should be available for attestation")
		} else {
			logger.Info("neednet file does not exist, will return ErrNeedNet if network is unavailable")
		}

		err = AttestationKeyRegistration(logger, cfg.Attestation.AttestationKey.Registration,
			attestationKey, platformName)
		if err != nil {
			// If neednet file doesn't exist, propagate it
			// (we're in fetch-offline and need to signal for network)
			if !needNetExists {
				return err
			}
			// If we got ErrNeedNet but neednet file exists, we're in fetch stage
			// Retry the registration with delays to allow network to come up
			if err == resource.ErrNeedNet && needNetExists {
				logger.Info("Network not ready yet in fetch stage, retrying with delays...")
				// Retry up to 10 times with increasing delays
				maxRetries := 20
				for attempt := 2; attempt <= maxRetries; attempt++ {
					delay := time.Duration(min(attempt*2, 10)) * time.Second
					logger.Info("Waiting %v before retry attempt %d/%d", delay, attempt, maxRetries)
					time.Sleep(delay)

					err = AttestationKeyRegistration(logger, cfg.Attestation.AttestationKey.Registration,
						attestationKey, platformName)
					if err == nil {
						break
					}
					logger.Info("Attestation registration attempt %d/%d failed: %v", attempt, maxRetries, err)
				}
				if err != nil {
					return fmt.Errorf("failed to register attestation key after retries: %w", err)
				}
			} else {
				return err
			}
		}
	}
	return nil
}

// GenerateAndPersistAK creates and persists the Attestation Key in the TPM
func GenerateAndPersistAK(logger *log.Logger) error {
	if err := os.MkdirAll(TPMDir, 0755); err != nil {
		return fmt.Errorf("couldn't create %s directory: %w", TPMDir, err)
	}

	if _, err := os.Stat(AKPath); err == nil {
		logger.Info("Attestation Key already exists, skipping generation")
		return nil
	}

	logger.Info("Generating Attestation Key")
	cmd := exec.Command("tpm2_createak", "-C", EKHandle,
		"-c", AKCtxPath, "-G", "rsa", "-g", "sha256",
		"-s", "rsassa", "-u", AKPath, "-f", "pem")
	if _, err := logger.LogCmd(cmd, "creating attestation key"); err != nil {
		return fmt.Errorf("failed to create attestation key: %w", err)
	}

	cmd = exec.Command("tpm2_evictcontrol", "-c", AKCtxPath, AKHandle)
	if _, err := logger.LogCmd(cmd, "persisting attestation key"); err != nil {
		return fmt.Errorf("failed to persist attestation key: %w", err)
	}

	return nil
}

// AttestationKeyRegistration sends a request to register an attestation key
func AttestationKeyRegistration(logger *log.Logger, registration types.Registration, attestationKey string, platform string) error {
	if registration.Url == nil || *registration.Url == "" {
		return fmt.Errorf("registration URL is required")
	}
	// Check if AK was already generated
	if _, err := os.Stat(AKRegisterd); err == nil {
		return nil
	}

	requestBody := map[string]string{
		"public_key": attestationKey,
		"platform":   platform,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	logger.Info("Registering attestation key with URL: %s", *registration.Url)
	logger.Info("Request body: %s", string(jsonBody))

	client := &http.Client{}

	if !util.NilOrEmpty(registration.Certificate) {
		tlsConfig, err := createTLSConfig(*registration.Certificate)
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %w", err)
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	// Single attempt - caller (HandleAttestation) handles retries
	req, err := http.NewRequest(http.MethodPut, *registration.Url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.Info("HTTP request failed: %v", err)
		return resource.ErrNeedNet
	}

	defer resp.Body.Close()

	logger.Info("Received response with status code: %d", resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read response body to get error details
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			logger.Info("Failed to read error response body: %v", readErr)
			return fmt.Errorf("registration failed with status code: %d", resp.StatusCode)
		}
		logger.Info("Registration failed - Status: %d, Response body: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("registration failed with status code: %d", resp.StatusCode)
	}

	// Registration successful
	if err := os.WriteFile(AKRegisterd, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create AK registered file: %w", err)
	}
	logger.Info("Register successfully the AK")
	return nil
}

// isNetworkUnreachable checks if the error indicates network is unreachable
func isNetworkUnreachable(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Check for ENETUNREACH (network unreachable)
		if errors.Is(opErr.Err, syscall.ENETUNREACH) {
			return true
		}
		// Check for EHOSTUNREACH (host unreachable)
		if errors.Is(opErr.Err, syscall.EHOSTUNREACH) {
			return true
		}
		// Check for "connect: network is unreachable" string
		if opErr.Err != nil && opErr.Err.Error() == "network is unreachable" {
			return true
		}
	}
	return false
}

func createTLSConfig(certPEM string) (*tls.Config, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)

	return &tls.Config{
		RootCAs: certPool,
	}, nil
}

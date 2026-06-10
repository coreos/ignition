// Copyright 2026 CoreOS, Inc.
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

package cmdline

import (
	"net/url"
	"os"
	"testing"

	"github.com/coreos/ignition/v2/internal/log"
)

func newTestLogger(t *testing.T) *log.Logger {
	t.Helper()
	logger := log.New(false)
	return &logger
}

// writeCmdline writes the given cmdline string to a temp file and returns its path.
func writeCmdline(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "cmdline")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestParseCmdlineURL(t *testing.T) {
	cmdlinePath := writeCmdline(t, "ignition.config.url=https://example.com/config.ign console=tty0")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, err := url.Parse("https://example.com/config.ign")
	if err != nil {
		t.Fatalf("failed to parse expected URL: %v", err)
	}
	if opts.Url == nil || opts.Url.String() != expected.String() {
		t.Errorf("expected URL %q, got %v", expected, opts.Url)
	}
	if opts.DeviceLabel != "" {
		t.Errorf("expected empty DeviceLabel, got %q", opts.DeviceLabel)
	}
	if opts.UserDataPath != "" {
		t.Errorf("expected empty UserDataPath, got %q", opts.UserDataPath)
	}
}

func TestParseCmdlineDeviceAndPath(t *testing.T) {
	cmdlinePath := writeCmdline(t, "ignition.config.device=CONFIG ignition.config.path=/ignition/config.ign")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Url != nil {
		t.Errorf("expected nil URL, got %v", opts.Url)
	}
	if opts.DeviceLabel != "CONFIG" {
		t.Errorf("expected DeviceLabel %q, got %q", "CONFIG", opts.DeviceLabel)
	}
	if opts.UserDataPath != "/ignition/config.ign" {
		t.Errorf("expected UserDataPath %q, got %q", "/ignition/config.ign", opts.UserDataPath)
	}
}

func TestParseCmdlineDeviceOnly(t *testing.T) {
	cmdlinePath := writeCmdline(t, "ignition.config.device=CONFIG")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.DeviceLabel != "CONFIG" {
		t.Errorf("expected DeviceLabel %q, got %q", "CONFIG", opts.DeviceLabel)
	}
	if opts.UserDataPath != "" {
		t.Errorf("expected empty UserDataPath, got %q", opts.UserDataPath)
	}
}

func TestParseCmdlinePathOnly(t *testing.T) {
	cmdlinePath := writeCmdline(t, "ignition.config.path=/ignition/config.ign")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.UserDataPath != "/ignition/config.ign" {
		t.Errorf("expected UserDataPath %q, got %q", "/ignition/config.ign", opts.UserDataPath)
	}
	if opts.DeviceLabel != "" {
		t.Errorf("expected empty DeviceLabel, got %q", opts.DeviceLabel)
	}
}

func TestParseCmdlineEmptyFlagsIgnored(t *testing.T) {
	cmdlinePath := writeCmdline(t, "ignition.config.url= ignition.config.device= ignition.config.path=")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Url != nil {
		t.Errorf("expected nil URL for empty flag, got %v", opts.Url)
	}
	if opts.DeviceLabel != "" {
		t.Errorf("expected empty DeviceLabel for empty flag, got %q", opts.DeviceLabel)
	}
	if opts.UserDataPath != "" {
		t.Errorf("expected empty UserDataPath for empty flag, got %q", opts.UserDataPath)
	}
}

func TestParseCmdlineNoIgnitionFlags(t *testing.T) {
	cmdlinePath := writeCmdline(t, "console=tty0 root=/dev/sda1 ro quiet")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Url != nil || opts.DeviceLabel != "" || opts.UserDataPath != "" {
		t.Errorf("expected all opts empty for unrelated cmdline, got %+v", opts)
	}
}

func TestParseCmdlineInvalidURL(t *testing.T) {
	// A URL with a control character is unparseable; the flag should be skipped.
	cmdlinePath := writeCmdline(t, "ignition.config.url=://bad url\x00")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Url != nil {
		t.Errorf("expected nil URL for invalid URL, got %v", opts.Url)
	}
}

func TestParseCmdlineURLTakesPrecedence(t *testing.T) {
	// If both url and device/path are set, url wins (checked in fetchConfig).
	// parseCmdline should populate all provided fields.
	cmdlinePath := writeCmdline(t, "ignition.config.url=https://example.com/config.ign ignition.config.device=CONFIG ignition.config.path=/config.ign")
	logger := newTestLogger(t)
	opts, err := parseCmdline(logger, cmdlinePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Url == nil {
		t.Error("expected URL to be set")
	}
	if opts.DeviceLabel != "CONFIG" {
		t.Errorf("expected DeviceLabel %q, got %q", "CONFIG", opts.DeviceLabel)
	}
	if opts.UserDataPath != "/config.ign" {
		t.Errorf("expected UserDataPath %q, got %q", "/config.ign", opts.UserDataPath)
	}
}

func TestValidateDeviceLabel(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{name: "simple label", label: "CONFIG", wantErr: false},
		{name: "dashes and dots", label: "config-2.1", wantErr: false},
		{name: "dotdot embedded in name is fine", label: "foo..bar", wantErr: false},
		{name: "contains slash", label: "foo/bar", wantErr: true},
		{name: "bare dotdot", label: "..", wantErr: true},
		{name: "bare dot", label: ".", wantErr: true},
		{name: "slash prefix", label: "/etc/passwd", wantErr: true},
		{name: "dotdot slash traversal", label: "../etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeviceLabel(tt.label)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for label %q", tt.label)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for label %q: %v", tt.label, err)
			}
		})
	}
}

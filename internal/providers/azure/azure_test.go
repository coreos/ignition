package azure

import (
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
)

func TestParseProvisioningConfig(t *testing.T) {
	raw := []byte(`
<wa:ProvisioningSection xmlns:wa="http://schemas.microsoft.com/windowsazure">
  <LinuxProvisioningConfigurationSet>
    <HostName>myhost</HostName>
    <UserName>azureuser</UserName>
    <UserPassword>password</UserPassword>
    <DisableSshPasswordAuthentication>false</DisableSshPasswordAuthentication>
    <SSH>
      <PublicKeys>
        <PublicKey>
          <Value>ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCu</Value>
        </PublicKey>
      </PublicKeys>
    </SSH>
  </LinuxProvisioningConfigurationSet>
</wa:ProvisioningSection>`)

	cfg, err := parseProvisioningConfig(raw)
	if err != nil {
		t.Fatalf("parseProvisioningConfig() err = %v", err)
	}
	if cfg.UserName != "azureuser" {
		t.Fatalf("expected username azureuser, got %s", cfg.UserName)
	}
	if len(cfg.SSH.PublicKeys) != 1 {
		t.Fatalf("expected 1 ssh key, got %d", len(cfg.SSH.PublicKeys))
	}
}

func TestBuildGeneratedConfig(t *testing.T) {
	meta := &instanceMetadata{
		Compute: instanceComputeMetadata{
			Hostname: "example",
			OSProfile: instanceOSProfile{
				AdminUsername: "meta-user",
			},
			PublicKeys: []instancePublicKey{
				{KeyData: "ssh-rsa AAAAB3Nza meta"},
			},
		},
	}
	prov := &linuxProvisioningConfigurationSet{
		UserName: "prov-user",
		SSH: sshSection{
			PublicKeys: []sshPublicKey{
				{Value: "ssh-ed25519 AAAAC3Nza prov"},
			},
		},
		UserPassword: "plaintext",
	}

	cfg, err := buildGeneratedConfig(meta, prov)
	if err != nil {
		t.Fatalf("buildGeneratedConfig() err = %v", err)
	}

	if len(cfg.Passwd.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(cfg.Passwd.Users))
	}
	user := cfg.Passwd.Users[0]
	if user.Name != "meta-user" {
		t.Fatalf("expected user meta-user, got %s", user.Name)
	}
	if len(user.SSHAuthorizedKeys) != 2 {
		t.Fatalf("expected 2 ssh keys, got %d", len(user.SSHAuthorizedKeys))
	}
	// Password should be hashed (starts with $6$ for SHA-512)
	if user.PasswordHash == nil {
		t.Fatalf("expected password hash to be set")
	}
	if !strings.HasPrefix(*user.PasswordHash, "$6$") {
		t.Fatalf("expected password hash to be SHA-512 (start with $6$), got %s", *user.PasswordHash)
	}

	if len(cfg.Storage.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(cfg.Storage.Files))
	}
}

func TestBuildGeneratedConfigWithPrehashedPassword(t *testing.T) {
	// Test that pre-hashed passwords are not double-hashed
	prehashedPassword := "$6$rounds=5000$saltsalt$hashedvalue"
	meta := &instanceMetadata{
		Compute: instanceComputeMetadata{
			OSProfile: instanceOSProfile{
				AdminUsername: "testuser",
			},
		},
	}
	prov := &linuxProvisioningConfigurationSet{
		UserPassword: prehashedPassword,
	}

	cfg, err := buildGeneratedConfig(meta, prov)
	if err != nil {
		t.Fatalf("buildGeneratedConfig() err = %v", err)
	}

	user := cfg.Passwd.Users[0]
	if user.PasswordHash == nil || *user.PasswordHash != prehashedPassword {
		t.Fatalf("expected pre-hashed password to be preserved, got %v", user.PasswordHash)
	}
}

func TestCollectSSHPublicKeysDedup(t *testing.T) {
	meta := &instanceMetadata{
		Compute: instanceComputeMetadata{
			PublicKeys: []instancePublicKey{
				{KeyData: "ssh-rsa AAAA"},
				{KeyData: "ssh-rsa AAAA"},
			},
		},
	}
	prov := &linuxProvisioningConfigurationSet{
		SSH: sshSection{
			PublicKeys: []sshPublicKey{
				{Value: "ssh-rsa BBBB"},
				{Value: "ssh-rsa AAAA"},
			},
		},
	}
	keys := collectSSHPublicKeys(meta, prov)
	if len(keys) != 2 {
		t.Fatalf("expected 2 unique keys, got %d", len(keys))
	}
}

func TestPasswordAuthDisabledParsing(t *testing.T) {
	trueCases := []string{"true", "TRUE", "1", " yes ", "YES"}
	for _, tc := range trueCases {
		prov := linuxProvisioningConfigurationSet{DisableSshPasswordAuthentication: tc}
		if !prov.passwordAuthDisabled() {
			t.Fatalf("expected %q to disable password auth", tc)
		}
	}
	falseCases := []string{"false", "0", "no", "", "NO", "False"}
	for _, tc := range falseCases {
		prov := linuxProvisioningConfigurationSet{DisableSshPasswordAuthentication: tc}
		if prov.passwordAuthDisabled() {
			t.Fatalf("expected %q to allow password auth", tc)
		}
	}
}

func TestNewDataFile(t *testing.T) {
	content := "line1\n"
	file := newDataFile("/tmp/example", 0640, content)
	if file.Path != "/tmp/example" {
		t.Fatalf("unexpected path %s", file.Path)
	}
	if file.Mode == nil || *file.Mode != 0640 {
		t.Fatalf("unexpected mode %#v", file.Mode)
	}
	if file.Contents.Source == nil || !strings.Contains(*file.Contents.Source, content) {
		t.Fatalf("expected contents to include original data")
	}
}

func TestBuildGeneratedConfigErrors(t *testing.T) {
	meta := &instanceMetadata{}
	prov := &linuxProvisioningConfigurationSet{}
	if _, err := buildGeneratedConfig(meta, prov); err == nil {
		t.Fatalf("expected error when username missing")
	}
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() err = %v", err)
	}

	// Verify hash format
	if !strings.HasPrefix(hash, "$6$") {
		t.Fatalf("expected SHA-512 hash prefix $6$, got %s", hash)
	}

	// Verify hash has expected structure: $6$<salt>$<hash>
	parts := strings.Split(hash, "$")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts in hash, got %d: %s", len(parts), hash)
	}
	if parts[1] != "6" {
		t.Fatalf("expected algorithm identifier '6', got %s", parts[1])
	}
	if len(parts[2]) != 16 {
		t.Fatalf("expected 16 character salt, got %d: %s", len(parts[2]), parts[2])
	}
	if len(parts[3]) != 86 {
		t.Fatalf("expected 86 character hash, got %d: %s", len(parts[3]), parts[3])
	}
}

func TestHashPasswordDifferentSalts(t *testing.T) {
	password := "testpassword123"
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() err = %v", err)
	}
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() err = %v", err)
	}

	// Hashes should be different due to random salt
	if hash1 == hash2 {
		t.Fatalf("expected different hashes for same password (different salts)")
	}
}

func TestIsPasswordHashed(t *testing.T) {
	tests := []struct {
		password string
		expected bool
	}{
		{"$6$salt$hash", true},
		{"$5$salt$hash", true},
		{"$y$salt$hash", true},
		{"$2a$10$hash", true},
		{"$2b$10$hash", true},
		{"$2y$10$hash", true},
		{"$1$salt$hash", true},
		{"plaintext", false},
		{"$invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsPasswordHashed(tt.password)
		if result != tt.expected {
			t.Errorf("IsPasswordHashed(%q) = %v, expected %v", tt.password, result, tt.expected)
		}
	}
}

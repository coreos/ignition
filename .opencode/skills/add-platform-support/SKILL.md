---
name: add-platform-support
description: Automate adding new cloud provider/platform support to Ignition
---

# Add Platform Support

This skill automates adding support for a new cloud provider or platform to Ignition, following the exact pattern from commits like [ef142f33](https://github.com/coreos/ignition/commit/ef142f33) (Hetzner) and [9b833b21](https://github.com/coreos/ignition/commit/9b833b21) (Akamai).

## What it does

Performs a complete platform addition by:

1. Creating provider implementation code in `internal/providers/{provider}/{provider}.go`
2. Registering the provider in `internal/register/providers.go`
3. Documenting the platform in `docs/supported-platforms.md`
4. Adding a release note entry in `docs/release-notes.md`
5. Running validation tests to ensure correctness
6. Creating a properly formatted commit

## Prerequisites

None - the skill handles everything automatically.

## Usage

### Interactive Mode (Recommended)

```bash
/add-platform-support
```

The skill will prompt you for:
- Platform ID (e.g., "hetzner")
- Platform name (e.g., "Hetzner Cloud")
- Metadata URL (e.g., "http://169.254.169.254/hetzner/v1/userdata")
- Documentation URL (e.g., "https://www.hetzner.com/cloud")
- Optional: Custom description

### Direct Mode

```bash
/add-platform-support --id hetzner --name "Hetzner Cloud" --url "http://169.254.169.254/hetzner/v1/userdata" --docs "https://www.hetzner.com/cloud"
```

## Step-by-Step Workflow

When invoked, follow these steps in order:

### Step 1: Gather Required Information

If not provided as arguments, ask the user for:

**Required:**
- `provider_id`: Platform ID (lowercase, alphanumeric, e.g., "hetzner")
- `provider_name`: Display name (e.g., "Hetzner Cloud")
- `metadata_url`: Full URL to fetch config from
- `provider_url`: Documentation URL for the platform

**Optional:**
- `description`: Custom description (default: "Ignition will read its configuration from the instance userdata. Cloud SSH keys are handled separately.")

**Validation:**
- Provider ID must be lowercase, alphanumeric (no spaces, no special chars except hyphen)
- Provider ID must not already exist in `internal/providers/`
- URLs must be well-formed

### Step 2: Determine Current Spec Version

Find the latest experimental spec version:

```bash
# List config spec directories
ls -d internal/config/v*_experimental/ | sort -V | tail -1
```

Extract the version (e.g., `v3_6_experimental`) to use in import paths.

### Step 3: Parse Metadata URL

Parse the metadata URL to extract components:

```
http://169.254.169.254/hetzner/v1/userdata
→ Scheme: "http"
→ Host: "169.254.169.254"
→ Path: "hetzner/v1/userdata" (with leading slash)
```

### Step 4: Generate Provider Code

Create `internal/providers/{provider_id}/{provider_id}.go` with this content:

```go
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

// The {provider_id} provider fetches a remote configuration from the {provider_name}
// user-data metadata service URL.

package {provider_id}

import (
	"net/url"

	"github.com/coreos/ignition/v2/config/{spec_version}/types"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataURL = url.URL{
		Scheme: "{scheme}",
		Host:   "{host}",
		Path:   "{path}",
	}
)

func init() {
	platform.Register(platform.Provider{
		Name:  "{provider_id}",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{})

	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
```

**Substitutions:**
- `{provider_id}` → provider ID (e.g., "hetzner")
- `{provider_name}` → provider name (e.g., "Hetzner Cloud")
- `{spec_version}` → current spec version (e.g., "v3_6_experimental")
- `{scheme}` → URL scheme (e.g., "http")
- `{host}` → URL host (e.g., "169.254.169.254")
- `{path}` → URL path with leading slash (e.g., "/hetzner/v1/userdata")

**Important:**
- Preserve all tabs/spaces for indentation
- License header must be exactly 13 lines (enforced by test script)
- Package comment describes what the provider does

### Step 5: Update Provider Registry

Read `internal/register/providers.go` to find the import section.

Add the new provider import in **alphabetical order**:

```go
import (
	_ "github.com/coreos/ignition/v2/internal/providers/exoscale"
	_ "github.com/coreos/ignition/v2/internal/providers/file"
	_ "github.com/coreos/ignition/v2/internal/providers/gcp"
+	_ "github.com/coreos/ignition/v2/internal/providers/{provider_id}"
	_ "github.com/coreos/ignition/v2/internal/providers/hyperv"
	...
)
```

Use the Edit tool to insert the import in the correct alphabetical position.

### Step 6: Update Platform Documentation

Read `docs/supported-platforms.md` to find:
1. The main platform list (bullet points)
2. The URL reference section at the bottom

**Main List Addition:**
Insert in alphabetical order by display name:

```markdown
* [Google Cloud] (`gcp`) - Ignition will read its...
+ * [{provider_name}] (`{provider_id}`) - {description}
* [Microsoft Hyper-V] (`hyperv`) - Ignition will read its...
```

**URL Reference Addition:**
Insert in alphabetical order at bottom:

```markdown
[Google Cloud]: https://cloud.google.com/compute
+ [{provider_name}]: {provider_url}
[Microsoft Hyper-V]: https://learn.microsoft.com/...
```

Use the Edit tool twice (once for list, once for reference).

### Step 7: Update Release Notes

Read `docs/release-notes.md` to find the "Unreleased" section.

Add entry under "### Features":

```markdown
## Upcoming Ignition X.Y.Z (unreleased)

### Breaking changes

### Features

+ - Support {provider_name}

### Changes
```

Use the Edit tool to insert after the "### Features" line.

### Step 8: Validate Changes

Run the test script to verify:

```bash
./test
```

**Expected results:**
- ✅ License headers are valid (13 lines)
- ✅ Platform ID is documented in supported-platforms.md
- ✅ All tests pass

If validation fails, show the error output and do NOT proceed to commit.

### Step 9: Create Commit

If all validation passes, create a commit:

```bash
# Stage all modified files
git add internal/providers/{provider_id}/{provider_id}.go \
        internal/register/providers.go \
        docs/supported-platforms.md \
        docs/release-notes.md

# Create commit
git commit -m "$(cat <<'EOF'
providers/{provider_id}: add support for {provider_name}

{Optional: Additional details about the provider}

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

**Commit message format:**
- Title: `providers/{provider_id}: add support for {provider_name}`
- Body: Optional details
- Co-author: Claude

### Step 10: Provide Summary

Show the user what was done:

```
✨ Platform support added successfully!

Platform: {provider_name} ({provider_id})
Metadata URL: {metadata_url}

Files created/modified:
  ✅ internal/providers/{provider_id}/{provider_id}.go (new file, 54 lines)
  ✅ internal/register/providers.go (+1 import)
  ✅ docs/supported-platforms.md (+2 lines)
  ✅ docs/release-notes.md (+1 line)

Validation:
  ✅ ./test script passed
  ✅ License headers correct
  ✅ Platform documented

Commit created:
  providers/{provider_id}: add support for {provider_name}

Next steps:
1. Review changes: git show
2. Test the provider if you have access to the platform
3. Push to remote when ready: git push
4. Create a PR to main branch
```

## Example Execution

### Input:
```
Provider ID: hetzner
Provider Name: Hetzner Cloud
Metadata URL: http://169.254.169.254/hetzner/v1/userdata
Documentation URL: https://www.hetzner.com/cloud
Description: (use default)
```

### Output:
```
✨ Platform support added successfully!

Platform: Hetzner Cloud (hetzner)
Metadata URL: http://169.254.169.254/hetzner/v1/userdata

Files created/modified:
  ✅ internal/providers/hetzner/hetzner.go (new file, 54 lines)
  ✅ internal/register/providers.go (+1 import)
  ✅ docs/supported-platforms.md (+2 lines)
  ✅ docs/release-notes.md (+1 line)

Commit: providers/hetzner: add support for Hetzner Cloud
```

## Validation Details

The `./test` script automatically validates:

1. **License headers**: All .go files must have Apache 2.0 header (13 lines)
2. **Platform documentation**: Checks that every `platform.Register()` call has corresponding docs
3. **Code compilation**: Ensures Go code is valid

**From test script (lines 70-85):**
```bash
platforms=$(grep -A 1 -h platform.Register internal/providers/*/* | grep Name: | cut -f2 -d\")
for id in ${platforms}; do
    case "${id}" in
    file) ;;  # Undocumented platform ID for testing
    *)
        if ! grep -qF "\`${id}\`" docs/supported-platforms.md; then
            echo "Undocumented platform ID: $id" >&2
            exit 1
        fi
        ;;
    esac
done
```

## What's NOT Covered

This skill creates **simple providers** only (single HTTP GET from metadata URL).

For complex providers requiring:
- Authentication tokens (like Akamai)
- Base64 decoding
- Gzip decompression
- Multiple URLs (IPv4/IPv6)
- Special HTTP headers

You'll need to manually enhance the generated code after the skill runs.

## References

**Example Commits:**
- [ef142f33](https://github.com/coreos/ignition/commit/ef142f33) - Hetzner Cloud (simple provider)
- [9b833b21](https://github.com/coreos/ignition/commit/9b833b21) - Akamai (complex provider with auth)

**Documentation:**
- `docs/supported-platforms.md` - Platform list
- `test` script (lines 70-85) - Validation logic

**Existing Providers:** 35+ examples in `internal/providers/`

## Troubleshooting

**"Provider already exists"**
→ Check if provider is already in `internal/providers/{id}/`

**"Test validation failed"**
→ Check the error message, likely:
  - License header formatting issue
  - Platform not properly documented
  - Code syntax error

**"Alphabetical order incorrect"**
→ Verify the import/list/reference was inserted in correct alphabetical position

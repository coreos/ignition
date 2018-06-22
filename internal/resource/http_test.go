package resource

import (
	"os"
	"strings"
	"testing"
)

func TestGettingProxyFromEnv(t *testing.T) {

	env := []string{
		"HTTPS_PROXY",
		"HTTP_PROXY",
		"HTTP_NOPROXY",
	}

	notSetCount := 0

	for _, e := range env {

		eLowercase := strings.ToLower(e)

		//check for upper and lower case
		if os.Getenv(e) == "" {
			if os.Getenv(eLowercase) == "" {
				// t.Logf("%s and %s are unset.", e, eLowercase)
				notSetCount++
			} else {
				t.Logf("%s=%s", eLowercase, os.Getenv(eLowercase))
			}
		} else {
			t.Logf("%s=%s", e, os.Getenv(e))
		}

	}

	if notSetCount == 3 {
		// none were set
		t.Errorf("None of these %q or their lowercase equivalent were set.", env)
	}

}

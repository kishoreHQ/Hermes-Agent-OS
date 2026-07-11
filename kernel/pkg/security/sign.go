package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Signing env:
//   HERMES_PLUGIN_HMAC_KEY  — shared secret for HMAC-SHA256 manifests
//   HERMES_REQUIRE_SIGNED_PLUGINS=1 — reject unsigned/invalid on verify path

// CanonicalBytes is the stable signing payload for a manifest identity.
func CanonicalBytes(apiVersion, kind, id, version string) []byte {
	s := strings.Join([]string{apiVersion, kind, id, version}, "|")
	return []byte(s)
}

// SignHMAC returns hex HMAC-SHA256 of the canonical bytes.
func SignHMAC(apiVersion, kind, id, version string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(CanonicalBytes(apiVersion, kind, id, version))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC checks signature against key.
func VerifyHMAC(apiVersion, kind, id, version, signature string, key []byte) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}
	want := SignHMAC(apiVersion, kind, id, version, key)
	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(want))) {
		return fmt.Errorf("invalid plugin signature for %s", id)
	}
	return nil
}

// KeyFromEnv returns HMAC key from environment (empty if unset).
func KeyFromEnv() []byte {
	k := os.Getenv("HERMES_PLUGIN_HMAC_KEY")
	if k == "" {
		return nil
	}
	return []byte(k)
}

// RequireSignedFromEnv is true when HERMES_REQUIRE_SIGNED_PLUGINS is truthy.
func RequireSignedFromEnv() bool {
	v := strings.ToLower(os.Getenv("HERMES_REQUIRE_SIGNED_PLUGINS"))
	return v == "1" || v == "true" || v == "yes"
}

// VerifyPluginIdentity applies env policy for plugin load.
func VerifyPluginIdentity(apiVersion, kind, id, version, signature string) error {
	key := KeyFromEnv()
	require := RequireSignedFromEnv()
	if require {
		if len(key) == 0 {
			return fmt.Errorf("HERMES_REQUIRE_SIGNED_PLUGINS set but HERMES_PLUGIN_HMAC_KEY empty")
		}
		return VerifyHMAC(apiVersion, kind, id, version, signature, key)
	}
	if signature != "" && len(key) > 0 {
		return VerifyHMAC(apiVersion, kind, id, version, signature, key)
	}
	return nil
}

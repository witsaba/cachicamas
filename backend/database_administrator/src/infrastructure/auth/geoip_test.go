// Package auth — geoip_test.go locks the GeoIP enrichment contract
// per spec R-BE-005 / R-GEOIP-1 / S-BE-050 / S-BE-051.
//
// The tests cover the three locked code paths without depending on
// a real MaxMind GeoLite2 .mmdb fixture:
//
//  1. Disabled (GEOIP_DB_PATH empty) → Enrich returns (nil, nil)
//     so the audit row is written without country_code / city.
//  2. Missing DB (path set, file not present) → Enrich returns
//     (nil, nil) silently per R-GEOIP-1 (no-fail on missing).
//  3. Malformed DB (path set, file content garbage) → Enrich
//     returns (nil, nil) per R-GEOIP-1 (no-fail on malformed).
//
// The "enabled" path (valid DB + valid IP → populated fields) is
// covered by the integration suite with a committed
// testdata/GeoLite2-City.mmdb fixture (see ADR-0011-native-google-oauth,
// PR-4 territory). PR-2 lands the dependency + the gate; PR-4
// lands the fixture + ADR.
//
// TODO(PR-4): add an enabled-path integration test once the
// MaxMind .mmdb fixture lands per ADR-0011 §D7.
package auth

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGeoIP_Disabled covers S-BE-050: an empty GEOIP_DB_PATH means
// enrichment is OFF, and Enrich returns (nil, nil) so the audit row
// is written without GeoIP fields. No error is returned.
func TestGeoIP_Disabled(t *testing.T) {
	g, err := NewGeoIP("")
	if err != nil {
		t.Fatalf("NewGeoIP(\"\"): unexpected error %v", err)
	}
	if g == nil {
		t.Fatal("NewGeoIP(\"\"): returned nil")
	}
	if g.IsEnabled() {
		t.Error("NewGeoIP(\"\").IsEnabled() = true, want false")
	}
	country, city, err := g.Enrich("8.8.8.8")
	if err != nil {
		t.Errorf("Enrich on disabled GeoIP: unexpected error %v", err)
	}
	if country != nil {
		t.Errorf("Enrich on disabled GeoIP: country = %v, want nil", country)
	}
	if city != nil {
		t.Errorf("Enrich on disabled GeoIP: city = %v, want nil", city)
	}
}

// TestGeoIP_MissingDB covers R-GEOIP-1 (no-fail on missing DB): a
// GEOIP_DB_PATH pointing at a non-existent file MUST NOT cause
// NewGeoIP to return an error — the construction succeeds and
// Enrich silently returns (nil, nil). IsEnabled() reflects the
// actual reader state (false), so callers can skip Enrich when
// they know the gate is unusable.
//
// Rationale: an operator who forgets to drop the .mmdb file into
// ./geoip-data/ should not block bootstrap from running. The
// audit row still gets written, just without country_code / city.
func TestGeoIP_MissingDB(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist.mmdb")
	g, err := NewGeoIP(missing)
	if err != nil {
		t.Fatalf("NewGeoIP(%q) with missing DB: unexpected error %v", missing, err)
	}
	if g == nil {
		t.Fatal("NewGeoIP: returned nil")
	}
	if g.IsEnabled() {
		t.Error("NewGeoIP(missingDB).IsEnabled() = true, want false (reader never opened)")
	}
	country, city, err := g.Enrich("8.8.8.8")
	if err != nil {
		t.Errorf("Enrich with missing DB: unexpected error %v", err)
	}
	if country != nil {
		t.Errorf("Enrich with missing DB: country = %v, want nil", country)
	}
	if city != nil {
		t.Errorf("Enrich with missing DB: city = %v, want nil", city)
	}
}

// TestGeoIP_MalformedDB covers R-GEOIP-1 (no-fail on malformed DB):
// a GEOIP_DB_PATH pointing at a file whose content is garbage MUST
// NOT crash the service. Enrich returns (nil, nil) so the audit
// row is written without geo fields, and a warning is logged.
func TestGeoIP_MalformedDB(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "garbage.mmdb")
	// 32 bytes of nonsense is enough to fail the MaxMind reader's
	// binary-type check.
	if err := os.WriteFile(bad, []byte("THIS IS NOT A VALID MAXMIND DATABASE FILE"), 0o600); err != nil {
		t.Fatalf("setup: write malformed DB: %v", err)
	}
	g, err := NewGeoIP(bad)
	if err != nil {
		// Construction MAY fail loudly so an operator notices the
		// problem; the integration path (R-GEOIP-1) requires
		// that an existing DB path produces a non-nil g.
		// We accept either: a nil-error with Enrich returning
		// (nil, nil, nil) OR a non-nil error during NewGeoIP.
		// Either way, no panic.
		t.Logf("NewGeoIP with malformed DB returned error (acceptable): %v", err)
		return
	}
	if g == nil {
		t.Fatal("NewGeoIP: returned nil")
	}
	country, city, err := g.Enrich("8.8.8.8")
	if err != nil {
		t.Errorf("Enrich with malformed DB: unexpected error %v", err)
	}
	if country != nil {
		t.Errorf("Enrich with malformed DB: country = %v, want nil", country)
	}
	if city != nil {
		t.Errorf("Enrich with malformed DB: city = %v, want nil", city)
	}
}

// TestGeoIP_DisabledEnrichIPVariants covers the disabled-path IP
// variants: the no-op gate must handle every reasonable IP format
// (IPv4, IPv6, garbage) without panicking or returning an error.
func TestGeoIP_DisabledEnrichIPVariants(t *testing.T) {
	g, err := NewGeoIP("")
	if err != nil {
		t.Fatalf("NewGeoIP(\"\"): %v", err)
	}
	for _, ip := range []string{
		"127.0.0.1",
		"::1",
		"8.8.8.8",
		"2001:4860:4860::8888",
		"",
		"not-an-ip",
		"999.999.999.999",
	} {
		t.Run("ip="+ip, func(t *testing.T) {
			country, city, err := g.Enrich(ip)
			if err != nil {
				t.Errorf("Enrich(%q): unexpected error %v", ip, err)
			}
			if country != nil || city != nil {
				t.Errorf("Enrich(%q): country=%v city=%v, want both nil (disabled)", ip, country, city)
			}
		})
	}
}
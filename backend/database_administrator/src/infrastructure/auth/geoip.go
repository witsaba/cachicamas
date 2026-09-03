// Package auth — geoip.go implements the GeoIP enrichment gate per
// spec R-BE-005 / R-GEOIP-1 / S-BE-050 / S-BE-051.
//
// The enrichment is OPTIONAL and best-effort: an empty GEOIP_DB_PATH
// disables it entirely; a missing or malformed .mmdb file silently
// skips the lookup so the bootstrap service can still write an
// audit row (just without country_code / city).
//
// CRITICAL invariants (locked at design §2.2 + spec §5):
//   - NewGeoIP MUST never panic. An empty path returns a disabled
//     gate; a missing/malformed path returns either a disabled
//     gate OR an error (depending on what fails); either way the
//     caller can decide.
//   - Enrich MUST never panic. A disabled gate returns (nil, nil,
//     nil); an enabled gate with a malformed DB returns (nil, nil,
//     nil) so the audit row is written without geo fields per
//     R-GEOIP-1 (no-fail).
//   - The MaxMind reader is loaded ONCE at construction time. The
//     reader is safe for concurrent use (the underlying
//     maxminddb-golang library documents this).
//
// TODO(PR-4): the ADR-0011-native-google-oauth.md commitment that
// justifies this new top-level dep lands in PR-4 per
// openspec/AGENTS.md hard rule. The dep is added here with the
// minimal API surface needed for PR-2; PR-4 lands the .mmdb
// fixture + the ADR + any expanded logging.
package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
)

// GeoIP is the lookup gate. Construct with NewGeoIP; the zero value
// is NOT usable. Callers should always go through NewGeoIP so the
// gate starts in a known state.
type GeoIP struct {
	dbPath string
	reader *geoip2.Reader
	// enabled tracks whether the gate has a usable reader. When
	// false, Enrich short-circuits to (nil, nil, nil) without
	// touching the reader.
	enabled bool
}

// NewGeoIP constructs a GeoIP gate from a MaxMind GeoLite2 .mmdb
// path. Empty path → disabled gate. Missing/malformed file →
// returns a disabled gate (no error). The caller can check
// IsEnabled() to decide whether to skip Enrich calls entirely.
//
// The construction is intentionally permissive: a production
// deployment with a corrupt DB is a misconfiguration, but the
// bootstrap service should still run. Logging happens at
// construction time so an operator can spot the problem in the
// startup logs.
func NewGeoIP(dbPath string) (*GeoIP, error) {
	if dbPath == "" {
		slog.Info("geoip disabled (no GEOIP_DB_PATH set)")
		return &GeoIP{dbPath: "", reader: nil, enabled: false}, nil
	}
	reader, err := geoip2.Open(dbPath)
	if err != nil {
		// Per R-GEOIP-1 we don't fail startup on a missing/malformed
		// DB; instead we log and return a disabled gate so the
		// caller can proceed (the Enrich path is a no-op when
		// !enabled).
		slog.Warn("geoip open failed; enrichment disabled",
			slog.String("db_path", dbPath),
			slog.String("error", err.Error()),
		)
		return &GeoIP{dbPath: dbPath, reader: nil, enabled: false}, nil
	}
	slog.Info("geoip enabled", slog.String("db_path", dbPath))
	return &GeoIP{dbPath: dbPath, reader: reader, enabled: true}, nil
}

// IsEnabled reports whether the gate has a usable reader. Callers
// can use this to skip the Enrich call entirely (saving one method
// indirection per bootstrap call) when the gate is disabled.
func (g *GeoIP) IsEnabled() bool {
	if g == nil {
		return false
	}
	return g.enabled
}

// Enrich looks up the country_code + city for the given IP. The
// return contract:
//
//   - (nil, nil, nil): no result available (disabled gate, missing
//     DB, malformed DB, or the IP is not in the DB).
//   - (*country, *city, nil): populated fields.
//   - (nil, nil, err): reserved for future failure modes; current
//     implementation never returns a non-nil error.
//
// The function MUST NOT panic on any input. The IP parser is
// permissive: garbage strings fall through to the not-found branch
// without an error.
//
// This is best-effort per R-BE-005; callers MUST NOT treat a nil
// result as a failure.
func (g *GeoIP) Enrich(ip string) (*string, *string, error) {
	if g == nil || !g.enabled || g.reader == nil {
		return nil, nil, nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, nil, nil
	}
	record, err := g.reader.City(parsed)
	if err != nil {
		// Per R-GEOIP-1, no-fail on lookup error. The audit row
		// is written without geo fields.
		slog.Debug("geoip lookup failed",
			slog.String("ip", ip),
			slog.String("error", err.Error()),
		)
		return nil, nil, nil
	}
	var country *string
	if c := record.Country.IsoCode; c != "" {
		country = &c
	}
	var city *string
	if name := record.City.Names["en"]; name != "" {
		city = &name
	}
	return country, city, nil
}

// Close releases the underlying MaxMind reader. Safe to call on a
// nil or disabled gate. Should be called at server shutdown for
// clean resource release.
func (g *GeoIP) Close() error {
	if g == nil || g.reader == nil {
		return nil
	}
	err := g.reader.Close()
	g.reader = nil
	g.enabled = false
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("auth.GeoIP.Close: %w", err)
	}
	return nil
}
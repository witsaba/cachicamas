package openaicompat

import "net/url"

// joinRequestPath joins base with relative path segments and returns a new
// URL. Empty segments are filtered out before delegating to
// (*url.URL).JoinPath, which pins the no-dropped-segment,
// no-doubled-separator contract deterministically (R-APC-006): RFC 3986
// reference-merging would otherwise treat a non-trailing-slash base's last
// path segment as a document to be replaced, silently truncating a
// gateway sub-path such as /proxy/openai.
//
// base is never mutated: JoinPath returns a new *url.URL, so one stored
// base is safe to reuse across any number of requests (S-APC-029).
func joinRequestPath(base *url.URL, segments ...string) *url.URL {
	filtered := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		filtered = append(filtered, segment)
	}
	return base.JoinPath(filtered...)
}

package utils

import "strings"

// IsExternalAssetRef reports whether an asset reference points somewhere the
// build must not touch: another origin (https://…, //cdn…), an inline data
// URI, or a site-root-absolute path that the deployment supplies itself.
// Anything else is a path inside the project, to be resolved and copied.
func IsExternalAssetRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if strings.HasPrefix(ref, "/") {
		return true
	}
	lower := strings.ToLower(ref)
	for _, prefix := range []string{"//", "http://", "https://", "data:"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

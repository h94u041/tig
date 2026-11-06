package tig

import (
	"path"
	"strings"
)

// A snapshot keeps every file it lists, forever, and .tig goes into Git with the
// project. So a key that nobody would ever commit must not reach it by being in
// the folder. Git's own word counts: a file Git tracks is already in its history,
// and it stays listed. Everything else that looks like a secret is left out.

var secretNames = map[string]bool{
	".netrc": true, "_netrc": true, ".pgpass": true, ".npmrc": true, ".pypirc": true,
	".git-credentials": true, ".htpasswd": true,
	"credentials": true, "credentials.json": true, "secrets.json": true, "secrets.yaml": true, "secrets.yml": true,
	"id_rsa": true, "id_dsa": true, "id_ecdsa": true, "id_ed25519": true,
}

var secretExtensions = map[string]bool{
	".pem": true, ".key": true, ".p12": true, ".pfx": true, ".jks": true, ".keystore": true, ".ppk": true,
}

var secretFolders = map[string]bool{".ssh": true, ".aws": true, ".gnupg": true}

// LooksSecret reports whether a project path looks like it holds a key or a
// credential. It goes by the name alone, so it can be wrong both ways: it is a
// guard against the usual accident, not a scan.
func LooksSecret(rel string) bool {
	parts := strings.Split(strings.ToLower(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if secretFolders[p] {
			return true
		}
	}
	base := parts[len(parts)-1]
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		// The ones written to be shared say so.
		switch strings.TrimPrefix(base, ".env.") {
		case "example", "sample", "template", "dist":
			return false
		}
		return true
	}
	return secretNames[base] || secretExtensions[path.Ext(base)]
}

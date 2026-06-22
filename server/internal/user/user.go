// Package user modélise les comptes du BBS et leur persistance.
//
// Les mots de passe ne sont jamais stockés en clair : ils sont hachés en
// PBKDF2-HMAC-SHA256 (stdlib `crypto/pbkdf2`, Go 1.24+) avec un sel aléatoire
// par compte. Le format encodé est auto-descriptif :
//
//	pbkdf2$sha256$<iterations>$<sel_base64>$<hash_base64>
//
// Voir docs/adr/0001-login-composant-page.md.
package user

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Paramètres de hachage. iterCount peut augmenter avec le temps ; le format
// encodé porte sa propre valeur, donc les anciens hachages restent vérifiables.
const (
	iterCount = 100_000 // itérations PBKDF2
	saltLen   = 16      // octets de sel
	keyLen    = 32      // octets de clé dérivée (SHA-256)
)

// Contraintes sur le pseudo (handle), adaptées à l'écran Oric 40 colonnes.
const (
	MinHandleLen   = 2
	MaxHandleLen   = 16
	MinPasswordLen = 4
)

// User est un compte du BBS.
type User struct {
	Handle    string    `json:"handle"`    // pseudo affiché (casse d'origine)
	PassHash  string    `json:"passHash"`  // mot de passe haché (jamais en clair)
	Created   time.Time `json:"created"`   // date de création
	LastLogin time.Time `json:"lastLogin"` // dernière connexion réussie
	Calls     int       `json:"calls"`     // nombre d'appels (connexions réussies)
}

// NormalizeHandle renvoie la forme canonique d'un pseudo, utilisée comme clé
// d'unicité (insensible à la casse, espaces de bord retirés).
func NormalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

// ValidateHandle vérifie qu'un pseudo est acceptable : longueur bornée et
// caractères alphanumériques ASCII (plus '-' et '_'). Renvoie le pseudo nettoyé
// (espaces de bord retirés) ou une erreur explicite.
func ValidateHandle(handle string) (string, error) {
	h := strings.TrimSpace(handle)
	if n := len([]rune(h)); n < MinHandleLen || n > MaxHandleLen {
		return "", fmt.Errorf("le pseudo doit faire entre %d et %d caracteres", MinHandleLen, MaxHandleLen)
	}
	for _, r := range h {
		if r > unicode.MaxASCII || (!isAlnum(r) && r != '-' && r != '_') {
			return "", fmt.Errorf("le pseudo n'accepte que lettres, chiffres, '-' et '_'")
		}
	}
	return h, nil
}

// ValidatePassword vérifie la longueur minimale du mot de passe.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return fmt.Errorf("le mot de passe doit faire au moins %d caracteres", MinPasswordLen)
	}
	return nil
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// HashPassword dérive le mot de passe avec un sel aléatoire et renvoie la chaîne
// encodée auto-descriptive.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("tirage du sel : %w", err)
	}
	dk, err := pbkdf2.Key(sha256.New, password, salt, iterCount, keyLen)
	if err != nil {
		return "", fmt.Errorf("derivation PBKDF2 : %w", err)
	}
	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s",
		iterCount,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk),
	), nil
}

// VerifyPassword compare un mot de passe en clair au hachage encodé.
// La comparaison finale est à temps constant (anti timing-attack).
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[2])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

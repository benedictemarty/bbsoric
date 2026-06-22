package user

import (
	"strings"
	"testing"
)

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := HashPassword("s3cr3t")
	if err != nil {
		t.Fatalf("HashPassword : %v", err)
	}
	if !strings.HasPrefix(h, "pbkdf2$sha256$") {
		t.Errorf("format encode inattendu : %q", h)
	}
	if strings.Contains(h, "s3cr3t") {
		t.Errorf("le mot de passe en clair ne doit pas apparaitre dans le hachage")
	}
	if !VerifyPassword(h, "s3cr3t") {
		t.Errorf("le bon mot de passe doit etre accepte")
	}
	if VerifyPassword(h, "mauvais") {
		t.Errorf("un mauvais mot de passe doit etre rejete")
	}
}

func TestHashSaltIsRandom(t *testing.T) {
	h1, _ := HashPassword("identique")
	h2, _ := HashPassword("identique")
	if h1 == h2 {
		t.Errorf("deux hachages du meme mot de passe doivent differer (sel aleatoire)")
	}
	if !VerifyPassword(h1, "identique") || !VerifyPassword(h2, "identique") {
		t.Errorf("les deux hachages doivent valider le mot de passe d'origine")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, bad := range []string{
		"", "plain", "pbkdf2$sha256$", "pbkdf2$md5$1$x$y",
		"pbkdf2$sha256$abc$AAAA$BBBB", "pbkdf2$sha256$1$@@@$BBBB",
	} {
		if VerifyPassword(bad, "x") {
			t.Errorf("hachage malforme accepte a tort : %q", bad)
		}
	}
}

func TestValidateHandle(t *testing.T) {
	ok := []string{"ab", "Oric_User", "retro-42", strings.Repeat("a", MaxHandleLen)}
	for _, h := range ok {
		if _, err := ValidateHandle(h); err != nil {
			t.Errorf("%q devrait etre valide : %v", h, err)
		}
	}
	bad := []string{"a", strings.Repeat("a", MaxHandleLen+1), "a b", "épée", "user!", ""}
	for _, h := range bad {
		if _, err := ValidateHandle(h); err == nil {
			t.Errorf("%q devrait etre invalide", h)
		}
	}
}

func TestValidateHandleTrims(t *testing.T) {
	got, err := ValidateHandle("  bob  ")
	if err != nil || got != "bob" {
		t.Errorf("ValidateHandle devrait retirer les espaces de bord, got %q err %v", got, err)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("123"); err == nil {
		t.Errorf("un mot de passe trop court doit etre rejete")
	}
	if err := ValidatePassword("1234"); err != nil {
		t.Errorf("un mot de passe de longueur minimale doit passer : %v", err)
	}
}

func TestNormalizeHandle(t *testing.T) {
	if NormalizeHandle("  BoB ") != "bob" {
		t.Errorf("normalisation incorrecte")
	}
}

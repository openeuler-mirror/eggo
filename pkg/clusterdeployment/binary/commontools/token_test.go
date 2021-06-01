package commontools

import (
	"regexp"
	"testing"
)

func TestGenerateBootstrapToken(t *testing.T) {
	idPattern := `[a-z0-9]{6}`
	secretPattern := `[a-z0-9]{16}`
	tokenPattern := `\A([a-z0-9]{6})\.([a-z0-9]{16})\z`
	token, id, secret, err := ParseBootstrapTokenStr("")
	if err != nil {
		t.Fatalf("run GenerateBootstrapToken failed: %v", err)
	}
	if ok, _ := regexp.Match(idPattern, []byte(id)); !ok {
		t.Fatalf("invalid token id: %s", id)
	}
	if ok, _ := regexp.Match(secretPattern, []byte(secret)); !ok {
		t.Fatalf("invalid token secret: %s", secret)
	}
	if ok, _ := regexp.Match(tokenPattern, []byte(token)); !ok {
		t.Fatalf("invalid token: %s", token)
	}
}

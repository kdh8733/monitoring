package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "v0:%s:%s", ts, body)
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySlackSignature_Valid(t *testing.T) {
	secret := "shhh"
	now := time.Unix(1_700_000_000, 0)
	ts := "1700000000"
	body := []byte("payload=%7B%7D")
	sig := sign(secret, ts, body)

	if err := VerifySlackSignature(secret, ts, body, sig, now); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifySlackSignature_BadSignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	err := VerifySlackSignature("shhh", "1700000000", []byte("x"), "v0=deadbeef", now)
	if err == nil {
		t.Fatal("expected signature mismatch error")
	}
}

func TestVerifySlackSignature_Stale(t *testing.T) {
	secret := "shhh"
	ts := "1700000000"
	body := []byte("x")
	sig := sign(secret, ts, body)
	// now is 10 minutes after the timestamp -> outside the 5m window
	now := time.Unix(1_700_000_600, 0)
	if err := VerifySlackSignature(secret, ts, body, sig, now); err == nil {
		t.Fatal("expected stale timestamp rejection")
	}
}

func TestVerifySlackSignature_NoSecret(t *testing.T) {
	if err := VerifySlackSignature("", "1700000000", []byte("x"), "v0=abc", time.Unix(1_700_000_000, 0)); err == nil {
		t.Fatal("expected error when secret unconfigured")
	}
}

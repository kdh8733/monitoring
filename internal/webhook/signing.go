package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// maxSkew rejects replayed Slack requests whose timestamp is too old.
const maxSkew = 5 * time.Minute

// VerifySlackSignature validates a Slack request per
// https://api.slack.com/authentication/verifying-requests-from-slack:
// the signature is "v0=" + HMAC-SHA256(signingSecret, "v0:"+ts+":"+body).
// now is injected for testability.
func VerifySlackSignature(signingSecret, timestamp string, body []byte, signatureHeader string, now time.Time) error {
	if signingSecret == "" {
		return fmt.Errorf("slack signing secret not configured")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid slack timestamp")
	}
	if d := now.Sub(time.Unix(ts, 0)); d > maxSkew || d < -maxSkew {
		return fmt.Errorf("slack timestamp outside allowed window")
	}

	mac := hmac.New(sha256.New, []byte(signingSecret))
	fmt.Fprintf(mac, "v0:%s:%s", timestamp, body)
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signatureHeader)) {
		return fmt.Errorf("slack signature mismatch")
	}
	return nil
}

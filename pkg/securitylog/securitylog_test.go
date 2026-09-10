package securitylog

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// captureOutput captures logrus output during fn execution.
func captureOutput(fn func()) string {
	origOutput := logrus.StandardLogger().Out
	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter

	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	}()
	fn()
	return buf.String()
}

func TestLogSuccess(t *testing.T) {
	output := captureOutput(func() {
		Log(context.Background(), "CREATE", "favorite", "qs-1", "success")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "CREATE")
	assert.Contains(t, output, "favorite")
	assert.Contains(t, output, "qs-1")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "level=info")
}

func TestLogFailure(t *testing.T) {
	output := captureOutput(func() {
		Log(context.Background(), "DELETE", "progress", "42", "failure")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "DELETE")
	assert.Contains(t, output, "level=warning")
}

func TestLogWithReason(t *testing.T) {
	output := captureOutput(func() {
		LogWithReason(context.Background(), "AUTHENTICATE", "api_request", "/api/test", "failure", "missing identity header")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "AUTHENTICATE")
	assert.Contains(t, output, "missing identity header")
	assert.Contains(t, output, "level=warning")
}

func TestLogWithReasonSuccess(t *testing.T) {
	output := captureOutput(func() {
		LogWithReason(context.Background(), "UPDATE", "progress", "user1", "success", "progress updated")
	})

	assert.Contains(t, output, "level=info")
}

func TestLogStartup(t *testing.T) {
	output := captureOutput(func() {
		LogStartup("quickstarts", ":8000")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "STARTUP")
	assert.Contains(t, output, "process")
	assert.Contains(t, output, "quickstarts")
	assert.Contains(t, output, "level=info")
}

func TestLogShutdownFailure(t *testing.T) {
	output := captureOutput(func() {
		LogShutdown("quickstarts", "failure", "server stopped")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "SHUTDOWN")
	assert.Contains(t, output, "server stopped")
	assert.Contains(t, output, "failure")
	assert.Contains(t, output, "level=error")
}

func TestLogShutdownSuccess(t *testing.T) {
	output := captureOutput(func() {
		LogShutdown("quickstarts", "success", "")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "SHUTDOWN")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "level=info")
	assert.False(t, strings.Contains(output, "reason"))
}

func TestLogWithPrincipal(t *testing.T) {
	ctx := context.WithValue(context.Background(), identity.Key, identity.XRHID{
		Identity: identity.Identity{
			OrgID: "org-123",
			User:  identity.User{UserID: "user-456"},
		},
	})

	output := captureOutput(func() {
		Log(ctx, "CREATE", "favorite", "qs-1", "success")
	})

	assert.Contains(t, output, "user-456")
	assert.Contains(t, output, "org-123")
}

func TestLogServiceAccountIdentity(t *testing.T) {
	// Service accounts may have empty User — must not panic
	ctx := context.WithValue(context.Background(), identity.Key, identity.XRHID{
		Identity: identity.Identity{
			OrgID: "org-789",
			User:  identity.User{},
		},
	})

	output := captureOutput(func() {
		Log(ctx, "CREATE", "progress", "1", "success")
	})

	assert.Contains(t, output, "security_event")
	assert.Contains(t, output, "org-789")
	assert.False(t, strings.Contains(output, "user_id"))
}

func TestLogWithoutReasonOmitsField(t *testing.T) {
	output := captureOutput(func() {
		Log(context.Background(), "CREATE", "test", "1", "success")
	})

	assert.Contains(t, output, "security_event")
	assert.False(t, strings.Contains(output, "reason"))
}

func TestLogNilContext(t *testing.T) {
	output := captureOutput(func() {
		Log(nil, "READ", "test", "1", "success")
	})

	assert.Contains(t, output, "security_event")
	assert.False(t, strings.Contains(output, "user_id"))
}

func TestLogNoIdentityInContext(t *testing.T) {
	output := captureOutput(func() {
		Log(context.Background(), "CREATE", "test", "1", "success")
	})

	assert.Contains(t, output, "security_event")
	assert.False(t, strings.Contains(output, "user_id"))
}

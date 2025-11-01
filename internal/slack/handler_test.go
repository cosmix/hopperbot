package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rudderlabs/hopperbot/pkg/config"
	"go.uber.org/zap"
)

// Test helpers for creating valid Slack requests

// createValidSlackRequest creates a properly signed Slack request
func createValidSlackRequest(method, path string, body []byte, signingSecret string) *http.Request {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sigBaseString := fmt.Sprintf("%s:%s:%s", SignatureVersion, timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBaseString))
	signature := SignaturePrefix + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set(HeaderSlackRequestTimestamp, timestamp)
	req.Header.Set(HeaderSlackSignature, signature)
	return req
}

// TestValidateSlackRequest_ValidSignature tests valid Slack request signature verification
func TestValidateSlackRequest_ValidSignature(t *testing.T) {
	signingSecret := "test-secret"
	cfg := &config.Config{
		SlackSigningSecret: signingSecret,
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	body := []byte("command=%2Fhopperbot&trigger_id=trigger123")
	req := createValidSlackRequest(http.MethodPost, "/slack/command", body, signingSecret)

	w := httptest.NewRecorder()
	slackReq, ok := handler.validateSlackRequest(w, req)

	if !ok {
		t.Error("expected valid request, got invalid")
	}
	if slackReq == nil {
		t.Fatal("expected non-nil slackRequest")
	}
	if string(slackReq.Body) != string(body) {
		t.Errorf("body mismatch: got %s, want %s", string(slackReq.Body), string(body))
	}
}

// TestValidateSlackRequest_InvalidSignature tests invalid signature detection
func TestValidateSlackRequest_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "correct-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	body := []byte("test=data")
	req := createValidSlackRequest(http.MethodPost, "/slack/command", body, "wrong-secret")

	w := httptest.NewRecorder()
	_, ok := handler.validateSlackRequest(w, req)

	if ok {
		t.Error("expected invalid signature to be detected")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// TestValidateSlackRequest_MissingTimestamp tests missing timestamp handling
func TestValidateSlackRequest_MissingTimestamp(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/slack/command", bytes.NewBufferString("test=data"))
	req.Header.Set(HeaderSlackSignature, "v0=somesignature")
	// Missing timestamp header

	w := httptest.NewRecorder()
	_, ok := handler.validateSlackRequest(w, req)

	if ok {
		t.Error("expected request without timestamp to be invalid")
	}
}

// TestValidateSlackRequest_ExpiredTimestamp tests timestamp validation
func TestValidateSlackRequest_ExpiredTimestamp(t *testing.T) {
	signingSecret := "test-secret"
	cfg := &config.Config{
		SlackSigningSecret: signingSecret,
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	body := []byte("test=data")
	// Create request with old timestamp (older than 5 minutes)
	oldTimestamp := strconv.FormatInt(time.Now().Unix()-400, 10) // 400 seconds = 6+ minutes
	sigBaseString := fmt.Sprintf("%s:%s:%s", SignatureVersion, oldTimestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBaseString))
	signature := SignaturePrefix + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/slack/command", bytes.NewBuffer(body))
	req.Header.Set(HeaderSlackRequestTimestamp, oldTimestamp)
	req.Header.Set(HeaderSlackSignature, signature)

	w := httptest.NewRecorder()
	_, ok := handler.validateSlackRequest(w, req)

	if ok {
		t.Error("expected request with expired timestamp to be invalid")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// TestVerifySlackRequest_MissingSignature tests missing signature handling
func TestVerifySlackRequest_MissingSignature(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	headers := make(http.Header)
	headers.Set(HeaderSlackRequestTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
	// Missing signature header

	result := handler.verifySlackRequest(headers, []byte("test"))

	if result {
		t.Error("expected verification to fail with missing signature")
	}
}

// TestParseInteractionPayload_ValidPayload tests valid interaction payload parsing
func TestParseInteractionPayload_ValidPayload(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	payloadObj := map[string]interface{}{
		"type": "view_submission",
		"view": map[string]interface{}{
			"id":          "V123",
			"callback_id": ModalCallbackIDSubmitForm,
		},
		"user": map[string]interface{}{
			"id":       "U123",
			"username": "testuser",
		},
	}

	payloadJSON, _ := json.Marshal(payloadObj)
	values := url.Values{}
	values.Set("payload", string(payloadJSON))

	payload, err := handler.parseInteractionPayload(values)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payload.Type != InteractionTypeViewSubmission {
		t.Errorf("expected type %s, got %s", InteractionTypeViewSubmission, payload.Type)
	}

	if payload.User.ID != "U123" {
		t.Errorf("expected user ID U123, got %s", payload.User.ID)
	}
}

// TestParseInteractionPayload_MissingPayload tests missing payload handling
func TestParseInteractionPayload_MissingPayload(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	values := url.Values{} // Empty values

	_, err := handler.parseInteractionPayload(values)

	if err == nil {
		t.Error("expected error for missing payload, got nil")
	}

	if !strings.Contains(err.Error(), "missing payload") {
		t.Errorf("expected 'missing payload' error message, got %v", err)
	}
}

// TestParseInteractionPayload_InvalidJSON tests invalid JSON handling
func TestParseInteractionPayload_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	values := url.Values{}
	values.Set("payload", "invalid json {")

	_, err := handler.parseInteractionPayload(values)

	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "failed to unmarshal") {
		t.Errorf("expected unmarshal error, got %v", err)
	}
}

// TestShouldProcessSubmission_Valid tests valid submission identification
func TestShouldProcessSubmission_Valid(t *testing.T) {
	payload := &InteractionPayload{
		Type: InteractionTypeViewSubmission,
		View: View{
			CallbackID: ModalCallbackIDSubmitForm,
		},
	}

	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	if !handler.shouldProcessSubmission(payload) {
		t.Error("expected valid submission to be processed")
	}
}

// TestShouldProcessSubmission_WrongType tests submission rejection for wrong type
func TestShouldProcessSubmission_WrongType(t *testing.T) {
	payload := &InteractionPayload{
		Type: "block_actions", // Wrong type
		View: View{
			CallbackID: ModalCallbackIDSubmitForm,
		},
	}

	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	if handler.shouldProcessSubmission(payload) {
		t.Error("expected submission with wrong type to be rejected")
	}
}

// TestShouldProcessSubmission_WrongCallbackID tests submission rejection for wrong callback ID
func TestShouldProcessSubmission_WrongCallbackID(t *testing.T) {
	payload := &InteractionPayload{
		Type: InteractionTypeViewSubmission,
		View: View{
			CallbackID: "wrong_callback_id",
		},
	}

	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	if handler.shouldProcessSubmission(payload) {
		t.Error("expected submission with wrong callback ID to be rejected")
	}
}

// TestHandleInteractive_InvalidMethod tests method validation
func TestHandleInteractive_InvalidMethod(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/slack/interactive", nil)
	w := httptest.NewRecorder()

	handler.HandleInteractive(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestHandleSlashCommand_InvalidMethod tests method validation
func TestHandleSlashCommand_InvalidMethod(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/slack/command", nil)
	w := httptest.NewRecorder()

	handler.HandleSlashCommand(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestRespondSuccess tests success response format
func TestRespondSuccess(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	w := httptest.NewRecorder()
	handler.respondSuccess(w)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("expected empty response object, got %v", resp)
	}
}

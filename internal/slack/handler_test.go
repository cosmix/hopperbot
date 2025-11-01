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
	"github.com/rudderlabs/hopperbot/pkg/constants"
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
		t.Error("expected expired timestamp to be rejected")
	}
}

// TestParseInteractionPayload_ValidPayload tests valid payload parsing
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

	payload := InteractionPayload{
		Type: "view_submission",
		User: User{ID: "U123", Username: "testuser"},
		Team: Team{ID: "T123", Domain: "test"},
		View: View{
			ID:         "V123",
			CallbackID: ModalCallbackIDSubmitForm,
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	values := url.Values{"payload": {string(payloadBytes)}}

	parsedPayload, err := handler.parseInteractionPayload(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsedPayload.Type != payload.Type {
		t.Errorf("type mismatch: got %s, want %s", parsedPayload.Type, payload.Type)
	}
	if parsedPayload.User.ID != payload.User.ID {
		t.Errorf("user ID mismatch: got %s, want %s", parsedPayload.User.ID, payload.User.ID)
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

	values := url.Values{}

	_, err := handler.parseInteractionPayload(values)
	if err == nil {
		t.Error("expected error for missing payload")
	}
	if !strings.Contains(err.Error(), "missing payload") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestParseInteractionPayload_InvalidJSON tests invalid JSON parsing
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

	values := url.Values{"payload": {"invalid json"}}

	_, err := handler.parseInteractionPayload(values)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestExtractAndValidateFields_RequiredFieldsPresent tests extraction with valid fields
func TestExtractAndValidateFields_RequiredFieldsPresent(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "New Feature Idea",
						Text:  OptionText{Type: "plain_text", Text: "New Feature Idea"},
					},
				},
			},
			BlockIDProductArea: {
				ActionIDProductAreaSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "AI/ML",
						Text:  OptionText{Type: "plain_text", Text: "AI/ML"},
					},
				},
			},
		},
	}

	fields, err := handler.extractAndValidateFields(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields[constants.AliasTitle] != titleVal {
		t.Errorf("title mismatch: got %s, want %s", fields[constants.AliasTitle], titleVal)
	}
	if fields[constants.AliasTheme] != "New Feature Idea" {
		t.Errorf("theme mismatch: got %s, want 'New Feature Idea'", fields[constants.AliasTheme])
	}
	if fields[constants.AliasProductArea] != "AI/ML" {
		t.Errorf("product area mismatch: got %s, want AI/ML", fields[constants.AliasProductArea])
	}
}

// TestExtractAndValidateFields_MissingTitle tests missing required title field
func TestExtractAndValidateFields_MissingTitle(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: nil,
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for missing title")
	}
}

// TestExtractAndValidateFields_TitleTooLong tests title length validation
func TestExtractAndValidateFields_TitleTooLong(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	longTitle := strings.Repeat("a", constants.MaxTitleLength+1)
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &longTitle,
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for title exceeding max length")
	}
}

// TestExtractAndValidateFields_NoTheme tests missing required theme
func TestExtractAndValidateFields_NoTheme(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type:           "static_select",
					SelectedOption: nil,
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for missing theme")
	}
}

// TestExtractAndValidateFields_InvalidTheme tests invalid theme value
func TestExtractAndValidateFields_InvalidTheme(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "invalid_theme",
						Text:  OptionText{Type: "plain_text", Text: "invalid_theme"},
					},
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for invalid theme")
	}
}

// TestExtractAndValidateFields_InvalidProductArea tests invalid product area
func TestExtractAndValidateFields_InvalidProductArea(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "New Feature Idea",
						Text:  OptionText{Type: "plain_text", Text: "New Feature Idea"},
					},
				},
			},
			BlockIDProductArea: {
				ActionIDProductAreaSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "invalid_area",
						Text:  OptionText{Type: "plain_text"},
					},
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for invalid product area")
	}
}

// TestExtractAndValidateFields_OptionalComments tests optional comments field
func TestExtractAndValidateFields_OptionalComments(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	comments := "This is a test comment"
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "New Feature Idea",
						Text:  OptionText{Type: "plain_text", Text: "New Feature Idea"},
					},
				},
			},
			BlockIDProductArea: {
				ActionIDProductAreaSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "AI/ML",
						Text:  OptionText{Type: "plain_text"},
					},
				},
			},
			BlockIDComments: {
				ActionIDCommentsInput: {
					Type:  "plain_text_input",
					Value: &comments,
				},
			},
		},
	}

	fields, err := handler.extractAndValidateFields(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields[constants.AliasComments] != comments {
		t.Errorf("comments mismatch: got %s, want %s", fields[constants.AliasComments], comments)
	}
}

// TestExtractAndValidateFields_CommentsTooLong tests comments length validation
func TestExtractAndValidateFields_CommentsTooLong(t *testing.T) {
	cfg := &config.Config{
		SlackSigningSecret: "test-secret",
		SlackBotToken:      "test-token",
		NotionAPIKey:       "notion-key",
		NotionDatabaseID:   "db-id",
		NotionClientsDBID:  "clients-db-id",
	}

	logger, _ := zap.NewDevelopment()
	handler := NewHandler(cfg, logger)

	titleVal := "Test Idea"
	longComments := strings.Repeat("a", constants.MaxCommentLength+1)
	state := ViewState{
		Values: map[string]map[string]StateValue{
			BlockIDTitle: {
				ActionIDTitleInput: {
					Type:  "plain_text_input",
					Value: &titleVal,
				},
			},
			BlockIDTheme: {
				ActionIDThemeSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "New Feature Idea",
						Text:  OptionText{Type: "plain_text", Text: "New Feature Idea"},
					},
				},
			},
			BlockIDProductArea: {
				ActionIDProductAreaSelect: {
					Type: "static_select",
					SelectedOption: &SelectedOption{
						Value: "AI/ML",
						Text:  OptionText{Type: "plain_text"},
					},
				},
			},
			BlockIDComments: {
				ActionIDCommentsInput: {
					Type:  "plain_text_input",
					Value: &longComments,
				},
			},
		},
	}

	_, err := handler.extractAndValidateFields(state)
	if err == nil {
		t.Error("expected error for comments exceeding max length")
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

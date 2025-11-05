package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rudderlabs/hopperbot/internal/notion"
	"github.com/rudderlabs/hopperbot/pkg/cache"
	"github.com/rudderlabs/hopperbot/pkg/config"
	"github.com/rudderlabs/hopperbot/pkg/constants"
	"github.com/rudderlabs/hopperbot/pkg/metrics"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// MaxOptionsResults is the maximum number of options to return in a block suggestion response
const MaxOptionsResults = 100

type Handler struct {
	config       *Config
	notionClient *notion.Client
	slackClient  *slack.Client
	logger       *zap.Logger
	metrics      *metrics.Metrics
	cacheManager *cache.Manager
}

type Config struct {
	SigningSecret string
	BotToken      string
}

type slackRequest struct {
	Body   []byte
	Values url.Values
}

func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		config: &Config{
			SigningSecret: cfg.SlackSigningSecret,
			BotToken:      cfg.SlackBotToken,
		},
		notionClient: notion.NewClient(cfg.NotionAPIKey, cfg.NotionDatabaseID, cfg.NotionClientsDBID, logger),
		slackClient:  slack.New(cfg.SlackBotToken),
		logger:       logger,
	}
}

// SetCacheManager sets the cache manager instance for the handler
func (h *Handler) SetCacheManager(cm *cache.Manager) {
	h.cacheManager = cm
}

// Initialize initializes the handler by fetching required data from Notion
func (h *Handler) Initialize() error {
	// Discover data source IDs for both main and customers databases
	// Required for API v2025-09-03 which uses data source IDs instead of database IDs
	if err := h.notionClient.InitializeDataSources(); err != nil {
		return fmt.Errorf("failed to initialize data sources: %w", err)
	}

	// Fetch the list of valid customers from the Customers database
	if err := h.notionClient.InitializeCustomers(); err != nil {
		return fmt.Errorf("failed to initialize clients: %w", err)
	}

	// Fetch the list of Notion workspace users for Slack-to-Notion user mapping
	if err := h.notionClient.InitializeUsers(); err != nil {
		return fmt.Errorf("failed to initialize users: %w", err)
	}

	return nil
}

// InitializeCustomers refreshes the customer cache by delegating to the notion client
func (h *Handler) InitializeCustomers() error {
	return h.notionClient.InitializeCustomers()
}

// InitializeUsers refreshes the user cache by delegating to the notion client
func (h *Handler) InitializeUsers() error {
	return h.notionClient.InitializeUsers()
}

// GetCachedUserEmails returns the list of cached user emails for debugging
func (h *Handler) GetCachedUserEmails() []string {
	return h.notionClient.GetCachedUserEmails()
}

// GetUserCacheSize returns the number of users in the cache
func (h *Handler) GetUserCacheSize() int {
	return h.notionClient.GetUserCacheSize()
}

// HandleSlashCommand handles incoming Slack slash commands
func (h *Handler) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate and parse Slack request
	req, ok := h.validateSlackRequest(w, r)
	if !ok {
		return
	}

	triggerID := req.Values.Get("trigger_id")
	userName := req.Values.Get("user_name")
	command := req.Values.Get("command")
	text := strings.TrimSpace(req.Values.Get("text"))

	h.logger.Info("received slash command",
		zap.String("command", command),
		zap.String("text", text),
		zap.String("user", userName),
		zap.String("trigger_id", triggerID),
		zap.Int("trigger_id_length", len(triggerID)),
	)

	// Check if this is a refresh-cache command
	if text == "refresh-cache" {
		h.handleRefreshCacheCommand(w, r)
		return
	}

	// Default behavior: open modal
	h.handleOpenModalCommand(w, r, triggerID, command)
}

// handleOpenModalCommand handles the default /hopperbot command to open the modal
func (h *Handler) handleOpenModalCommand(w http.ResponseWriter, _ *http.Request, triggerID, command string) {
	// Validate trigger_id
	if triggerID == "" {
		h.logger.Error("trigger_id is empty")
		h.recordSlackCommand(command, "error")
		respondToSlack(w, "Internal error: missing trigger_id")
		return
	}

	// Build modal (customer options loaded dynamically via external select)
	modal := BuildSubmissionModal()

	// Debug: log modal structure to diagnose issue
	if modalJSON, err := json.MarshalIndent(modal, "", "  "); err == nil {
		h.logger.Debug("modal structure being sent to Slack", zap.String("json", string(modalJSON)))
	}

	// Open the modal
	viewResponse, err := h.slackClient.OpenView(triggerID, modal)
	if err != nil {
		h.logger.Error("failed to open modal",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
		)

		// Check if it's a SlackErrorResponse with more details
		if slackErr, ok := err.(slack.SlackErrorResponse); ok {
			h.logger.Error("slack API error details",
				zap.String("error", slackErr.Err),
				zap.String("response_metadata", fmt.Sprintf("%+v", slackErr.ResponseMetadata)),
			)
		} else if slackErrPtr, ok := err.(*slack.SlackErrorResponse); ok {
			h.logger.Error("slack API error details (pointer)",
				zap.String("error", slackErrPtr.Err),
				zap.String("response_metadata", fmt.Sprintf("%+v", slackErrPtr.ResponseMetadata)),
			)
		} else {
			// Log the raw error string if type assertion fails
			h.logger.Error("unable to extract slack error details",
				zap.String("error_string", err.Error()),
			)
		}

		// Also log the modal structure on error for debugging
		if modalJSON, marshalErr := json.MarshalIndent(modal, "", "  "); marshalErr == nil {
			h.logger.Error("modal that failed to open", zap.String("modal_json", string(modalJSON)))
		}

		h.recordSlackCommand(command, "error")
		respondToSlack(w, "Failed to open submission form. Please try again.")
		return
	}

	h.logger.Info("modal opened successfully", zap.String("view_id", viewResponse.ID))
	h.recordSlackCommand(command, "success")

	// Respond with 200 OK immediately (empty response)
	w.WriteHeader(http.StatusOK)
}

// handleRefreshCacheCommand handles the /hopperbot refresh-cache command
func (h *Handler) handleRefreshCacheCommand(w http.ResponseWriter, _ *http.Request) {
	h.logger.Info("refresh-cache command received")

	if h.cacheManager == nil {
		h.logger.Error("cache manager not initialized, cannot process refresh-cache command")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Info("manual cache refresh triggered via slash command")

	// Trigger async refresh (non-blocking)
	h.cacheManager.ManualRefresh()

	// Silent response - just return 200 OK (no visible message to user)
	w.WriteHeader(http.StatusOK)
}

// HandleInteractive handles incoming Slack interactive component submissions
func (h *Handler) HandleInteractive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate and parse Slack request
	req, ok := h.validateSlackRequest(w, r)
	if !ok {
		return
	}

	payload, err := h.parseInteractionPayload(req.Values)
	if err != nil {
		h.handleError(w, err, "Bad request", http.StatusBadRequest)
		return
	}

	// Validate the payload
	if err := payload.Validate(); err != nil {
		h.handleError(w, err, "Invalid interaction payload", http.StatusBadRequest)
		return
	}

	h.logger.Info("received interaction",
		zap.String("type", payload.Type),
		zap.String("callback_id", payload.View.CallbackID),
		zap.String("user", payload.User.Username),
	)

	// Record interaction received
	h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "received")

	if !h.shouldProcessSubmission(payload) {
		h.logger.Info("ignoring interaction",
			zap.String("type", payload.Type),
			zap.String("callback_id", payload.View.CallbackID),
		)
		h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "ignored")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Fetch Slack user email and map to Notion user
	slackUser, err := h.slackClient.GetUserInfo(payload.User.ID)
	if err != nil {
		h.logger.Error("failed to fetch Slack user info", zap.Error(err), zap.String("user_id", payload.User.ID))
		h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "user_lookup_error")
		h.recordModalSubmission("error")
		respondWithErrors(w, map[string]string{
			BlockIDTitle: "Failed to identify user. Please try again.",
		})
		return
	}

	// Map Slack user email to Notion user UUID
	slackEmail := slackUser.Profile.Email
	h.logger.Info("attempting to map Slack user to Notion user",
		zap.String("slack_email", slackEmail),
		zap.String("slack_user_id", payload.User.ID),
		zap.String("slack_username", payload.User.Username),
		zap.String("slack_real_name", slackUser.RealName),
	)

	notionUserID, found := h.notionClient.GetNotionUserIDByEmail(slackEmail)
	if !found {
		h.logger.Warn("Slack user email not found in Notion workspace",
			zap.String("email", slackEmail),
			zap.String("normalized_email", strings.ToLower(strings.TrimSpace(slackEmail))),
			zap.String("slack_user_id", payload.User.ID),
			zap.String("slack_username", payload.User.Username),
			zap.Int("notion_user_cache_size", h.notionClient.GetUserCacheSize()),
		)
		h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "user_not_found")
		h.recordModalSubmission("error")
		respondWithErrors(w, map[string]string{
			BlockIDTitle: fmt.Sprintf("Your Slack email (%s) is not associated with a Notion account in this workspace. Please contact your administrator.", slackEmail),
		})
		return
	}

	h.logger.Info("successfully mapped Slack user to Notion user",
		zap.String("slack_email", slackEmail),
		zap.String("notion_user_id", notionUserID),
	)

	fields, err := h.extractAndValidateFields(payload.View.State)
	if err != nil {
		h.logger.Warn("field validation failed", zap.Error(err))
		h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "validation_error")
		h.recordModalSubmission("validation_error")
		respondWithErrors(w, err.(fieldValidationError).errors)
		return
	}

	// Add the submitter's Notion user ID to the fields
	fields[constants.AliasSubmittedBy] = notionUserID

	h.logger.Info("extracted form fields",
		zap.String("title", fields[constants.AliasTitle]),
		zap.String("theme", fields[constants.AliasTheme]),
		zap.String("product_area", fields[constants.AliasProductArea]),
		zap.String("comments", fields[constants.AliasComments]),
		zap.String("customer_org", fields[constants.AliasCustomerOrg]),
		zap.String("submitted_by", notionUserID),
		zap.String("slack_email", slackUser.Profile.Email),
	)

	if err := h.notionClient.SubmitForm(fields); err != nil {
		h.logger.Error("failed to submit to Notion", zap.Error(err))
		h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "notion_error")
		h.recordModalSubmission("error")
		respondWithErrors(w, map[string]string{
			BlockIDTitle: fmt.Sprintf("Failed to submit: %v", err),
		})
		return
	}

	h.logger.Info("successfully submitted form to Notion",
		zap.String("user", payload.User.Username),
	)

	// Record successful submission
	h.recordSlackInteraction(payload.Type, payload.View.CallbackID, "success")
	h.recordModalSubmission("success")

	// Respond with success - modal will close automatically
	h.respondSuccess(w)
}

// HandleOptionsRequest handles block suggestion requests for external select options
func (h *Handler) HandleOptionsRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Add a 5-second timeout to this request
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	// Validate and parse Slack request
	req, ok := h.validateSlackRequest(w, r)
	if !ok {
		return
	}

	// Parse the options request payload
	optionsRequest, err := h.parseOptionsRequest(req.Values)
	if err != nil {
		h.handleError(w, err, "Bad request", http.StatusBadRequest)
		return
	}

	// Validate the options request
	if err := optionsRequest.Validate(); err != nil {
		h.handleError(w, err, "Invalid options request", http.StatusBadRequest)
		return
	}

	// Validate action_id is for customer org selection
	if optionsRequest.ActionID != ActionIDCustomerOrgSelect {
		h.logger.Warn("unexpected action_id in options request",
			zap.String("action_id", optionsRequest.ActionID),
			zap.String("expected", ActionIDCustomerOrgSelect),
		)
		h.respondWithOptions(w, []Option{})
		return
	}

	// Get all valid customers from cache and filter based on search query
	allCustomers := h.notionClient.GetValidCustomers()
	filteredOptions := FilterCustomerOptions(allCustomers, optionsRequest.Value, constants.MaxOptionsResults)

	h.logger.Debug("responding to options request",
		zap.String("action_id", optionsRequest.ActionID),
		zap.String("query", optionsRequest.Value),
		zap.Int("results_count", len(filteredOptions)),
	)

	h.respondWithOptions(w, filteredOptions)
}

// parseOptionsRequest parses and unmarshals an options request from the request values
func (h *Handler) parseOptionsRequest(values url.Values) (*OptionsRequest, error) {
	payloadStr := values.Get("payload")
	if payloadStr == "" {
		return nil, fmt.Errorf("missing payload in request")
	}

	var optionsRequest OptionsRequest
	if err := json.Unmarshal([]byte(payloadStr), &optionsRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options request: %w", err)
	}

	return &optionsRequest, nil
}

// respondWithOptions sends an options response to Slack
func (h *Handler) respondWithOptions(w http.ResponseWriter, options []Option) {
	response := OptionsResponse{
		Options: options,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode options response", zap.Error(err))
	}
}

// parseInteractionPayload parses and unmarshals the interaction payload from the request
func (h *Handler) parseInteractionPayload(values url.Values) (*InteractionPayload, error) {
	payloadStr := values.Get("payload")
	if payloadStr == "" {
		return nil, fmt.Errorf("missing payload in request")
	}

	var payload InteractionPayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &payload, nil
}

// shouldProcessSubmission checks if the interaction should be processed
// Returns true only for view submissions with the correct callback ID
func (h *Handler) shouldProcessSubmission(payload *InteractionPayload) bool {
	return payload.Type == InteractionTypeViewSubmission &&
		payload.View.CallbackID == ModalCallbackIDSubmitForm
}

// fieldValidationError wraps validation errors with the error map for Slack
type fieldValidationError struct {
	errors map[string]string
}

func (e fieldValidationError) Error() string {
	return fmt.Sprintf("validation failed: %v", e.errors)
}

// extractAndValidateFields extracts all form fields from the view state
// and validates required fields with comprehensive length and value checks.
// Returns a combined map of all fields or validation errors.
func (h *Handler) extractAndValidateFields(state ViewState) (map[string]string, error) {
	fields := make(map[string]string)
	validationErrors := make(map[string]string)

	// Extract and validate title (required, max 2000 chars)
	title, err := state.GetValue(BlockIDTitle, ActionIDTitleInput)
	if err != nil {
		validationErrors[BlockIDTitle] = fmt.Sprintf("Failed to extract title: %v", err)
		h.recordValidationError("title")
	} else {
		title = strings.TrimSpace(title)
		if title == "" {
			validationErrors[BlockIDTitle] = "Title is required"
			h.recordValidationError("title")
		} else if len(title) > constants.MaxTitleLength {
			validationErrors[BlockIDTitle] = fmt.Sprintf("Title exceeds maximum length of %d characters (current: %d)",
				constants.MaxTitleLength, len(title))
			h.recordValidationError("title")
		} else {
			fields[constants.AliasTitle] = title
		}
	}

	// Extract and validate theme (single select, required)
	theme, err := state.GetSelectedOption(BlockIDTheme, ActionIDThemeSelect)
	if err != nil {
		validationErrors[BlockIDTheme] = fmt.Sprintf("Failed to extract theme: %v", err)
		h.recordValidationError("theme")
	} else {
		theme = strings.TrimSpace(theme)
		if theme == "" {
			validationErrors[BlockIDTheme] = "Theme is required"
			h.recordValidationError("theme")
		} else if !slices.Contains(constants.ValidThemeCategories, theme) {
			validationErrors[BlockIDTheme] = fmt.Sprintf("Invalid theme selected: %s", theme)
			h.recordValidationError("theme")
		} else {
			fields[constants.AliasTheme] = theme
		}
	}

	// Extract and validate product area (single select, required)
	productArea, err := state.GetSelectedOption(BlockIDProductArea, ActionIDProductAreaSelect)
	if err != nil {
		validationErrors[BlockIDProductArea] = fmt.Sprintf("Failed to extract product area: %v", err)
		h.recordValidationError("product_area")
	} else {
		productArea = strings.TrimSpace(productArea)
		if productArea == "" {
			validationErrors[BlockIDProductArea] = "Product area is required"
			h.recordValidationError("product_area")
		} else if !slices.Contains(constants.ValidProductAreas, productArea) {
			validationErrors[BlockIDProductArea] = fmt.Sprintf("Invalid product area selected: %s", productArea)
			h.recordValidationError("product_area")
		} else {
			fields[constants.AliasProductArea] = productArea
		}
	}

	// Return validation errors if any required fields failed
	if len(validationErrors) > 0 {
		return nil, fieldValidationError{
			errors: validationErrors,
		}
	}

	// Extract and validate comments (optional, max 2000 chars)
	if comments, err := state.GetValue(BlockIDComments, ActionIDCommentsInput); err == nil {
		comments = strings.TrimSpace(comments)
		if comments != "" {
			if len(comments) > constants.MaxCommentLength {
				h.recordValidationError("comments")
				return nil, fieldValidationError{
					errors: map[string]string{
						BlockIDComments: fmt.Sprintf("Comments exceed maximum length of %d characters (current: %d)",
							constants.MaxCommentLength, len(comments)),
					},
				}
			}
			fields[constants.AliasComments] = comments
		}
	}

	// Extract and validate customer org (multi-select, optional, max 10)
	if orgs, err := state.GetSelectedOptions(BlockIDCustomerOrg, ActionIDCustomerOrgSelect); err == nil && len(orgs) > 0 {
		if len(orgs) > constants.MaxCustomerOrgSelections {
			h.recordValidationError("customer_org")
			return nil, fieldValidationError{
				errors: map[string]string{
					BlockIDCustomerOrg: fmt.Sprintf("Too many customer orgs selected (max: %d, selected: %d)",
						constants.MaxCustomerOrgSelections, len(orgs)),
				},
			}
		}
		// Validate each customer org against valid values
		validCustomers := h.notionClient.GetValidCustomers()
		for _, org := range orgs {
			if !slices.Contains(validCustomers, org) {
				h.recordValidationError("customer_org")
				return nil, fieldValidationError{
					errors: map[string]string{
						BlockIDCustomerOrg: fmt.Sprintf("Invalid customer org selected: %s", org),
					},
				}
			}
		}
		fields[constants.AliasCustomerOrg] = strings.Join(orgs, ",")
	}

	return fields, nil
}

// respondSuccess sends a successful empty response to Slack that closes the modal
func (h *Handler) respondSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// handleError handles errors consistently across all handlers by logging the error
// and sending an appropriate HTTP response with a user-friendly message
func (h *Handler) handleError(w http.ResponseWriter, err error, userMessage string, statusCode int) {
	h.logger.Error("handler error",
		zap.Error(err),
		zap.String("user_message", userMessage),
		zap.Int("status_code", statusCode),
	)
	http.Error(w, userMessage, statusCode)
}

// validateSlackRequest validates and parses a Slack request
// Returns the parsed request and true if valid, or nil and false if invalid (error response already written)
func (h *Handler) validateSlackRequest(w http.ResponseWriter, r *http.Request) (*slackRequest, bool) {
	// Read body
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.handleError(w, err, "Bad request", http.StatusBadRequest)
		return nil, false
	}

	// Verify Slack request signature
	if !h.verifySlackRequest(r.Header, body) {
		h.handleError(w, fmt.Errorf("invalid Slack signature"), "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	// Parse form data
	values, err := url.ParseQuery(string(body))
	if err != nil {
		h.handleError(w, err, "Bad request", http.StatusBadRequest)
		return nil, false
	}

	return &slackRequest{
		Body:   body,
		Values: values,
	}, true
}

// verifySlackRequest verifies that the request came from Slack
func (h *Handler) verifySlackRequest(headers http.Header, body []byte) bool {
	timestamp := headers.Get(HeaderSlackRequestTimestamp)
	signature := headers.Get(HeaderSlackSignature)

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp is within 5 minutes
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-ts > constants.MaxSlackRequestAge {
		return false
	}

	// Compute signature
	sigBaseString := fmt.Sprintf("%s:%s:%s", SignatureVersion, timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(h.config.SigningSecret))
	mac.Write([]byte(sigBaseString))
	expectedSignature := SignaturePrefix + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// respondToSlack sends a response back to Slack
func respondToSlack(w http.ResponseWriter, message string) {
	response := map[string]string{
		"response_type": "ephemeral",
		"text":          message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// respondWithErrors sends a view submission response with validation errors
func respondWithErrors(w http.ResponseWriter, errors map[string]string) {
	response := ViewSubmissionResponse{
		ResponseAction: ResponseActionErrors,
		Errors:         errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

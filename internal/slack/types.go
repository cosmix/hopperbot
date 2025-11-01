// Package slack provides handlers and types for Slack integration.
//
// This package implements the core Slack bot functionality including:
// - Slash command handling (/hopperbot)
// - Interactive modal views with searchable dropdowns
// - Form field extraction and validation
// - Slack request signature verification for security
//
// The package follows Slack's Block Kit modal architecture where users
// interact with forms through modal views triggered by slash commands.
package slack

import (
	"encoding/json"
	"fmt"
)

// InteractionPayload represents the main payload structure for Slack interactions
// including modal submissions, button clicks, and other interactive components.
//
// When a user submits a modal or interacts with a component, Slack sends a POST
// request with this payload structure. The payload is URL-encoded in the "payload"
// form parameter and must be parsed and validated before use.
type InteractionPayload struct {
	Type        string    `json:"type"`
	User        User      `json:"user"`
	View        View      `json:"view"`
	TriggerID   string    `json:"trigger_id,omitempty"`
	Team        Team      `json:"team"`
	APIAppID    string    `json:"api_app_id"`
	Token       string    `json:"token"`
	ResponseURL string    `json:"response_url,omitempty"`
	Actions     []Action  `json:"actions,omitempty"`
	Container   Container `json:"container,omitempty"`
}

// User represents the Slack user who triggered the interaction
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	TeamID   string `json:"team_id"`
	Email    string `json:"email,omitempty"` // Populated via Slack API GetUserInfo call
}

// Team represents the Slack workspace
type Team struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

// View represents a Slack modal view
type View struct {
	ID                 string            `json:"id"`
	TeamID             string            `json:"team_id"`
	Type               string            `json:"type"`
	CallbackID         string            `json:"callback_id"`
	Title              ViewElement       `json:"title"`
	Close              ViewElement       `json:"close,omitempty"`
	Submit             ViewElement       `json:"submit,omitempty"`
	Blocks             []json.RawMessage `json:"blocks"`
	PrivateMetadata    string            `json:"private_metadata,omitempty"`
	State              ViewState         `json:"state"`
	Hash               string            `json:"hash"`
	ClearOnClose       bool              `json:"clear_on_close"`
	NotifyOnClose      bool              `json:"notify_on_close"`
	RootViewID         string            `json:"root_view_id,omitempty"`
	AppID              string            `json:"app_id,omitempty"`
	ExternalID         string            `json:"external_id,omitempty"`
	AppInstalledTeamID string            `json:"app_installed_team_id,omitempty"`
	BotID              string            `json:"bot_id,omitempty"`
}

// ViewElement represents a text element in a view (title, submit button, etc.)
//
// Used for modal titles, button labels, and other text-based UI elements.
// The Type field should be "plain_text" for most cases.
//
// Note: This has the same structure as OptionText but serves a different semantic purpose.
// ViewElement is for modal UI elements, while OptionText is for select menu options.
// Keeping them separate improves code clarity and follows Slack's API conventions.
type ViewElement struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// ViewState represents the state of a view, containing all input values.
//
// The structure is nested: state.values[block_id][action_id] -> StateValue
// This allows the modal to organize field values by block and action IDs.
// Use the helper methods (GetValue, GetSelectedOption, GetSelectedOptions)
// to safely extract values from the state.
type ViewState struct {
	Values map[string]map[string]StateValue `json:"values"`
}

// StateValue represents a single input value from the view state.
//
// Slack sends different field structures based on the input type:
// - Text inputs: Value field contains the string
// - Single-select: SelectedOption contains the chosen option
// - Multi-select: SelectedOptions contains array of chosen options
// - Date/time/user/channel: Respective fields contain the selection
//
// The Type field indicates which field(s) will be populated.
type StateValue struct {
	Type                 string           `json:"type"`
	Value                *string          `json:"value,omitempty"`
	SelectedDate         string           `json:"selected_date,omitempty"`
	SelectedTime         string           `json:"selected_time,omitempty"`
	SelectedUser         string           `json:"selected_user,omitempty"`
	SelectedChannel      string           `json:"selected_channel,omitempty"`
	SelectedConversation string           `json:"selected_conversation,omitempty"`
	SelectedOption       *SelectedOption  `json:"selected_option,omitempty"`
	SelectedOptions      []SelectedOption `json:"selected_options,omitempty"`
}

// SelectedOption represents a selected option from a select menu.
//
// Used in both single-select and multi-select dropdowns to represent
// the user's selection. The Value field contains the actual data to process,
// while Text contains the display label shown to the user.
type SelectedOption struct {
	Text  OptionText `json:"text"`
	Value string     `json:"value"`
}

// OptionText represents the text of an option in a select menu.
//
// Contains formatting information for how the option text is displayed.
// Type should typically be "plain_text" unless rich formatting is needed.
//
// Note: This has the same structure as ViewElement but serves a different semantic purpose.
// OptionText is for select menu options, while ViewElement is for modal UI elements.
// Keeping them separate improves code clarity and follows Slack's API conventions.
type OptionText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// Action represents an action taken in an interactive component
type Action struct {
	Type            string           `json:"type"`
	ActionID        string           `json:"action_id"`
	BlockID         string           `json:"block_id"`
	Value           string           `json:"value,omitempty"`
	ActionTS        string           `json:"action_ts"`
	SelectedOption  *SelectedOption  `json:"selected_option,omitempty"`
	SelectedOptions []SelectedOption `json:"selected_options,omitempty"`
}

// Container represents the container of an interactive component
type Container struct {
	Type        string `json:"type"`
	MessageTS   string `json:"message_ts,omitempty"`
	ChannelID   string `json:"channel_id,omitempty"`
	IsEphemeral bool   `json:"is_ephemeral,omitempty"`
	ViewID      string `json:"view_id,omitempty"`
}

// OptionsRequest represents a block suggestion request from Slack for external select options.
//
// When a user types in a search field for an external select menu (e.g., Client Organization),
// Slack sends a POST request with this payload to fetch matching options.
// The request is URL-encoded with the payload in the "payload" form parameter.
type OptionsRequest struct {
	Type      string    `json:"type"`       // "block_suggestion"
	ActionID  string    `json:"action_id"`  // The action ID of the select menu
	BlockID   string    `json:"block_id"`   // The block ID containing the select menu
	Value     string    `json:"value"`      // User's search input text
	Team      Team      `json:"team"`       // The Slack workspace info
	User      User      `json:"user"`       // The user making the request
	Container Container `json:"container"`  // Container information
	TriggerID string    `json:"trigger_id"` // Trigger ID for potential follow-up actions
	APIAppID  string    `json:"api_app_id"` // App ID
	Token     string    `json:"token"`      // Verification token
}

// Option represents a single option in a select menu.
//
// Used in OptionsResponse to return matching options to Slack.
// The Value field is what gets submitted when the option is selected,
// while Text is what the user sees.
type Option struct {
	Text  OptionText `json:"text"`
	Value string     `json:"value"`
}

// OptionsResponse represents the response structure for block suggestion requests.
//
// When handling a block suggestion request, the server responds with this structure
// containing the filtered/matching options to display to the user.
type OptionsResponse struct {
	Options []Option `json:"options"`
}

// ResponseAction represents the type of response action for view submissions.
//
// Slack supports different response actions when processing modal submissions:
// - errors: Display field-specific validation errors to the user
// - clear: Close the modal without showing errors
// - update: Update the current modal with new content
// - push: Push a new modal onto the stack (for multi-step flows)
type ResponseAction string

// Common response actions for modal submission responses.
const (
	// ResponseActionErrors displays validation errors to the user without closing the modal.
	// The Errors map should contain block_id -> error_message pairs.
	ResponseActionErrors ResponseAction = "errors"

	// ResponseActionClear closes the modal without showing any errors.
	ResponseActionClear ResponseAction = "clear"

	// ResponseActionUpdate updates the current modal view with new content.
	ResponseActionUpdate ResponseAction = "update"

	// ResponseActionPush pushes a new modal view onto the navigation stack.
	ResponseActionPush ResponseAction = "push"
)

// ViewSubmissionResponse represents the response structure for view submissions.
//
// When a modal is submitted, the server can respond with this structure to:
// 1. Display validation errors (using ResponseActionErrors)
// 2. Update the modal content (using ResponseActionUpdate)
// 3. Push a new modal view (using ResponseActionPush)
// 4. Clear/close the modal (using ResponseActionClear or empty response)
type ViewSubmissionResponse struct {
	ResponseAction ResponseAction    `json:"response_action,omitempty"`
	Errors         map[string]string `json:"errors,omitempty"`
	View           *View             `json:"view,omitempty"`
}

// Validate checks if the InteractionPayload has all required fields
func (ip *InteractionPayload) Validate() error {
	if ip.Type == "" {
		return fmt.Errorf("interaction type is required")
	}
	if ip.User.ID == "" {
		return fmt.Errorf("user ID is required")
	}
	if ip.Team.ID == "" {
		return fmt.Errorf("team ID is required")
	}
	return nil
}

// Validate checks if the OptionsRequest has all required fields
func (or *OptionsRequest) Validate() error {
	if or.Type != "block_suggestion" {
		return fmt.Errorf("expected type 'block_suggestion', got %q", or.Type)
	}
	if or.ActionID == "" {
		return fmt.Errorf("action_id is required")
	}
	if or.BlockID == "" {
		return fmt.Errorf("block_id is required")
	}
	if or.Team.ID == "" {
		return fmt.Errorf("team ID is required")
	}
	return nil
}

// IsTextInput returns true if the StateValue represents a text input field.
//
// Recognizes both plain text inputs (single-line or multi-line) and
// rich text inputs (formatted text with styling).
func (sv *StateValue) IsTextInput() bool {
	return sv.Type == "plain_text_input" || sv.Type == "rich_text_input"
}

// IsSelect returns true if the StateValue represents a single-select field.
//
// Recognizes all single-select types including:
// - static_select: predefined options
// - external_select: dynamically loaded options
// - users_select: user picker
// - conversations_select: conversation picker
// - channels_select: channel picker
func (sv *StateValue) IsSelect() bool {
	return sv.Type == "static_select" || sv.Type == "external_select" ||
		sv.Type == "users_select" || sv.Type == "conversations_select" ||
		sv.Type == "channels_select"
}

// IsMultiSelect returns true if the StateValue represents a multi-select field.
//
// Recognizes all multi-select types including:
// - multi_static_select: predefined options allowing multiple selections
// - multi_external_select: dynamically loaded options with multiple selections
// - multi_users_select: multiple user picker
// - multi_conversations_select: multiple conversation picker
// - multi_channels_select: multiple channel picker
func (sv *StateValue) IsMultiSelect() bool {
	return sv.Type == "multi_static_select" || sv.Type == "multi_external_select" ||
		sv.Type == "multi_users_select" || sv.Type == "multi_conversations_select" ||
		sv.Type == "multi_channels_select"
}

// GetValue extracts a plain text value from the view state.
//
// Used for text input fields (plain_text_input or rich_text_input).
// Returns the text value if found, or an error if the block/action doesn't exist.
// Returns an empty string (without error) if the field exists but has no value.
//
// Example:
//
//	title, err := state.GetValue("title_block", "title_input")
//	if err != nil {
//	    // Field doesn't exist in the form
//	}
func (vs *ViewState) GetValue(blockID, actionID string) (string, error) {
	if vs.Values == nil {
		return "", fmt.Errorf("view state values is nil")
	}

	block, exists := vs.Values[blockID]
	if !exists {
		return "", fmt.Errorf("block %q not found in view state", blockID)
	}

	stateValue, exists := block[actionID]
	if !exists {
		return "", fmt.Errorf("action %q not found in block %q", actionID, blockID)
	}

	if stateValue.Value != nil {
		return *stateValue.Value, nil
	}

	return "", nil
}

// GetSelectedOption extracts a selected option value from a single-select dropdown.
//
// Used for single-select fields (static_select, external_select, etc.).
// Returns the selected option's value if one is chosen, or an empty string if no selection.
// Returns an error only if the block/action doesn't exist in the form.
//
// IMPORTANT: This method includes nil checks to prevent panics when no option is selected.
// Slack omits the SelectedOption field entirely when nothing is selected.
//
// Example:
//
//	productArea, err := state.GetSelectedOption("product_area_block", "product_area_select")
//	if err != nil {
//	    // Field doesn't exist in the form
//	}
//	if productArea == "" {
//	    // No option selected (for optional fields)
//	}
func (vs *ViewState) GetSelectedOption(blockID, actionID string) (string, error) {
	if vs.Values == nil {
		return "", fmt.Errorf("view state values is nil")
	}

	block, exists := vs.Values[blockID]
	if !exists {
		return "", fmt.Errorf("block %q not found in view state", blockID)
	}

	stateValue, exists := block[actionID]
	if !exists {
		return "", fmt.Errorf("action %q not found in block %q", actionID, blockID)
	}

	// CRITICAL: Check if SelectedOption is nil before accessing its fields
	// When no option is selected, Slack doesn't send the field and it will be nil
	if stateValue.SelectedOption == nil {
		return "", nil
	}

	// Validate that the option has a non-empty value
	if stateValue.SelectedOption.Value != "" {
		return stateValue.SelectedOption.Value, nil
	}

	return "", nil
}

// GetSelectedOptions extracts multiple selected option values from a multi-select dropdown.
//
// Used for multi-select fields (multi_static_select, multi_external_select, etc.).
// Returns a slice of selected values. Empty slice indicates no selections.
// Returns an error only if the block/action doesn't exist in the form.
//
// The returned slice contains only non-empty values. Empty option values are filtered out.
//
// Example:
//
//	themes, err := state.GetSelectedOptions("theme_block", "theme_select")
//	if err != nil {
//	    // Field doesn't exist in the form
//	}
//	if len(themes) == 0 {
//	    // No options selected (for optional fields)
//	}
//	if len(themes) > 2 {
//	    // Too many selections (enforce business rule)
//	}
func (vs *ViewState) GetSelectedOptions(blockID, actionID string) ([]string, error) {
	if vs.Values == nil {
		return nil, fmt.Errorf("view state values is nil")
	}

	block, exists := vs.Values[blockID]
	if !exists {
		return nil, fmt.Errorf("block %q not found in view state", blockID)
	}

	stateValue, exists := block[actionID]
	if !exists {
		return nil, fmt.Errorf("action %q not found in block %q", actionID, blockID)
	}

	// Extract values from all selected options
	var values []string
	for _, opt := range stateValue.SelectedOptions {
		if opt.Value != "" {
			values = append(values, opt.Value)
		}
	}

	return values, nil
}

// Package slack provides handlers and types for Slack integration.
//
// This file implements Slack modal building functions. The modal is the
// interactive form that appears when users invoke the /hopperbot slash command.
// It contains fields for submitting ideas to the Notion database:
//
// Required Fields:
//   - Title: Single-line text input
//   - Theme/Category: Single-select dropdown
//   - Product Area: Single-select dropdown
//
// Optional Fields:
//   - Comments: Multiline text input
//   - Customer Org: Multi-select external dropdown (loads options dynamically)
//
// Modal Structure:
// The modal is built as a View with Blocks. Each block represents a form field.
// Blocks use ActionIDs to identify field values when the modal is submitted.
//
// Example of building a modal:
//
//	modal := BuildSubmissionModal()
//	// Returns a ModalViewRequest with all 5 form fields configured
package slack

import (
	"github.com/rudderlabs/hopperbot/pkg/constants"
	"github.com/slack-go/slack"
)

// BuildSubmissionModal constructs the main Slack modal view for the /hopperbot command.
// The modal includes all required and optional form fields with proper labels and placeholders.
// Each field is configured with appropriate element types (text input, select, multi-select).
//
// The modal has 5 blocks:
// 1. Title (required) - Single-line text input
// 2. Theme/Category (required) - Single-select dropdown with 4 theme options
// 3. Product Area (required) - Single-select dropdown with product area options
// 4. Comments (optional) - Multiline text input
// 5. Customer Org (optional) - Multi-select external dropdown (loads options dynamically)
//
// Example:
//
//	modal := BuildSubmissionModal()
//	// modal.Type == VTModal
//	// modal.CallbackID == "submit_form_modal"
//	// len(modal.Blocks.BlockSet) == 5
func BuildSubmissionModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: ModalCallbackIDSubmitForm,
		Title:      newPlainText(ModalTitle),
		Submit:     newPlainText(ModalSubmitText),
		Close:      newPlainText(ModalCancelText),
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				buildTitleBlock(),
				buildThemeBlock(),
				buildProductAreaBlock(),
				buildCommentsBlock(),
				buildCustomerOrgBlock(),
			},
		},
	}
}

// buildTitleBlock creates the "Title" form field block.
// This is a required single-line text input for the idea/topic name.
//
// Returns an InputBlock with a PlainTextInput element.
// BlockID: "title_block"
// ActionID: "title_input"
// Optional: false (required field)
//
// Example:
//
//	block := buildTitleBlock()
//	// block.Label.Text == "Title"
//	// block.Optional == false
func buildTitleBlock() *slack.InputBlock {
	return createTextInputBlock(
		BlockIDTitle,
		ActionIDTitleInput,
		LabelTitle,
		PlaceholderTitle,
		true,
		false,
	)
}

// buildThemeBlock creates the "Theme/Category" form field block.
// This is a required single-select dropdown for selecting the idea theme.
// Valid options come from constants.ValidThemeCategories.
//
// Returns an InputBlock with a SelectBlockElement.
// BlockID: "theme_block"
// ActionID: "theme_select"
// Optional: false (required field)
//
// Example:
//
//	block := buildThemeBlock()
//	// block.Label.Text == "Theme/Category"
//	// block.Optional == false
//	// len(element.Options) == 4
func buildThemeBlock() *slack.InputBlock {
	options := createOptions(constants.ValidThemeCategories)

	element := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		newPlainText(PlaceholderTheme),
		ActionIDThemeSelect,
		options...,
	)

	return slack.NewInputBlock(
		BlockIDTheme,
		newPlainText(LabelThemeCategory),
		nil,
		element,
	)
}

// buildProductAreaBlock creates the "Product Area" form field block.
// This is a required single-select dropdown for selecting the product area.
// Valid options come from constants.ValidProductAreas.
//
// Returns an InputBlock with a SelectBlockElement.
// BlockID: "product_area_block"
// ActionID: "product_area_select"
// Optional: false (required field)
//
// Example:
//
//	block := buildProductAreaBlock()
//	// block.Label.Text == "Product Area"
//	// block.Optional == false
func buildProductAreaBlock() *slack.InputBlock {
	options := createOptions(constants.ValidProductAreas)

	element := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		newPlainText(PlaceholderProductArea),
		ActionIDProductAreaSelect,
		options...,
	)

	return slack.NewInputBlock(
		BlockIDProductArea,
		newPlainText(LabelProductArea),
		nil,
		element,
	)
}

// buildCommentsBlock creates the "Comments" form field block.
// This is an optional multiline text input for additional context.
//
// Returns an InputBlock with a PlainTextInput element.
// BlockID: "comments_block"
// ActionID: "comments_input"
// Optional: true (optional field)
// Multiline: true (allows multiple lines)
//
// Example:
//
//	block := buildCommentsBlock()
//	// block.Label.Text == "Comments"
//	// block.Optional == true
//	// element.Multiline == true
func buildCommentsBlock() *slack.InputBlock {
	return createTextInputBlock(
		BlockIDComments,
		ActionIDCommentsInput,
		LabelComments,
		PlaceholderComments,
		false,
		true,
	)
}

// buildCustomerOrgBlock creates the "Customer Organization" form field block.
// This is an optional multi-select external dropdown for selecting customer organizations.
// Unlike static selects, external selects load their options dynamically as the user types.
// This allows supporting hundreds or thousands of customers without sending them all in the modal.
//
// Returns an InputBlock with a MultiSelectBlockElement configured for external option loading.
// BlockID: "client_org_block"
// ActionID: "client_org_select"
// Optional: true (optional field)
// MaxSelectedItems: 10 (enforced from constants.MaxCustomerOrgSelections)
//
// Note: Requires Slack app to have "Options Load URL" configured pointing to /slack/options endpoint.
// Without this configuration, the modal will fail to open with "invalid_arguments" error.
//
// Example:
//
//	block := buildCustomerOrgBlock()
//	// block.Label.Text == "Client Organization"
//	// block.Optional == true
//	// element.Type == "multi_external_select"
//	// *element.MaxSelectedItems == 10
func buildCustomerOrgBlock() *slack.InputBlock {
	element := slack.NewOptionsMultiSelectBlockElement(
		slack.MultiOptTypeExternal,
		newPlainText(PlaceholderCustomerOrg),
		ActionIDCustomerOrgSelect,
	)

	// Set maximum selections limit
	setMaxSelections(element, constants.MaxCustomerOrgSelections)

	block := slack.NewInputBlock(
		BlockIDCustomerOrg,
		newPlainText(LabelCustomerOrg),
		newPlainText(HintCustomerOrg),
		element,
	)

	// Mark as optional
	block.Optional = true

	return block
}

// createTextInputBlock creates a generic text input block (InputBlock).
// Used to build both single-line and multiline text input fields.
//
// Parameters:
//   - blockID: Slack block identifier
//   - actionID: Slack action identifier for field value extraction
//   - label: User-friendly field label displayed in the modal
//   - placeholder: Placeholder text shown in the input
//   - isRequired: If true, Optional = false (required field)
//   - isMultiline: If true, allows multiple lines of input
//
// Returns an InputBlock with a PlainTextInputBlockElement.
//
// Example (required, single-line):
//
//	block := createTextInputBlock(
//		"title_block",
//		"title_input",
//		"Title",
//		"Enter title...",
//		true,  // isRequired
//		false, // isMultiline
//	)
//	// block.Optional == false
//	// element.Multiline == false
//
// Example (optional, multiline):
//
//	block := createTextInputBlock(
//		"comments_block",
//		"comments_input",
//		"Comments",
//		"Add context...",
//		false, // isRequired
//		true,  // isMultiline
//	)
//	// block.Optional == true
//	// element.Multiline == true
func createTextInputBlock(
	blockID string,
	actionID string,
	label string,
	placeholder string,
	isRequired bool,
	isMultiline bool,
) *slack.InputBlock {
	element := slack.NewPlainTextInputBlockElement(
		newPlainText(placeholder),
		actionID,
	)
	element.Multiline = isMultiline

	block := slack.NewInputBlock(
		blockID,
		newPlainText(label),
		nil,
		element,
	)

	block.Optional = !isRequired

	return block
}

// createMultiSelectBlock creates a generic multi-select dropdown block (InputBlock).
// Supports both static and external multi-select elements.
//
// Parameters:
//   - blockID: Slack block identifier
//   - actionID: Slack action identifier for field value extraction
//   - label: User-friendly field label
//   - hint: Optional help text displayed below the field
//   - options: List of OptionBlockObjects for static selects (ignored for external selects)
//   - maxSelections: Maximum number of items that can be selected
//   - isRequired: If true, Optional = false (required field)
//
// Returns an InputBlock with a MultiSelectBlockElement.
// The element type (static vs external) is determined by the options provided.
// Empty options parameter results in an element with no initial options set.
//
// Example (with options):
//
//	options := []*slack.OptionBlockObject{
//		slack.NewOptionBlockObject("opt1", slack.NewTextBlockObject(slack.PlainTextType, "Option 1", false, false), nil),
//		slack.NewOptionBlockObject("opt2", slack.NewTextBlockObject(slack.PlainTextType, "Option 2", false, false), nil),
//	}
//	block := createMultiSelectBlock(
//		"test_block",
//		"test_action",
//		"Test Label",
//		"Select up to 5",
//		options,
//		5,
//		true,
//	)
//	// block.Optional == false
//	// *element.MaxSelectedItems == 5
//
// Example (without hint):
//
//	block := createMultiSelectBlock(
//		"test_block",
//		"test_action",
//		"Test Label",
//		"",      // empty hint
//		options,
//		5,
//		true,
//	)
//	// block.Hint == nil
func createMultiSelectBlock(
	blockID string,
	actionID string,
	label string,
	hint string,
	options []*slack.OptionBlockObject,
	maxSelections int,
	isRequired bool,
) *slack.InputBlock {
	element := slack.NewOptionsMultiSelectBlockElement(
		slack.MultiOptTypeStatic,
		newPlainText("Select..."),
		actionID,
		options...,
	)

	setMaxSelections(element, maxSelections)

	var hintObj *slack.TextBlockObject
	if hint != "" {
		hintObj = newPlainText(hint)
	}

	block := slack.NewInputBlock(
		blockID,
		newPlainText(label),
		hintObj,
		element,
	)

	block.Optional = !isRequired

	return block
}

// createOptions creates Slack OptionBlockObjects from a list of string values.
// Each value becomes both the option value and display text.
// Useful for building static select/multi-select dropdown options.
//
// Parameters:
//   - values: List of option values to convert
//
// Returns a slice of OptionBlockObjects ready to use in select elements.
// If values is empty, returns an empty slice.
//
// Example:
//
//	values := []string{"Option A", "Option B", "Option C"}
//	options := createOptions(values)
//	// Returns 3 OptionBlockObjects with:
//	// - Value: "Option A", Text: "Option A"
//	// - Value: "Option B", Text: "Option B"
//	// - Value: "Option C", Text: "Option C"
func createOptions(values []string) []*slack.OptionBlockObject {
	if len(values) == 0 {
		return []*slack.OptionBlockObject{}
	}

	options := make([]*slack.OptionBlockObject, 0, len(values))
	for _, value := range values {
		option := slack.NewOptionBlockObject(
			value,
			newPlainText(value),
			nil,
		)
		options = append(options, option)
	}

	return options
}

// newPlainText creates a Slack TextBlockObject of type "plain_text".
// Used for labels, placeholders, hints, and button text in modals.
// Plain text type disables markdown formatting (simple text only).
//
// Parameters:
//   - text: The text content to display
//
// Returns a TextBlockObject configured as plain text with emoji and verbatim disabled.
//
// Example:
//
//	text := newPlainText("Hello World")
//	// Returns TextBlockObject{
//	//   Type: "plain_text",
//	//   Text: "Hello World",
//	//   Emoji: false,
//	//   Verbatim: false,
//	// }
func newPlainText(text string) *slack.TextBlockObject {
	return slack.NewTextBlockObject(slack.PlainTextType, text, false, false)
}

// setMaxSelections sets the maximum number of items that can be selected
// in a multi-select block element.
//
// Parameters:
//   - element: The MultiSelectBlockElement to modify
//   - max: Maximum number of items allowed
//
// This modifies the element in-place by setting element.MaxSelectedItems.
//
// Example:
//
//	element := slack.NewOptionsMultiSelectBlockElement(...)
//	setMaxSelections(element, 10)
//	// element.MaxSelectedItems == &10
func setMaxSelections(element *slack.MultiSelectBlockElement, max int) {
	element.MaxSelectedItems = &max
}

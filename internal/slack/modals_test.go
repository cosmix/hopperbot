package slack

import (
	"testing"

	"github.com/rudderlabs/hopperbot/pkg/constants"
	"github.com/slack-go/slack"
)

// TestBuildSubmissionModal tests the main modal building function
func TestBuildSubmissionModal(t *testing.T) {
	modal := BuildSubmissionModal()

	if modal.Type != slack.VTModal {
		t.Errorf("modal type = %v, want %v", modal.Type, slack.VTModal)
	}

	if modal.CallbackID != ModalCallbackIDSubmitForm {
		t.Errorf("callback ID = %s, want %s", modal.CallbackID, ModalCallbackIDSubmitForm)
	}

	if modal.Title.Text != ModalTitle {
		t.Errorf("title text = %s, want %s", modal.Title.Text, ModalTitle)
	}

	if modal.Submit.Text != ModalSubmitText {
		t.Errorf("submit text = %s, want %s", modal.Submit.Text, ModalSubmitText)
	}

	if modal.Close.Text != ModalCancelText {
		t.Errorf("close text = %s, want %s", modal.Close.Text, ModalCancelText)
	}

	if len(modal.Blocks.BlockSet) != 5 {
		t.Errorf("number of blocks = %d, want 5", len(modal.Blocks.BlockSet))
	}
}

// TestBuildSubmissionModal_MultipleInvocations tests modal is consistent across invocations
func TestBuildSubmissionModal_MultipleInvocations(t *testing.T) {
	modal1 := BuildSubmissionModal()
	modal2 := BuildSubmissionModal()

	if len(modal1.Blocks.BlockSet) != len(modal2.Blocks.BlockSet) {
		t.Errorf("expected consistent block count, got %d and %d", len(modal1.Blocks.BlockSet), len(modal2.Blocks.BlockSet))
	}
}

// TestBuildTitleBlock tests title block creation
func TestBuildTitleBlock(t *testing.T) {
	block := buildTitleBlock()

	if block.BlockID != BlockIDTitle {
		t.Errorf("block ID = %s, want %s", block.BlockID, BlockIDTitle)
	}

	if block.Label.Text != LabelTitle {
		t.Errorf("label = %s, want %s", block.Label.Text, LabelTitle)
	}

	if block.Optional {
		t.Error("title block should be required (Optional = false)")
	}

	element, ok := block.Element.(*slack.PlainTextInputBlockElement)
	if !ok {
		t.Fatal("expected PlainTextInputBlockElement")
	}

	if element.ActionID != ActionIDTitleInput {
		t.Errorf("action ID = %s, want %s", element.ActionID, ActionIDTitleInput)
	}

	if element.Multiline {
		t.Error("title block should be single-line")
	}
}

// TestBuildThemeBlock tests theme block creation (single select)
func TestBuildThemeBlock(t *testing.T) {
	block := buildThemeBlock()

	if block.BlockID != BlockIDTheme {
		t.Errorf("block ID = %s, want %s", block.BlockID, BlockIDTheme)
	}

	if block.Label.Text != LabelThemeCategory {
		t.Errorf("label = %s, want %s", block.Label.Text, LabelThemeCategory)
	}

	if block.Optional {
		t.Error("theme block should be required (Optional = false)")
	}

	element, ok := block.Element.(*slack.SelectBlockElement)
	if !ok {
		t.Fatal("expected SelectBlockElement (single select)")
	}

	if element.ActionID != ActionIDThemeSelect {
		t.Errorf("action ID = %s, want %s", element.ActionID, ActionIDThemeSelect)
	}

	if len(element.Options) != len(constants.ValidThemeCategories) {
		t.Errorf("number of options = %d, want %d", len(element.Options), len(constants.ValidThemeCategories))
	}
}

// TestBuildProductAreaBlock tests product area block creation
func TestBuildProductAreaBlock(t *testing.T) {
	block := buildProductAreaBlock()

	if block.BlockID != BlockIDProductArea {
		t.Errorf("block ID = %s, want %s", block.BlockID, BlockIDProductArea)
	}

	if block.Label.Text != LabelProductArea {
		t.Errorf("label = %s, want %s", block.Label.Text, LabelProductArea)
	}

	if block.Optional {
		t.Error("product area block should be required (Optional = false)")
	}

	element, ok := block.Element.(*slack.SelectBlockElement)
	if !ok {
		t.Fatal("expected SelectBlockElement")
	}

	if element.ActionID != ActionIDProductAreaSelect {
		t.Errorf("action ID = %s, want %s", element.ActionID, ActionIDProductAreaSelect)
	}
}

// TestBuildCommentsBlock tests comments block creation
func TestBuildCommentsBlock(t *testing.T) {
	block := buildCommentsBlock()

	if block.BlockID != BlockIDComments {
		t.Errorf("block ID = %s, want %s", block.BlockID, BlockIDComments)
	}

	if block.Label.Text != LabelComments {
		t.Errorf("label = %s, want %s", block.Label.Text, LabelComments)
	}

	if !block.Optional {
		t.Error("comments block should be optional (Optional = true)")
	}

	element, ok := block.Element.(*slack.PlainTextInputBlockElement)
	if !ok {
		t.Fatal("expected PlainTextInputBlockElement")
	}

	if element.ActionID != ActionIDCommentsInput {
		t.Errorf("action ID = %s, want %s", element.ActionID, ActionIDCommentsInput)
	}

	if !element.Multiline {
		t.Error("comments block should be multiline")
	}
}

// TestBuildCustomerOrgBlock tests customer org block creation (external select)
func TestBuildCustomerOrgBlock(t *testing.T) {
	block := buildCustomerOrgBlock()

	if block.BlockID != BlockIDCustomerOrg {
		t.Errorf("block ID = %s, want %s", block.BlockID, BlockIDCustomerOrg)
	}

	if block.Label.Text != LabelCustomerOrg {
		t.Errorf("label = %s, want %s", block.Label.Text, LabelCustomerOrg)
	}

	if !block.Optional {
		t.Error("customer org block should be optional (Optional = true)")
	}

	element, ok := block.Element.(*slack.MultiSelectBlockElement)
	if !ok {
		t.Fatal("expected MultiSelectBlockElement")
	}

	if element.ActionID != ActionIDCustomerOrgSelect {
		t.Errorf("action ID = %s, want %s", element.ActionID, ActionIDCustomerOrgSelect)
	}

	// Verify it's using external select (not static select)
	if element.Type != slack.MultiOptTypeExternal {
		t.Errorf("element type = %s, want %s (external select)", element.Type, slack.MultiOptTypeExternal)
	}

	if element.MaxSelectedItems == nil || *element.MaxSelectedItems != constants.MaxCustomerOrgSelections {
		t.Errorf("max selections not set correctly")
	}
}

// TestCreateTextInputBlock tests text input block creation
func TestCreateTextInputBlock(t *testing.T) {
	block := createTextInputBlock(
		"test_block",
		"test_action",
		"Test Label",
		"Test Placeholder",
		true,
		false,
	)

	if block.BlockID != "test_block" {
		t.Errorf("block ID = %s, want test_block", block.BlockID)
	}

	if block.Optional {
		t.Error("required block should have Optional = false")
	}

	element, ok := block.Element.(*slack.PlainTextInputBlockElement)
	if !ok {
		t.Fatal("expected PlainTextInputBlockElement")
	}

	if element.Multiline {
		t.Error("single-line block should have Multiline = false")
	}
}

// TestCreateTextInputBlock_Optional tests optional text input block
func TestCreateTextInputBlock_Optional(t *testing.T) {
	block := createTextInputBlock(
		"test_block",
		"test_action",
		"Test Label",
		"Test Placeholder",
		false,
		false,
	)

	if !block.Optional {
		t.Error("optional block should have Optional = true")
	}
}

// TestCreateTextInputBlock_Multiline tests multiline text input block
func TestCreateTextInputBlock_Multiline(t *testing.T) {
	block := createTextInputBlock(
		"test_block",
		"test_action",
		"Test Label",
		"Test Placeholder",
		false,
		true,
	)

	element, ok := block.Element.(*slack.PlainTextInputBlockElement)
	if !ok {
		t.Fatal("expected PlainTextInputBlockElement")
	}

	if !element.Multiline {
		t.Error("multiline block should have Multiline = true")
	}
}

// TestCreateMultiSelectBlock tests multi-select block creation
func TestCreateMultiSelectBlock(t *testing.T) {
	options := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject("opt1", slack.NewTextBlockObject(slack.PlainTextType, "Option 1", false, false), nil),
		slack.NewOptionBlockObject("opt2", slack.NewTextBlockObject(slack.PlainTextType, "Option 2", false, false), nil),
	}

	block := createMultiSelectBlock(
		"test_block",
		"test_action",
		"Test Label",
		"Test Hint",
		options,
		5,
		true,
	)

	if block.BlockID != "test_block" {
		t.Errorf("block ID = %s, want test_block", block.BlockID)
	}

	if block.Optional {
		t.Error("required block should have Optional = false")
	}

	if block.Hint.Text != "Test Hint" {
		t.Errorf("hint text = %s, want Test Hint", block.Hint.Text)
	}

	element, ok := block.Element.(*slack.MultiSelectBlockElement)
	if !ok {
		t.Fatal("expected MultiSelectBlockElement")
	}

	if element.MaxSelectedItems == nil || *element.MaxSelectedItems != 5 {
		t.Errorf("max selections = %v, want 5", element.MaxSelectedItems)
	}
}

// TestCreateMultiSelectBlock_NoHint tests multi-select block without hint
func TestCreateMultiSelectBlock_NoHint(t *testing.T) {
	options := []*slack.OptionBlockObject{}

	block := createMultiSelectBlock(
		"test_block",
		"test_action",
		"Test Label",
		"",
		options,
		5,
		true,
	)

	if block.Hint != nil {
		t.Error("expected nil hint when hint text is empty")
	}
}

// TestCreateOptions tests option creation
func TestCreateOptions(t *testing.T) {
	values := []string{"Option A", "Option B", "Option C"}
	options := createOptions(values)

	if len(options) != len(values) {
		t.Errorf("number of options = %d, want %d", len(options), len(values))
	}

	for i, option := range options {
		if option.Value != values[i] {
			t.Errorf("option value = %s, want %s", option.Value, values[i])
		}

		if option.Text.Text != values[i] {
			t.Errorf("option text = %s, want %s", option.Text.Text, values[i])
		}
	}
}

// TestCreateOptions_Empty tests option creation with empty list
func TestCreateOptions_Empty(t *testing.T) {
	values := []string{}
	options := createOptions(values)

	if len(options) != 0 {
		t.Errorf("expected 0 options for empty values, got %d", len(options))
	}
}

// TestNewPlainText tests plain text creation
func TestNewPlainText(t *testing.T) {
	text := newPlainText("Test Text")

	if text.Type != slack.PlainTextType {
		t.Errorf("text type = %s, want %s", text.Type, slack.PlainTextType)
	}

	if text.Text != "Test Text" {
		t.Errorf("text content = %s, want Test Text", text.Text)
	}

	if text.Emoji != nil && *text.Emoji {
		t.Error("emoji should be false or nil")
	}

	if text.Verbatim {
		t.Error("verbatim should be false")
	}
}

// TestSetMaxSelections tests max selections setter
func TestSetMaxSelections(t *testing.T) {
	element := slack.NewOptionsMultiSelectBlockElement(
		slack.MultiOptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, "Select...", false, false),
		"test_action",
	)

	// Initial state may already have a value set - that's acceptable for this library

	setMaxSelections(element, 3)

	if element.MaxSelectedItems == nil || *element.MaxSelectedItems != 3 {
		t.Errorf("MaxSelectedItems = %v, want 3", element.MaxSelectedItems)
	}

	setMaxSelections(element, 10)

	if element.MaxSelectedItems == nil || *element.MaxSelectedItems != 10 {
		t.Errorf("MaxSelectedItems = %v, want 10", element.MaxSelectedItems)
	}
}

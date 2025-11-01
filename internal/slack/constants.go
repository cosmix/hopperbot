package slack

// Modal callback IDs
const (
	ModalCallbackIDSubmitForm = "submit_form_modal"
)

// Block IDs for modal form fields
const (
	BlockIDTitle       = "title_block"
	BlockIDTheme       = "theme_block"
	BlockIDProductArea = "product_area_block"
	BlockIDComments    = "comments_block"
	BlockIDCustomerOrg = "client_org_block" // Keep original ID for Slack compatibility
)

// Action IDs for modal form fields
const (
	ActionIDTitleInput        = "title_input"
	ActionIDThemeSelect       = "theme_select"
	ActionIDProductAreaSelect = "product_area_select"
	ActionIDCommentsInput     = "comments_input"
	ActionIDCustomerOrgSelect = "client_org_select" // Keep original ID for Slack compatibility
)

// Modal UI text
const (
	ModalTitle      = "Add Idea to Hopper" // Must be < 25 chars (Slack limit)
	ModalSubmitText = "Submit"
	ModalCancelText = "Cancel"
)

// Field labels
const (
	LabelTitle         = "Title"
	LabelThemeCategory = "Theme/Category"
	LabelProductArea   = "Product Area"
	LabelComments      = "Comments"
	LabelCustomerOrg   = "Client Organization" // Keep original label - Slack may have this cached
)

// Field placeholders
const (
	PlaceholderTitle       = "Enter a descriptive title"
	PlaceholderTheme       = "Select theme..."
	PlaceholderProductArea = "Select product area..."
	PlaceholderComments    = "Add any additional context or details..."
	PlaceholderCustomerOrg = "Select customers..."
)

// Field hints
const (
	HintCustomerOrg = "Select up to 10 customer organizations"
)

// Slack request headers
const (
	HeaderSlackRequestTimestamp = "X-Slack-Request-Timestamp"
	HeaderSlackSignature        = "X-Slack-Signature"
)

// Slack signature components
const (
	SignatureVersion = "v0"
	SignaturePrefix  = "v0="
)

// Interaction types
const (
	InteractionTypeViewSubmission = "view_submission"
)

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
	ModalSubmitText = "Submit"
	ModalCancelText = "Cancel"
)

// ModalTitles contains a list of witty titles that rotate each time the modal is opened.
// Each title is relevant to the three types of submissions:
// 1. New feature ideas
// 2. Feature improvements
// 3. Customer/market intelligence from sales or CS interactions
// All titles must be under 25 characters due to Slack API limits.
var ModalTitles = []string{
	"Share Your Intel",      // Customer/market intelligence
	"From the Field",        // Sales/CS insights from calls/events
	"Drop a Feature Idea",   // New feature requests
	"Heard in the Wild",     // Customer intelligence from the field
	"Idea Drop Zone",        // General feature ideas/improvements
	"Customer Wisdom",       // Insights from customer interactions
	"Ship Your Insight",     // General insights/improvements
}

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

package slack

import "testing"

// TestGetSelectedOption_NilPointerSafety tests that GetSelectedOption handles nil pointers safely
func TestGetSelectedOption_NilPointerSafety(t *testing.T) {
	tests := []struct {
		name      string
		state     *ViewState
		blockID   string
		actionID  string
		want      string
		wantError bool
	}{
		{
			name:      "nil ViewState Values",
			state:     &ViewState{Values: nil},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: true,
		},
		{
			name: "missing block ID",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"other_block": {},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: true,
		},
		{
			name: "missing action ID",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"other_action": {},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: true,
		},
		{
			name: "nil SelectedOption pointer - CRITICAL TEST",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type:           "static_select",
							SelectedOption: nil, // This is the critical case that could cause panic
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: false,
		},
		{
			name: "empty SelectedOption value",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type: "static_select",
							SelectedOption: &SelectedOption{
								Text:  OptionText{Type: "plain_text", Text: "Option"},
								Value: "", // Empty value
							},
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: false,
		},
		{
			name: "valid SelectedOption",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type: "static_select",
							SelectedOption: &SelectedOption{
								Text:  OptionText{Type: "plain_text", Text: "Product A"},
								Value: "product_a",
							},
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "product_a",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that GetSelectedOption doesn't panic
			// even with nil SelectedOption pointers
			got, err := tt.state.GetSelectedOption(tt.blockID, tt.actionID)
			if (err != nil) != tt.wantError {
				t.Errorf("GetSelectedOption() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("GetSelectedOption() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetValue_NilPointerSafety tests that GetValue handles nil pointers safely
func TestGetValue_NilPointerSafety(t *testing.T) {
	tests := []struct {
		name      string
		state     *ViewState
		blockID   string
		actionID  string
		want      string
		wantError bool
	}{
		{
			name:      "nil ViewState Values",
			state:     &ViewState{Values: nil},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: true,
		},
		{
			name: "nil Value pointer",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type:  "plain_text_input",
							Value: nil,
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "",
			wantError: false,
		},
		{
			name: "valid Value",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type:  "plain_text_input",
							Value: stringPtr("test value"),
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      "test value",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.state.GetValue(tt.blockID, tt.actionID)
			if (err != nil) != tt.wantError {
				t.Errorf("GetValue() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// stringPtr is a helper function to create string pointers for tests
func stringPtr(s string) *string {
	return &s
}

// TestInteractionPayload_Validate tests the Validate method
func TestInteractionPayload_Validate(t *testing.T) {
	tests := []struct {
		name      string
		payload   *InteractionPayload
		wantError bool
	}{
		{
			name: "valid payload",
			payload: &InteractionPayload{
				Type: "view_submission",
				User: User{ID: "U123", Username: "testuser"},
				Team: Team{ID: "T123", Domain: "test"},
			},
			wantError: false,
		},
		{
			name: "missing type",
			payload: &InteractionPayload{
				Type: "",
				User: User{ID: "U123", Username: "testuser"},
				Team: Team{ID: "T123", Domain: "test"},
			},
			wantError: true,
		},
		{
			name: "missing user ID",
			payload: &InteractionPayload{
				Type: "view_submission",
				User: User{ID: "", Username: "testuser"},
				Team: Team{ID: "T123", Domain: "test"},
			},
			wantError: true,
		},
		{
			name: "missing team ID",
			payload: &InteractionPayload{
				Type: "view_submission",
				User: User{ID: "U123", Username: "testuser"},
				Team: Team{ID: "", Domain: "test"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestStateValue_IsTextInput tests the IsTextInput method
func TestStateValue_IsTextInput(t *testing.T) {
	tests := []struct {
		name  string
		value StateValue
		want  bool
	}{
		{name: "plain text input", value: StateValue{Type: "plain_text_input"}, want: true},
		{name: "rich text input", value: StateValue{Type: "rich_text_input"}, want: true},
		{name: "static select", value: StateValue{Type: "static_select"}, want: false},
		{name: "multi static select", value: StateValue{Type: "multi_static_select"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsTextInput(); got != tt.want {
				t.Errorf("IsTextInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStateValue_IsSelect tests the IsSelect method
func TestStateValue_IsSelect(t *testing.T) {
	tests := []struct {
		name  string
		value StateValue
		want  bool
	}{
		{name: "static select", value: StateValue{Type: "static_select"}, want: true},
		{name: "external select", value: StateValue{Type: "external_select"}, want: true},
		{name: "users select", value: StateValue{Type: "users_select"}, want: true},
		{name: "conversations select", value: StateValue{Type: "conversations_select"}, want: true},
		{name: "channels select", value: StateValue{Type: "channels_select"}, want: true},
		{name: "multi static select", value: StateValue{Type: "multi_static_select"}, want: false},
		{name: "plain text input", value: StateValue{Type: "plain_text_input"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsSelect(); got != tt.want {
				t.Errorf("IsSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStateValue_IsMultiSelect tests the IsMultiSelect method
func TestStateValue_IsMultiSelect(t *testing.T) {
	tests := []struct {
		name  string
		value StateValue
		want  bool
	}{
		{name: "multi static select", value: StateValue{Type: "multi_static_select"}, want: true},
		{name: "multi external select", value: StateValue{Type: "multi_external_select"}, want: true},
		{name: "multi users select", value: StateValue{Type: "multi_users_select"}, want: true},
		{name: "multi conversations select", value: StateValue{Type: "multi_conversations_select"}, want: true},
		{name: "multi channels select", value: StateValue{Type: "multi_channels_select"}, want: true},
		{name: "static select", value: StateValue{Type: "static_select"}, want: false},
		{name: "plain text input", value: StateValue{Type: "plain_text_input"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsMultiSelect(); got != tt.want {
				t.Errorf("IsMultiSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetSelectedOptions tests the GetSelectedOptions method
func TestGetSelectedOptions(t *testing.T) {
	tests := []struct {
		name      string
		state     *ViewState
		blockID   string
		actionID  string
		want      []string
		wantError bool
	}{
		{
			name:      "nil ViewState Values",
			state:     &ViewState{Values: nil},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      nil,
			wantError: true,
		},
		{
			name: "empty selected options",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type:            "multi_static_select",
							SelectedOptions: []SelectedOption{},
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      []string{},
			wantError: false,
		},
		{
			name: "multiple selected options",
			state: &ViewState{
				Values: map[string]map[string]StateValue{
					"test_block": {
						"test_action": {
							Type: "multi_static_select",
							SelectedOptions: []SelectedOption{
								{Value: "option1", Text: OptionText{Type: "plain_text", Text: "Option 1"}},
								{Value: "option2", Text: OptionText{Type: "plain_text", Text: "Option 2"}},
								{Value: "option3", Text: OptionText{Type: "plain_text", Text: "Option 3"}},
							},
						},
					},
				},
			},
			blockID:   "test_block",
			actionID:  "test_action",
			want:      []string{"option1", "option2", "option3"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.state.GetSelectedOptions(tt.blockID, tt.actionID)
			if (err != nil) != tt.wantError {
				t.Errorf("GetSelectedOptions() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("GetSelectedOptions() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetSelectedOptions()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

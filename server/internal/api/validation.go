package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/hitl-sh/handoff-server/internal/models"
)

type selectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Color string `json:"color"`
}

func validateCreateRequest(input *models.CreateRequestInput) error {
	if input.ProcessingType != "time-sensitive" && input.ProcessingType != "deferred" {
		return fmt.Errorf("processing_type must be 'time-sensitive' or 'deferred'")
	}
	if input.Type != "markdown" && input.Type != "image" {
		return fmt.Errorf("type must be 'markdown' or 'image'")
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" || len(input.Title) > 200 {
		return fmt.Errorf("title is required and must be 1-200 characters")
	}
	input.RequestText = strings.TrimSpace(input.RequestText)
	if input.RequestText == "" || len(input.RequestText) > 10000 {
		return fmt.Errorf("request_text is required and must be 1-10000 characters")
	}
	if input.Type == "image" && (input.ImageURL == nil || *input.ImageURL == "") {
		return fmt.Errorf("image_url is required when type is 'image'")
	}
	if input.Priority != "" {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if !validPriorities[input.Priority] {
			return fmt.Errorf("priority must be 'low', 'medium', 'high', or 'critical'")
		}
	}

	validResponseTypes := map[string]bool{
		"text": true, "single_select": true, "multi_select": true,
		"rating": true, "number": true, "boolean": true,
	}
	if !validResponseTypes[input.ResponseType] {
		return fmt.Errorf("response_type must be one of: text, single_select, multi_select, rating, number, boolean")
	}

	if input.ResponseConfig == nil || len(input.ResponseConfig) == 0 {
		return fmt.Errorf("response_config is required")
	}

	if input.ProcessingType == "time-sensitive" {
		if input.TimeoutSeconds == nil {
			return fmt.Errorf("timeout_seconds is required for time-sensitive requests")
		}
		if *input.TimeoutSeconds < 60 || *input.TimeoutSeconds > 604800 {
			return fmt.Errorf("timeout_seconds must be between 60 and 604800")
		}
	}

	// Validate context JSON size
	if input.Context != nil && len(input.Context) > 50*1024 {
		return fmt.Errorf("context must be under 50KB")
	}

	return nil
}

func validateResponseConfig(responseType string, configJSON json.RawMessage) error {
	switch responseType {
	case "text":
		var cfg struct {
			Placeholder string `json:"placeholder"`
			MinLength   *int   `json:"min_length"`
			MaxLength   *int   `json:"max_length"`
			Required    *bool  `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid text config: %w", err)
		}
		if cfg.MinLength != nil && *cfg.MinLength < 0 {
			return fmt.Errorf("min_length must be >= 0")
		}
		if cfg.MaxLength != nil {
			if *cfg.MaxLength > 50000 {
				return fmt.Errorf("max_length must be <= 50000")
			}
			if cfg.MinLength != nil && *cfg.MaxLength < *cfg.MinLength {
				return fmt.Errorf("max_length must be >= min_length")
			}
		}

	case "single_select":
		var cfg struct {
			Options  []selectOption `json:"options"`
			Required *bool          `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid single_select config: %w", err)
		}
		if len(cfg.Options) < 2 || len(cfg.Options) > 20 {
			return fmt.Errorf("options must have 2-20 items")
		}
		if err := validateOptions(cfg.Options); err != nil {
			return err
		}

	case "multi_select":
		var cfg struct {
			Options       []selectOption `json:"options"`
			MinSelections *int           `json:"min_selections"`
			MaxSelections *int           `json:"max_selections"`
			Required      *bool          `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid multi_select config: %w", err)
		}
		if len(cfg.Options) < 2 || len(cfg.Options) > 20 {
			return fmt.Errorf("options must have 2-20 items")
		}
		if err := validateOptions(cfg.Options); err != nil {
			return err
		}
		if cfg.MinSelections != nil && (*cfg.MinSelections < 0 || *cfg.MinSelections > len(cfg.Options)) {
			return fmt.Errorf("min_selections must be 0-%d", len(cfg.Options))
		}
		if cfg.MaxSelections != nil {
			if *cfg.MaxSelections > len(cfg.Options) {
				return fmt.Errorf("max_selections must be <= %d", len(cfg.Options))
			}
			if cfg.MinSelections != nil && *cfg.MaxSelections < *cfg.MinSelections {
				return fmt.Errorf("max_selections must be >= min_selections")
			}
		}

	case "rating":
		var cfg struct {
			ScaleMin  *int              `json:"scale_min"`
			ScaleMax  *int              `json:"scale_max"`
			ScaleStep *float64          `json:"scale_step"`
			Labels    map[string]string `json:"labels"`
			Required  *bool             `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid rating config: %w", err)
		}
		if cfg.ScaleMax == nil {
			return fmt.Errorf("scale_max is required")
		}
		scaleMin := 1
		if cfg.ScaleMin != nil {
			scaleMin = *cfg.ScaleMin
		}
		if *cfg.ScaleMax <= scaleMin {
			return fmt.Errorf("scale_max must be > scale_min")
		}
		if *cfg.ScaleMax > 100 {
			return fmt.Errorf("scale_max must be <= 100")
		}

	case "number":
		var cfg struct {
			MinValue      *float64 `json:"min_value"`
			MaxValue      *float64 `json:"max_value"`
			DecimalPlaces *int     `json:"decimal_places"`
			AllowNegative *bool    `json:"allow_negative"`
			Required      *bool    `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid number config: %w", err)
		}
		if cfg.MaxValue == nil {
			return fmt.Errorf("max_value is required")
		}
		minVal := 0.0
		if cfg.MinValue != nil {
			minVal = *cfg.MinValue
		}
		if *cfg.MaxValue <= minVal {
			return fmt.Errorf("max_value must be > min_value")
		}
		if cfg.DecimalPlaces != nil && (*cfg.DecimalPlaces < 0 || *cfg.DecimalPlaces > 10) {
			return fmt.Errorf("decimal_places must be 0-10")
		}

	case "boolean":
		var cfg struct {
			TrueLabel  string `json:"true_label"`
			FalseLabel string `json:"false_label"`
			TrueColor  string `json:"true_color"`
			FalseColor string `json:"false_color"`
			Required   *bool  `json:"required"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return fmt.Errorf("invalid boolean config: %w", err)
		}
	}

	return nil
}

func validateResponseData(responseType string, configJSON, dataJSON json.RawMessage) error {
	switch responseType {
	case "text":
		var data string
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("response_data must be a string for text type")
		}

		var cfg struct {
			MinLength *int  `json:"min_length"`
			MaxLength *int  `json:"max_length"`
			Required  *bool `json:"required"`
		}
		json.Unmarshal(configJSON, &cfg)

		trimmed := strings.TrimSpace(data)
		if cfg.Required != nil && *cfg.Required && trimmed == "" {
			return fmt.Errorf("text response is required")
		}
		if cfg.MinLength != nil && len(trimmed) < *cfg.MinLength {
			return fmt.Errorf("text must be at least %d characters", *cfg.MinLength)
		}
		maxLen := 5000
		if cfg.MaxLength != nil {
			maxLen = *cfg.MaxLength
		}
		if len(trimmed) > maxLen {
			return fmt.Errorf("text must be at most %d characters", maxLen)
		}

	case "single_select":
		var data struct {
			SelectedValue string `json:"selected_value"`
			SelectedLabel string `json:"selected_label"`
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("invalid single_select response format")
		}

		var cfg struct {
			Options []selectOption `json:"options"`
		}
		json.Unmarshal(configJSON, &cfg)

		found := false
		for _, opt := range cfg.Options {
			if opt.Value == data.SelectedValue {
				if opt.Label != data.SelectedLabel {
					return fmt.Errorf("selected_label does not match the option label")
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("selected_value is not a valid option")
		}

	case "multi_select":
		var data struct {
			SelectedValues []string `json:"selected_values"`
			SelectedLabels []string `json:"selected_labels"`
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("invalid multi_select response format")
		}

		if len(data.SelectedValues) != len(data.SelectedLabels) {
			return fmt.Errorf("selected_values and selected_labels must have the same length")
		}

		var cfg struct {
			Options       []selectOption `json:"options"`
			MinSelections *int           `json:"min_selections"`
			MaxSelections *int           `json:"max_selections"`
			Required      *bool          `json:"required"`
		}
		json.Unmarshal(configJSON, &cfg)

		optMap := make(map[string]string)
		for _, opt := range cfg.Options {
			optMap[opt.Value] = opt.Label
		}

		for i, v := range data.SelectedValues {
			label, ok := optMap[v]
			if !ok {
				return fmt.Errorf("selected_value '%s' is not a valid option", v)
			}
			if label != data.SelectedLabels[i] {
				return fmt.Errorf("selected_label for '%s' does not match", v)
			}
		}

		minSel := 1
		if cfg.MinSelections != nil {
			minSel = *cfg.MinSelections
		}
		maxSel := len(cfg.Options)
		if cfg.MaxSelections != nil {
			maxSel = *cfg.MaxSelections
		}
		if len(data.SelectedValues) < minSel {
			return fmt.Errorf("must select at least %d options", minSel)
		}
		if len(data.SelectedValues) > maxSel {
			return fmt.Errorf("must select at most %d options", maxSel)
		}

		if cfg.Required != nil && *cfg.Required && len(data.SelectedValues) == 0 {
			return fmt.Errorf("at least one selection is required")
		}

	case "rating":
		var data struct {
			Rating      *float64 `json:"rating"`
			RatingLabel string   `json:"rating_label"`
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("invalid rating response format")
		}
		if data.Rating == nil {
			return fmt.Errorf("rating is required")
		}

		var cfg struct {
			ScaleMin  *int     `json:"scale_min"`
			ScaleMax  *int     `json:"scale_max"`
			ScaleStep *float64 `json:"scale_step"`
		}
		json.Unmarshal(configJSON, &cfg)

		scaleMin := 1.0
		if cfg.ScaleMin != nil {
			scaleMin = float64(*cfg.ScaleMin)
		}
		scaleMax := 5.0
		if cfg.ScaleMax != nil {
			scaleMax = float64(*cfg.ScaleMax)
		}
		scaleStep := 1.0
		if cfg.ScaleStep != nil {
			scaleStep = *cfg.ScaleStep
		}

		if *data.Rating < scaleMin || *data.Rating > scaleMax {
			return fmt.Errorf("rating must be between %.0f and %.0f", scaleMin, scaleMax)
		}

		// Check step alignment
		diff := *data.Rating - scaleMin
		steps := diff / scaleStep
		if math.Abs(steps-math.Round(steps)) > 1e-9 {
			return fmt.Errorf("rating must fall on a valid step")
		}

	case "number":
		var data struct {
			Number         *float64 `json:"number"`
			FormattedValue string   `json:"formatted_value"`
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("invalid number response format")
		}
		if data.Number == nil {
			return fmt.Errorf("number is required")
		}

		var cfg struct {
			MinValue      *float64 `json:"min_value"`
			MaxValue      *float64 `json:"max_value"`
			DecimalPlaces *int     `json:"decimal_places"`
			AllowNegative *bool    `json:"allow_negative"`
		}
		json.Unmarshal(configJSON, &cfg)

		minVal := 0.0
		if cfg.MinValue != nil {
			minVal = *cfg.MinValue
		}
		if *data.Number < minVal {
			return fmt.Errorf("number must be >= %v", minVal)
		}
		if cfg.MaxValue != nil && *data.Number > *cfg.MaxValue {
			return fmt.Errorf("number must be <= %v", *cfg.MaxValue)
		}
		if cfg.AllowNegative != nil && !*cfg.AllowNegative && *data.Number < 0 {
			return fmt.Errorf("negative numbers are not allowed")
		}

		decimalPlaces := 2
		if cfg.DecimalPlaces != nil {
			decimalPlaces = *cfg.DecimalPlaces
		}
		multiplied := *data.Number * math.Pow(10, float64(decimalPlaces))
		if math.Abs(multiplied-math.Round(multiplied)) > 1e-9 {
			return fmt.Errorf("number must have at most %d decimal places", decimalPlaces)
		}

	case "boolean":
		var data struct {
			Boolean      *bool  `json:"boolean"`
			BooleanLabel string `json:"boolean_label"`
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return fmt.Errorf("invalid boolean response format")
		}
		if data.Boolean == nil {
			return fmt.Errorf("boolean is required")
		}

		var cfg struct {
			TrueLabel  string `json:"true_label"`
			FalseLabel string `json:"false_label"`
		}
		json.Unmarshal(configJSON, &cfg)

		if cfg.TrueLabel == "" {
			cfg.TrueLabel = "Yes"
		}
		if cfg.FalseLabel == "" {
			cfg.FalseLabel = "No"
		}

		expectedLabel := cfg.FalseLabel
		if *data.Boolean {
			expectedLabel = cfg.TrueLabel
		}
		if data.BooleanLabel != expectedLabel {
			return fmt.Errorf("boolean_label must be '%s'", expectedLabel)
		}
	}

	return nil
}

func validateOptions(options []selectOption) error {
	seen := make(map[string]bool)
	for _, opt := range options {
		opt.Value = strings.TrimSpace(opt.Value)
		opt.Label = strings.TrimSpace(opt.Label)
		if opt.Value == "" || len(opt.Value) > 100 {
			return fmt.Errorf("each option value must be 1-100 characters")
		}
		if opt.Label == "" || len(opt.Label) > 100 {
			return fmt.Errorf("each option label must be 1-100 characters")
		}
		if seen[opt.Value] {
			return fmt.Errorf("duplicate option value: %s", opt.Value)
		}
		seen[opt.Value] = true
	}
	return nil
}

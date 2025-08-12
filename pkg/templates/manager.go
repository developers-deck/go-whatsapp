package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type Template struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Content       string                 `json:"content"`
	Variables     []Variable             `json:"variables"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	Language      string                 `json:"language"`
	Version       string                 `json:"version"`
	IsActive      bool                   `json:"is_active"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	UsageCount    int                    `json:"usage_count"`
	LastUsedAt    *time.Time             `json:"last_used_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
	Conditions    []Condition            `json:"conditions,omitempty"`
	Transformers  []Transformer          `json:"transformers,omitempty"`
	Validations   []Validation           `json:"validations,omitempty"`
	Scheduling    *ScheduleConfig        `json:"scheduling,omitempty"`
}

type Variable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // text, number, date, email, phone, url, select, boolean
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description,omitempty"`
	Options      []string    `json:"options,omitempty"` // for select type
	Validation   string      `json:"validation,omitempty"` // regex pattern
	Format       string      `json:"format,omitempty"` // date format, number format, etc.
}

type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value    interface{} `json:"value"`
	Action   string      `json:"action"` // show, hide, require, optional
}

type Transformer struct {
	Variable string `json:"variable"`
	Type     string `json:"type"` // uppercase, lowercase, capitalize, format_date, format_number
	Options  map[string]interface{} `json:"options,omitempty"`
}

type Validation struct {
	Variable string `json:"variable"`
	Rule     string `json:"rule"` // required, min_length, max_length, regex, email, phone
	Value    interface{} `json:"value,omitempty"`
	Message  string `json:"message"`
}

type ScheduleConfig struct {
	Enabled   bool      `json:"enabled"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	TimeZone  string    `json:"timezone"`
	Recurring bool      `json:"recurring"`
	Frequency string    `json:"frequency"` // daily, weekly, monthly
}

type TemplateVersion struct {
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	Changes   string    `json:"changes"`
}

type RenderContext struct {
	Variables   map[string]interface{} `json:"variables"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IP          string                 `json:"ip,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Language    string                 `json:"language,omitempty"`
	Timezone    string                 `json:"timezone,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TemplateManager struct {
	templatesPath string
	versionsPath  string
	templates     map[string]*Template
	versions      map[string][]TemplateVersion
	funcMap       textTemplate.FuncMap
}

func NewTemplateManager() *TemplateManager {
	templatesPath := filepath.Join(config.PathStorages, "templates")
	versionsPath := filepath.Join(config.PathStorages, "template_versions")
	os.MkdirAll(templatesPath, 0755)
	os.MkdirAll(versionsPath, 0755)

	tm := &TemplateManager{
	templatesPath: templatesPath,
	versionsPath:  versionsPath,
	templates:     make(map[string]*Template),
	versions:      make(map[string][]TemplateVersion),
}
tm.funcMap = tm.createFuncMap()

	// Load existing templates and versions
	tm.loadTemplates()
	tm.loadVersions()
	
	// Create default templates if none exist
	if len(tm.templates) == 0 {
		tm.createDefaultTemplates()
	}

	return tm
}

// CreateTemplate creates a new message template
func (tm *TemplateManager) CreateTemplate(name, description, content, category string) (*Template, error) {
	return tm.CreateAdvancedTemplate(&Template{
		Name:        name,
		Description: description,
		Content:     content,
		Category:    category,
		Language:    "en",
		Version:     "1.0.0",
		IsActive:    true,
		Tags:        []string{},
		Metadata:    make(map[string]interface{}),
	})
}

// CreateAdvancedTemplate creates a new advanced template with full configuration
func (tm *TemplateManager) CreateAdvancedTemplate(template *Template) (*Template, error) {
	if template.Name == "" || template.Content == "" {
		return nil, fmt.Errorf("name and content are required")
	}

	// Generate unique ID
	template.ID = tm.generateTemplateID(template.Name)

	// Extract and analyze variables from content
	template.Variables = tm.extractAdvancedVariables(template.Content)

	// Set defaults
	if template.Language == "" {
		template.Language = "en"
	}
	if template.Version == "" {
		template.Version = "1.0.0"
	}
	if template.Tags == nil {
		template.Tags = []string{}
	}
	if template.Metadata == nil {
		template.Metadata = make(map[string]interface{})
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.UsageCount = 0

	// Validate template
	if err := tm.validateTemplate(template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	// Save template
	if err := tm.saveTemplate(template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	// Create initial version
	tm.createVersion(template.ID, template.Content, "system", "Initial version")

	tm.templates[template.ID] = template
	logrus.Infof("[TEMPLATES] Created advanced template: %s (%s)", template.Name, template.ID)

	return template, nil
}

// GetTemplate retrieves a template by ID
func (tm *TemplateManager) GetTemplate(id string) (*Template, error) {
	template, exists := tm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return template, nil
}

// ListTemplates returns all templates, optionally filtered by category
func (tm *TemplateManager) ListTemplates(category string) []*Template {
	var templates []*Template
	
	for _, template := range tm.templates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}

	return templates
}

// UpdateTemplate updates an existing template
func (tm *TemplateManager) UpdateTemplate(id string, name, description, content, category string) (*Template, error) {
	template, exists := tm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	// Update fields
	if name != "" {
		template.Name = name
	}
	if description != "" {
		template.Description = description
	}
	if content != "" {
		template.Content = content
		template.Variables = tm.extractAdvancedVariables(content)
	}
	if category != "" {
		template.Category = category
	}
	
	template.UpdatedAt = time.Now()

	// Save updated template
	if err := tm.saveTemplate(template); err != nil {
		return nil, fmt.Errorf("failed to save updated template: %w", err)
	}

	logrus.Infof("[TEMPLATES] Updated template: %s (%s)", template.Name, id)
	return template, nil
}

// DeleteTemplate removes a template
func (tm *TemplateManager) DeleteTemplate(id string) error {
	template, exists := tm.templates[id]
	if !exists {
		return fmt.Errorf("template not found: %s", id)
	}

	// Remove file
	filePath := filepath.Join(tm.templatesPath, id+".json")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove template file: %w", err)
	}

	// Remove from memory
	delete(tm.templates, id)
	
	logrus.Infof("[TEMPLATES] Deleted template: %s (%s)", template.Name, id)
	return nil
}

// RenderTemplate renders a template with provided variables (backward compatibility)
func (tm *TemplateManager) RenderTemplate(id string, variables map[string]string) (string, error) {
	// Convert string map to interface map
	vars := make(map[string]interface{})
	for k, v := range variables {
		vars[k] = v
	}

	context := &RenderContext{
		Variables: vars,
		Timestamp: time.Now(),
		Language:  "en",
	}

	return tm.RenderAdvancedTemplate(id, context)
}

// RenderAdvancedTemplate renders a template with advanced context and features
func (tm *TemplateManager) RenderAdvancedTemplate(id string, context *RenderContext) (string, error) {
	tmpl, exists := tm.templates[id]
	if !exists {
		return "", fmt.Errorf("template not found: %s", id)
	}

	if !tmpl.IsActive {
		return "", fmt.Errorf("template is inactive: %s", id)
	}

	// Check scheduling constraints
	if tmpl.Scheduling != nil && tmpl.Scheduling.Enabled {
		if !tm.isTemplateScheduleValid(tmpl.Scheduling) {
			return "", fmt.Errorf("template is not available at this time")
		}
	}

	// Validate required variables
	if err := tm.validateRenderContext(tmpl, context); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// Apply transformers
	transformedVars := tm.applyTransformers(tmpl.Transformers, context.Variables)
	context.Variables = transformedVars

	// Set default values for missing variables
	tm.setDefaultValues(tmpl.Variables, context.Variables)

	// Parse and execute template
	goTemplate, err := textTemplate.New(tmpl.ID).Funcs(tm.funcMap).Parse(tmpl.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := goTemplate.Execute(&result, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Update usage statistics
	now := time.Now()
	tmpl.UsageCount++
	tmpl.LastUsedAt = &now
	tmpl.UpdatedAt = now
	tm.saveTemplate(tmpl)

	return result.String(), nil
}

// GetTemplateStats returns usage statistics
func (tm *TemplateManager) GetTemplateStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_templates": len(tm.templates),
		"categories":      make(map[string]int),
		"most_used":       "",
		"total_usage":     0,
	}

	categories := make(map[string]int)
	var mostUsed *Template
	totalUsage := 0

	for _, template := range tm.templates {
		// Count by category
		categories[template.Category]++
		
		// Track most used
		if mostUsed == nil || template.UsageCount > mostUsed.UsageCount {
			mostUsed = template
		}
		
		totalUsage += template.UsageCount
	}

	stats["categories"] = categories
	stats["total_usage"] = totalUsage
	
	if mostUsed != nil {
		stats["most_used"] = map[string]interface{}{
			"id":          mostUsed.ID,
			"name":        mostUsed.Name,
			"usage_count": mostUsed.UsageCount,
		}
	}

	return stats
}

// Advanced template methods

// CloneTemplate creates a copy of an existing template
func (tm *TemplateManager) CloneTemplate(id, newName string) (*Template, error) {
	original, exists := tm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	clone := *original // Copy struct
	clone.ID = tm.generateTemplateID(newName)
	clone.Name = newName
	clone.Version = "1.0.0"
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = time.Now()
	clone.UsageCount = 0
	clone.LastUsedAt = nil

	if err := tm.saveTemplate(&clone); err != nil {
		return nil, fmt.Errorf("failed to save cloned template: %w", err)
	}

	tm.templates[clone.ID] = &clone
	tm.createVersion(clone.ID, clone.Content, "system", "Cloned from "+original.Name)

	return &clone, nil
}

// GetTemplateVersions returns all versions of a template
func (tm *TemplateManager) GetTemplateVersions(id string) ([]TemplateVersion, error) {
	if _, exists := tm.templates[id]; !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	versions, exists := tm.versions[id]
	if !exists {
		return []TemplateVersion{}, nil
	}

	return versions, nil
}

// RestoreTemplateVersion restores a template to a specific version
func (tm *TemplateManager) RestoreTemplateVersion(id, version string) error {
	tmpl, exists := tm.templates[id]
	if !exists {
		return fmt.Errorf("template not found: %s", id)
	}

	versions, exists := tm.versions[id]
	if !exists {
		return fmt.Errorf("no versions found for template: %s", id)
	}

	var targetVersion *TemplateVersion
	for _, v := range versions {
		if v.Version == version {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		return fmt.Errorf("version not found: %s", version)
	}

	// Create backup of current version
	tm.createVersion(id, tmpl.Content, "system", "Backup before restore to "+version)

	// Restore content
	tmpl.Content = targetVersion.Content
	tmpl.Variables = tm.extractAdvancedVariables(tmpl.Content)
	tmpl.UpdatedAt = time.Now()

	return tm.saveTemplate(tmpl)
}

// SearchTemplates searches templates by various criteria
func (tm *TemplateManager) SearchTemplates(query string, filters map[string]interface{}) []*Template {
	var results []*Template
	query = strings.ToLower(query)

	for _, tmpl := range tm.templates {
		if tm.matchesSearchCriteria(tmpl, query, filters) {
			results = append(results, tmpl)
		}
	}

	return results
}

// BulkUpdateTemplates updates multiple templates at once
func (tm *TemplateManager) BulkUpdateTemplates(updates map[string]map[string]interface{}) error {
	for id, updateData := range updates {
		tmpl, exists := tm.templates[id]
		if !exists {
			continue
		}

		// Apply updates
		if name, ok := updateData["name"].(string); ok && name != "" {
			tmpl.Name = name
		}
		if category, ok := updateData["category"].(string); ok {
			tmpl.Category = category
		}
		if isActive, ok := updateData["is_active"].(bool); ok {
			tmpl.IsActive = isActive
		}
		if tags, ok := updateData["tags"].([]string); ok {
			tmpl.Tags = tags
		}

		tmpl.UpdatedAt = time.Now()
		tm.saveTemplate(tmpl)
	}

	return nil
}

// Private methods

func (tm *TemplateManager) generateTemplateID(name string) string {
	// Create ID from name + timestamp
	cleanName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	cleanName = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(cleanName, "")
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s", cleanName, timestamp)
}

func (tm *TemplateManager) extractAdvancedVariables(content string) []Variable {
	var variables []Variable
	variableMap := make(map[string]bool)

	// Find all {{.Variables.variable}} patterns (Go template format)
	re := regexp.MustCompile(`\{\{\.Variables\.(\w+)(?:\s*\|\s*(\w+))?\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		varName := match[1]
		varType := "text" // default type
		
		if len(match) > 2 && match[2] != "" {
			varType = match[2]
		}

		if !variableMap[varName] {
			variables = append(variables, Variable{
				Name:        varName,
				Type:        varType,
				Required:    true,
				Description: fmt.Sprintf("Variable: %s", varName),
			})
			variableMap[varName] = true
		}
	}

	// Also find simple {{variable}} patterns for backward compatibility
	simpleRe := regexp.MustCompile(`\{\{(\w+)\}\}`)
	simpleMatches := simpleRe.FindAllStringSubmatch(content, -1)

	for _, match := range simpleMatches {
		varName := match[1]
		if !variableMap[varName] {
			variables = append(variables, Variable{
				Name:        varName,
				Type:        "text",
				Required:    true,
				Description: fmt.Sprintf("Variable: %s", varName),
			})
			variableMap[varName] = true
		}
	}

	return variables
}

func (tm *TemplateManager) validateTemplate(template *Template) error {
	// Create a custom template with required functions
	tmpl := textTemplate.New(template.ID)
	
	// Add custom template functions
	tmpl = tmpl.Funcs(textTemplate.FuncMap{
		"default": func(value interface{}, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"formatNumber": func(value interface{}, format string) string {
			// Simple number formatting
			switch v := value.(type) {
			case float64:
				if format == "currency" {
					return fmt.Sprintf("$%.2f", v)
				}
				return fmt.Sprintf("%.2f", v)
			case int:
				if format == "currency" {
					return fmt.Sprintf("$%d.00", v)
				}
				return fmt.Sprintf("%d", v)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		"formatDate": func(value interface{}, format string) string {
			// Simple date formatting
			switch v := value.(type) {
			case time.Time:
				return v.Format(format)
			case string:
				if t, err := time.Parse("2006-01-02", v); err == nil {
					return t.Format(format)
				}
				return v
			default:
				return fmt.Sprintf("%v", v)
			}
		},
	})
	
	// Parse the template content
	_, err := tmpl.Parse(template.Content)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	// Validate variables
	for _, variable := range template.Variables {
		if variable.Name == "" {
			return fmt.Errorf("variable name cannot be empty")
		}
		if !tm.isValidVariableType(variable.Type) {
			return fmt.Errorf("invalid variable type: %s", variable.Type)
		}
	}

	// Validate conditions
	for _, condition := range template.Conditions {
		if !tm.isValidOperator(condition.Operator) {
			return fmt.Errorf("invalid condition operator: %s", condition.Operator)
		}
	}

	return nil
}

func (tm *TemplateManager) validateRenderContext(tmpl *Template, context *RenderContext) error {
	// Check required variables
	for _, variable := range tmpl.Variables {
		if variable.Required {
			if _, exists := context.Variables[variable.Name]; !exists {
				return fmt.Errorf("required variable missing: %s", variable.Name)
			}
		}

		// Validate variable type and format
		if value, exists := context.Variables[variable.Name]; exists {
			if err := tm.validateVariableValue(variable, value); err != nil {
				return fmt.Errorf("invalid value for variable %s: %w", variable.Name, err)
			}
		}
	}

	// Run custom validations
	for _, validation := range tmpl.Validations {
		if err := tm.runValidation(validation, context.Variables); err != nil {
			return err
		}
	}

	return nil
}

func (tm *TemplateManager) validateVariableValue(variable Variable, value interface{}) error {
	switch variable.Type {
	case "email":
		if str, ok := value.(string); ok {
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(str) {
				return fmt.Errorf("invalid email format")
			}
		}
	case "phone":
		if str, ok := value.(string); ok {
			phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
			if !phoneRegex.MatchString(str) {
				return fmt.Errorf("invalid phone format")
			}
		}
	case "url":
		if str, ok := value.(string); ok {
			urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
			if !urlRegex.MatchString(str) {
				return fmt.Errorf("invalid URL format")
			}
		}
	case "number":
		if _, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64); err != nil {
			return fmt.Errorf("invalid number format")
		}
	}

	// Custom regex validation
	if variable.Validation != "" {
		if str, ok := value.(string); ok {
			regex, err := regexp.Compile(variable.Validation)
			if err != nil {
				return fmt.Errorf("invalid validation regex")
			}
			if !regex.MatchString(str) {
				return fmt.Errorf("value does not match validation pattern")
			}
		}
	}

	return nil
}

func (tm *TemplateManager) saveTemplate(template *Template) error {
	filePath := filepath.Join(tm.templatesPath, template.ID+".json")
	
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0644)
}

func (tm *TemplateManager) loadTemplates() {
	pattern := filepath.Join(tm.templatesPath, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logrus.Errorf("[TEMPLATES] Failed to load templates: %v", err)
		return
	}

	for _, filePath := range matches {
		data, err := os.ReadFile(filePath)
		if err != nil {
			logrus.Errorf("[TEMPLATES] Failed to read template file %s: %v", filePath, err)
			continue
		}

		var template Template
		if err := json.Unmarshal(data, &template); err != nil {
			logrus.Errorf("[TEMPLATES] Failed to unmarshal template file %s: %v", filePath, err)
			continue
		}

		tm.templates[template.ID] = &template
	}

	logrus.Infof("[TEMPLATES] Loaded %d templates", len(tm.templates))
}

func (tm *TemplateManager) createFuncMap() textTemplate.FuncMap {
	return textTemplate.FuncMap{
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"trim":       strings.TrimSpace,
		"now":        time.Now,
		"formatDate": tm.formatDate,
		"formatNumber": tm.formatNumber,
		"default":    tm.defaultValue,
		"contains":   strings.Contains,
		"replace":    strings.ReplaceAll,
		"substr":     tm.substr,
		"add":        tm.add,
		"multiply":   tm.multiply,
		"divide":     tm.divide,
		"modulo":     tm.modulo,
		"eq":         tm.eq,
		"ne":         tm.ne,
		"gt":         tm.gt,
		"lt":         tm.lt,
		"gte":        tm.gte,
		"lte":        tm.lte,
		"and":        tm.and,
		"or":         tm.or,
		"not":        tm.not,
		"join":       strings.Join,
		"split":      strings.Split,
		"len":        tm.length,
		"first":      tm.first,
		"last":       tm.last,
		"slice":      tm.slice,
	}
}

// Template helper functions
func (tm *TemplateManager) formatDate(format string, date interface{}) string {
	var t time.Time
	switch v := date.(type) {
	case time.Time:
		t = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return v
		}
		t = parsed
	default:
		return fmt.Sprintf("%v", date)
	}
	return t.Format(format)
}

func (tm *TemplateManager) formatNumber(format string, number interface{}) string {
	switch format {
	case "currency":
		if f, err := strconv.ParseFloat(fmt.Sprintf("%v", number), 64); err == nil {
			return fmt.Sprintf("$%.2f", f)
		}
	case "percent":
		if f, err := strconv.ParseFloat(fmt.Sprintf("%v", number), 64); err == nil {
			return fmt.Sprintf("%.1f%%", f*100)
		}
	}
	return fmt.Sprintf("%v", number)
}

func (tm *TemplateManager) defaultValue(defaultVal, value interface{}) interface{} {
	if value == nil || value == "" {
		return defaultVal
	}
	return value
}

func (tm *TemplateManager) substr(start, length int, str string) string {
	if start >= len(str) {
		return ""
	}
	end := start + length
	if end > len(str) {
		end = len(str)
	}
	return str[start:end]
}

func (tm *TemplateManager) add(a, b interface{}) float64 {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat + bFloat
}

func (tm *TemplateManager) multiply(a, b interface{}) float64 {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat * bFloat
}

func (tm *TemplateManager) divide(a, b interface{}) float64 {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	if bFloat == 0 {
		return 0
	}
	return aFloat / bFloat
}

func (tm *TemplateManager) modulo(a, b interface{}) int {
	aInt, _ := strconv.Atoi(fmt.Sprintf("%v", a))
	bInt, _ := strconv.Atoi(fmt.Sprintf("%v", b))
	if bInt == 0 {
		return 0
	}
	return aInt % bInt
}

func (tm *TemplateManager) eq(a, b interface{}) bool { return a == b }
func (tm *TemplateManager) ne(a, b interface{}) bool { return a != b }
func (tm *TemplateManager) gt(a, b interface{}) bool {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat > bFloat
}
func (tm *TemplateManager) lt(a, b interface{}) bool {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat < bFloat
}
func (tm *TemplateManager) gte(a, b interface{}) bool {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat >= bFloat
}
func (tm *TemplateManager) lte(a, b interface{}) bool {
	aFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
	bFloat, _ := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
	return aFloat <= bFloat
}

func (tm *TemplateManager) and(a, b bool) bool { return a && b }
func (tm *TemplateManager) or(a, b bool) bool  { return a || b }
func (tm *TemplateManager) not(a bool) bool    { return !a }

func (tm *TemplateManager) length(v interface{}) int {
	switch val := v.(type) {
	case string:
		return len(val)
	case []interface{}:
		return len(val)
	case map[string]interface{}:
		return len(val)
	default:
		return 0
	}
}

func (tm *TemplateManager) first(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[0]
		}
	case string:
		if len(val) > 0 {
			return string(val[0])
		}
	}
	return nil
}

func (tm *TemplateManager) last(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[len(val)-1]
		}
	case string:
		if len(val) > 0 {
			return string(val[len(val)-1])
		}
	}
	return nil
}

func (tm *TemplateManager) slice(start, end int, v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if start >= 0 && end <= len(val) && start <= end {
			return val[start:end]
		}
	case string:
		if start >= 0 && end <= len(val) && start <= end {
			return val[start:end]
		}
	}
	return v
}

// Additional helper methods
func (tm *TemplateManager) applyTransformers(transformers []Transformer, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range variables {
		result[k] = v
	}

	for _, transformer := range transformers {
		if value, exists := result[transformer.Variable]; exists {
			switch transformer.Type {
			case "uppercase":
				if str, ok := value.(string); ok {
					result[transformer.Variable] = strings.ToUpper(str)
				}
			case "lowercase":
				if str, ok := value.(string); ok {
					result[transformer.Variable] = strings.ToLower(str)
				}
			case "capitalize":
				if str, ok := value.(string); ok {
					result[transformer.Variable] = strings.Title(strings.ToLower(str))
				}
			case "format_date":
				if format, ok := transformer.Options["format"].(string); ok {
					result[transformer.Variable] = tm.formatDate(format, value)
				}
			case "format_number":
				if format, ok := transformer.Options["format"].(string); ok {
					result[transformer.Variable] = tm.formatNumber(format, value)
				}
			}
		}
	}

	return result
}

func (tm *TemplateManager) setDefaultValues(variables []Variable, context map[string]interface{}) {
	for _, variable := range variables {
		if _, exists := context[variable.Name]; !exists && variable.DefaultValue != nil {
			context[variable.Name] = variable.DefaultValue
		}
	}
}

func (tm *TemplateManager) isTemplateScheduleValid(schedule *ScheduleConfig) bool {
	now := time.Now()
	return now.After(schedule.StartDate) && now.Before(schedule.EndDate)
}

func (tm *TemplateManager) isValidVariableType(varType string) bool {
	validTypes := []string{"text", "number", "date", "email", "phone", "url", "select", "boolean"}
	for _, valid := range validTypes {
		if varType == valid {
			return true
		}
	}
	return false
}

func (tm *TemplateManager) isValidOperator(operator string) bool {
	validOperators := []string{"eq", "ne", "gt", "lt", "contains", "regex"}
	for _, valid := range validOperators {
		if operator == valid {
			return true
		}
	}
	return false
}

func (tm *TemplateManager) runValidation(validation Validation, variables map[string]interface{}) error {
	value, exists := variables[validation.Variable]
	if !exists {
		if validation.Rule == "required" {
			return fmt.Errorf(validation.Message)
		}
		return nil
	}

	switch validation.Rule {
	case "min_length":
		if minLen, ok := validation.Value.(float64); ok {
			if str, ok := value.(string); ok && len(str) < int(minLen) {
				return fmt.Errorf(validation.Message)
			}
		}
	case "max_length":
		if maxLen, ok := validation.Value.(float64); ok {
			if str, ok := value.(string); ok && len(str) > int(maxLen) {
				return fmt.Errorf(validation.Message)
			}
		}
	case "regex":
		if pattern, ok := validation.Value.(string); ok {
			if str, ok := value.(string); ok {
				if regex, err := regexp.Compile(pattern); err == nil {
					if !regex.MatchString(str) {
						return fmt.Errorf(validation.Message)
					}
				}
			}
		}
	}

	return nil
}

func (tm *TemplateManager) matchesSearchCriteria(tmpl *Template, query string, filters map[string]interface{}) bool {
	// Text search
	if query != "" {
		searchText := strings.ToLower(tmpl.Name + " " + tmpl.Description + " " + tmpl.Content)
		if !strings.Contains(searchText, query) {
			return false
		}
	}

	// Filter by category
	if category, ok := filters["category"].(string); ok && category != "" {
		if tmpl.Category != category {
			return false
		}
	}

	// Filter by active status
	if isActive, ok := filters["is_active"].(bool); ok {
		if tmpl.IsActive != isActive {
			return false
		}
	}

	// Filter by tags
	if tags, ok := filters["tags"].([]string); ok && len(tags) > 0 {
		hasTag := false
		for _, filterTag := range tags {
			for _, tmplTag := range tmpl.Tags {
				if tmplTag == filterTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

func (tm *TemplateManager) createVersion(templateID, content, createdBy, changes string) {
	if tm.versions[templateID] == nil {
		tm.versions[templateID] = []TemplateVersion{}
	}

	version := TemplateVersion{
		Version:   fmt.Sprintf("1.%d.0", len(tm.versions[templateID])),
		Content:   content,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
		Changes:   changes,
	}

	tm.versions[templateID] = append(tm.versions[templateID], version)
	tm.saveVersions(templateID)
}

func (tm *TemplateManager) loadVersions() {
	pattern := filepath.Join(tm.versionsPath, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logrus.Errorf("[TEMPLATES] Failed to load versions: %v", err)
		return
	}

	for _, filePath := range matches {
		templateID := strings.TrimSuffix(filepath.Base(filePath), ".json")
		
		data, err := os.ReadFile(filePath)
		if err != nil {
			logrus.Errorf("[TEMPLATES] Failed to read version file %s: %v", filePath, err)
			continue
		}

		var versions []TemplateVersion
		if err := json.Unmarshal(data, &versions); err != nil {
			logrus.Errorf("[TEMPLATES] Failed to unmarshal version file %s: %v", filePath, err)
			continue
		}

		tm.versions[templateID] = versions
	}
}

func (tm *TemplateManager) saveVersions(templateID string) error {
	filePath := filepath.Join(tm.versionsPath, templateID+".json")
	
	data, err := json.MarshalIndent(tm.versions[templateID], "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0644)
}

func (tm *TemplateManager) createDefaultTemplates() {
	defaultTemplates := []*Template{
		{
			Name:        "Welcome Message",
			Description: "Advanced welcome message with personalization",
			Content:     `Hello {{.Variables.name | default "there"}}! 🎉

Welcome to our WhatsApp service. We're excited to have you with us!

{{if .Variables.company}}You're now connected to {{.Variables.company}}.{{end}}

How can we help you today? Here are some quick options:
• 📞 Speak to support
• 📋 View our services  
• 💬 Ask a question

Reply with the option number or just tell us what you need!

Best regards,
The {{.Variables.company | default "Support"}} Team`,
			Category:    "greeting",
			Language:    "en",
			Tags:        []string{"welcome", "greeting", "onboarding"},
			Variables: []Variable{
				{Name: "name", Type: "text", Required: false, DefaultValue: "there", Description: "Customer's name"},
				{Name: "company", Type: "text", Required: false, Description: "Company name"},
			},
		},
		{
			Name:        "Order Confirmation",
			Description: "Professional order confirmation with details",
			Content:     `🛍️ ORDER CONFIRMED

Hi {{.Variables.customer_name}},

Your order has been successfully confirmed!

📋 Order Details:
• Order ID: #{{.Variables.order_id}}
• Total: {{.Variables.total_amount | formatNumber "currency"}}
• Items: {{.Variables.item_count}} item(s)

📅 Delivery Information:
• Expected: {{.Variables.delivery_date | formatDate "January 2, 2006"}}
• Address: {{.Variables.delivery_address}}

📱 Track your order: {{.Variables.tracking_url}}

Questions? Reply to this message or call {{.Variables.support_phone}}.

Thank you for your business! 🙏`,
			Category:    "business",
			Language:    "en",
			Tags:        []string{"order", "confirmation", "ecommerce"},
			Variables: []Variable{
				{Name: "customer_name", Type: "text", Required: true, Description: "Customer's name"},
				{Name: "order_id", Type: "text", Required: true, Description: "Order ID"},
				{Name: "total_amount", Type: "number", Required: true, Description: "Total amount"},
				{Name: "item_count", Type: "number", Required: true, Description: "Number of items"},
				{Name: "delivery_date", Type: "date", Required: true, Description: "Delivery date"},
				{Name: "delivery_address", Type: "text", Required: true, Description: "Delivery address"},
				{Name: "tracking_url", Type: "url", Required: false, Description: "Tracking URL"},
				{Name: "support_phone", Type: "phone", Required: false, Description: "Support phone number"},
			},
		},
		{
			Name:        "Appointment Reminder",
			Description: "Smart appointment reminder with confirmation",
			Content:     `⏰ APPOINTMENT REMINDER

Hi {{.Variables.name}},

This is a friendly reminder about your upcoming appointment:

📅 Date: {{.Variables.date | formatDate "Monday, January 2, 2006"}}
🕐 Time: {{.Variables.time}}
📍 Location: {{.Variables.location}}
👨‍⚕️ With: {{.Variables.provider | default "our team"}}

{{if .Variables.preparation}}
📝 Please remember to:
{{.Variables.preparation}}
{{end}}

Please reply with:
✅ CONFIRM - to confirm your appointment
❌ CANCEL - to cancel
🔄 RESCHEDULE - to change date/time

Need directions? {{.Variables.maps_link}}

See you soon! 😊`,
			Category:    "reminder",
			Language:    "en",
			Tags:        []string{"appointment", "reminder", "healthcare", "booking"},
			Variables: []Variable{
				{Name: "name", Type: "text", Required: true, Description: "Patient/client name"},
				{Name: "date", Type: "date", Required: true, Description: "Appointment date"},
				{Name: "time", Type: "text", Required: true, Description: "Appointment time"},
				{Name: "location", Type: "text", Required: true, Description: "Appointment location"},
				{Name: "provider", Type: "text", Required: false, Description: "Service provider name"},
				{Name: "preparation", Type: "text", Required: false, Description: "Preparation instructions"},
				{Name: "maps_link", Type: "url", Required: false, Description: "Google Maps link"},
			},
		},
		{
			Name:        "Thank You Message",
			Description: "Personalized thank you with follow-up",
			Content:     `🙏 THANK YOU!

Dear {{.Variables.name}},

Thank you so much for choosing {{.Variables.company | default "our service"}}! 

{{if .Variables.service}}We're thrilled that you used our {{.Variables.service}} service.{{end}}

Your satisfaction means the world to us. We hope we exceeded your expectations!

⭐ How was your experience?
We'd love to hear your feedback. It helps us serve you better.

🎁 Special Offer:
As a token of our appreciation, enjoy {{.Variables.discount | default "10"}}% off your next purchase with code: THANKYOU{{.Variables.discount | default "10"}}

Stay connected:
📧 Email: {{.Variables.email}}
📱 Phone: {{.Variables.phone}}
🌐 Website: {{.Variables.website}}

We look forward to serving you again soon!

Warm regards,
The {{.Variables.company | default "Team"}} 💙`,
			Category:    "greeting",
			Language:    "en",
			Tags:        []string{"thank-you", "appreciation", "follow-up", "loyalty"},
			Variables: []Variable{
				{Name: "name", Type: "text", Required: true, Description: "Customer's name"},
				{Name: "company", Type: "text", Required: false, Description: "Company name"},
				{Name: "service", Type: "text", Required: false, Description: "Service used"},
				{Name: "discount", Type: "number", Required: false, DefaultValue: 10, Description: "Discount percentage"},
				{Name: "email", Type: "email", Required: false, Description: "Contact email"},
				{Name: "phone", Type: "phone", Required: false, Description: "Contact phone"},
				{Name: "website", Type: "url", Required: false, Description: "Website URL"},
			},
		},
	}

	for _, tmpl := range defaultTemplates {
		tmpl.Version = "1.0.0"
		tmpl.IsActive = true
		tmpl.Metadata = make(map[string]interface{})
		
		if _, err := tm.CreateAdvancedTemplate(tmpl); err != nil {
			logrus.Errorf("[TEMPLATES] Failed to create default template %s: %v", tmpl.Name, err)
		}
	}
}
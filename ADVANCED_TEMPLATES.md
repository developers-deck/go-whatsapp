# Advanced Template System Documentation

## Overview

The Advanced Template System is a powerful, feature-rich templating engine for WhatsApp messages that goes far beyond simple variable substitution. It provides enterprise-level capabilities including conditional logic, data transformations, validation, versioning, and much more.

## 🚀 Key Features

### 1. **Advanced Variable System**
- **Multiple Data Types**: text, number, date, email, phone, url, select, boolean
- **Default Values**: Automatic fallback values
- **Validation**: Built-in validation rules and custom regex patterns
- **Required/Optional**: Flexible variable requirements

### 2. **Go Template Engine Integration**
- **Full Go Template Syntax**: Complete template language support
- **Custom Functions**: 25+ built-in helper functions
- **Conditional Logic**: if/else statements, loops, comparisons
- **Data Manipulation**: String operations, math functions, formatting

### 3. **Data Transformations**
- **Text Transformations**: uppercase, lowercase, capitalize
- **Date Formatting**: Custom date formats
- **Number Formatting**: Currency, percentage, custom formats
- **Pipeline Operations**: Chain multiple transformations

### 4. **Template Versioning**
- **Version Control**: Track all template changes
- **Rollback Capability**: Restore to any previous version
- **Change History**: Detailed change logs
- **Backup System**: Automatic backups before changes

### 5. **Advanced Validation**
- **Input Validation**: Email, phone, URL format validation
- **Custom Rules**: Regex patterns, length constraints
- **Required Fields**: Enforce mandatory variables
- **Error Messages**: Custom validation error messages

### 6. **Template Management**
- **Categorization**: Organize templates by category
- **Tagging System**: Multiple tags per template
- **Search & Filter**: Advanced search capabilities
- **Bulk Operations**: Update multiple templates at once
- **Clone Templates**: Duplicate existing templates

### 7. **Scheduling & Conditions**
- **Time-based Scheduling**: Templates active only during specific periods
- **Conditional Rendering**: Show/hide content based on variables
- **Dynamic Content**: Adapt content based on context

## 📝 Template Syntax

### Basic Variable Substitution
```go
Hello {{.Variables.name}}!
```

### With Default Values
```go
Hello {{.Variables.name | default "there"}}!
```

### Conditional Content
```go
{{if .Variables.company}}
You're connected to {{.Variables.company}}.
{{else}}
Welcome to our service!
{{end}}
```

### Loops and Arrays
```go
Your items:
{{range .Variables.items}}
• {{.}}
{{end}}
```

### Advanced Formatting
```go
Total: {{.Variables.amount | formatNumber "currency"}}
Date: {{.Variables.date | formatDate "January 2, 2006"}}
```

### String Manipulation
```go
Name: {{.Variables.name | upper}}
Email: {{.Variables.email | lower}}
```

### Mathematical Operations
```go
Subtotal: {{.Variables.price | formatNumber "currency"}}
Tax: {{.Variables.price | multiply 0.1 | formatNumber "currency"}}
Total: {{.Variables.price | multiply 1.1 | formatNumber "currency"}}
```

## 🛠️ Built-in Functions

### String Functions
- `upper` - Convert to uppercase
- `lower` - Convert to lowercase  
- `title` - Title case
- `trim` - Remove whitespace
- `contains` - Check if string contains substring
- `replace` - Replace text
- `substr` - Extract substring
- `join` - Join array elements
- `split` - Split string into array

### Math Functions
- `add` - Addition
- `multiply` - Multiplication
- `divide` - Division
- `modulo` - Modulo operation

### Comparison Functions
- `eq` - Equal
- `ne` - Not equal
- `gt` - Greater than
- `lt` - Less than
- `gte` - Greater than or equal
- `lte` - Less than or equal

### Logic Functions
- `and` - Logical AND
- `or` - Logical OR
- `not` - Logical NOT

### Array Functions
- `len` - Get length
- `first` - Get first element
- `last` - Get last element
- `slice` - Extract slice

### Utility Functions
- `now` - Current timestamp
- `formatDate` - Format dates
- `formatNumber` - Format numbers
- `default` - Default value

## 🔧 API Endpoints

### Basic Template Operations
```bash
# Create simple template
POST /templates
{
  "name": "Welcome Message",
  "description": "Basic welcome",
  "content": "Hello {{name}}!",
  "category": "greeting"
}

# Create advanced template
POST /templates/advanced
{
  "name": "Order Confirmation",
  "description": "Advanced order confirmation",
  "content": "...",
  "variables": [...],
  "conditions": [...],
  "transformers": [...],
  "validations": [...]
}
```

### Template Management
```bash
# List templates
GET /templates?category=greeting&is_active=true

# Search templates
GET /templates/search?q=welcome&tags=greeting,onboarding

# Get template details
GET /templates/{id}

# Update template
PUT /templates/{id}

# Delete template
DELETE /templates/{id}
```

### Advanced Operations
```bash
# Clone template
POST /templates/{id}/clone
{
  "new_name": "Welcome Message Copy"
}

# Get template versions
GET /templates/{id}/versions

# Restore version
POST /templates/{id}/restore/{version}

# Bulk update
PUT /templates/bulk
{
  "template_id_1": {"is_active": false},
  "template_id_2": {"category": "updated"}
}
```

### Template Rendering
```bash
# Simple rendering
POST /templates/{id}/render
{
  "variables": {
    "name": "John Doe",
    "amount": 99.99
  }
}

# Advanced rendering
POST /templates/{id}/render-advanced
{
  "variables": {
    "name": "John Doe",
    "amount": 99.99,
    "items": ["Item 1", "Item 2"]
  },
  "language": "en",
  "timezone": "UTC",
  "metadata": {
    "source": "web"
  }
}
```

## 📊 Template Examples

### 1. Welcome Message with Personalization
```go
Hello {{.Variables.name | default "there"}}! 🎉

Welcome to our WhatsApp service. We're excited to have you with us!

{{if .Variables.company}}
You're now connected to {{.Variables.company}}.
{{end}}

How can we help you today? Here are some quick options:
• 📞 Speak to support
• 📋 View our services  
• 💬 Ask a question

Reply with the option number or just tell us what you need!

Best regards,
The {{.Variables.company | default "Support"}} Team
```

### 2. Order Confirmation with Calculations
```go
🛍️ ORDER CONFIRMED

Hi {{.Variables.customer_name}},

Your order has been successfully confirmed!

📋 Order Details:
• Order ID: #{{.Variables.order_id}}
• Items: {{.Variables.items | len}} item(s)
• Subtotal: {{.Variables.subtotal | formatNumber "currency"}}
• Tax: {{.Variables.subtotal | multiply 0.1 | formatNumber "currency"}}
• Total: {{.Variables.subtotal | multiply 1.1 | formatNumber "currency"}}

📅 Delivery: {{.Variables.delivery_date | formatDate "January 2, 2006"}}

{{if .Variables.tracking_url}}
📱 Track your order: {{.Variables.tracking_url}}
{{end}}

Thank you for your business! 🙏
```

### 3. Dynamic Content Based on Conditions
```go
{{if eq .Variables.user_type "premium"}}
🌟 PREMIUM CUSTOMER ALERT

Hi {{.Variables.name}},

As a premium customer, you get:
• Priority support
• Free shipping
• Exclusive discounts

{{else if eq .Variables.user_type "vip"}}
💎 VIP CUSTOMER SPECIAL

Dear {{.Variables.name}},

Your VIP benefits include:
• Dedicated account manager
• 24/7 support
• Custom solutions

{{else}}
👋 Hello {{.Variables.name}},

Thank you for being a valued customer!
{{end}}

{{if gt .Variables.loyalty_points 1000}}
🎁 Bonus: You have {{.Variables.loyalty_points}} loyalty points!
{{end}}
```

## 🔒 Variable Types & Validation

### Variable Definition
```json
{
  "name": "email",
  "type": "email",
  "required": true,
  "description": "Customer email address",
  "validation": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
}
```

### Supported Types
- **text**: Plain text (default)
- **number**: Numeric values
- **date**: Date values
- **email**: Email addresses (auto-validated)
- **phone**: Phone numbers (auto-validated)
- **url**: URLs (auto-validated)
- **select**: Dropdown options
- **boolean**: True/false values

### Validation Rules
```json
{
  "validations": [
    {
      "variable": "name",
      "rule": "min_length",
      "value": 2,
      "message": "Name must be at least 2 characters"
    },
    {
      "variable": "age",
      "rule": "regex",
      "value": "^[0-9]+$",
      "message": "Age must be a number"
    }
  ]
}
```

## 🔄 Data Transformers

### Transformer Configuration
```json
{
  "transformers": [
    {
      "variable": "name",
      "type": "capitalize"
    },
    {
      "variable": "date",
      "type": "format_date",
      "options": {
        "format": "January 2, 2006"
      }
    },
    {
      "variable": "price",
      "type": "format_number",
      "options": {
        "format": "currency"
      }
    }
  ]
}
```

## 📅 Scheduling

### Schedule Configuration
```json
{
  "scheduling": {
    "enabled": true,
    "start_date": "2024-01-01T00:00:00Z",
    "end_date": "2024-12-31T23:59:59Z",
    "timezone": "UTC",
    "recurring": false
  }
}
```

## 📈 Usage Statistics

### Template Stats
```bash
GET /templates/stats
```

Response:
```json
{
  "total_templates": 25,
  "categories": {
    "greeting": 8,
    "business": 12,
    "reminder": 5
  },
  "most_used": {
    "id": "welcome_20240101120000",
    "name": "Welcome Message",
    "usage_count": 1250
  },
  "total_usage": 5430
}
```

## 🎯 Best Practices

### 1. **Template Organization**
- Use clear, descriptive names
- Categorize templates logically
- Add relevant tags for easy searching
- Include detailed descriptions

### 2. **Variable Design**
- Use descriptive variable names
- Set appropriate default values
- Add validation rules for data quality
- Include helpful descriptions

### 3. **Content Structure**
- Keep templates focused and concise
- Use consistent formatting
- Include clear call-to-actions
- Test with various data scenarios

### 4. **Version Management**
- Create versions for significant changes
- Document changes in version notes
- Test thoroughly before deploying
- Keep backup versions

### 5. **Performance Optimization**
- Avoid overly complex logic
- Use efficient template functions
- Cache frequently used templates
- Monitor rendering performance

## 🔧 Integration Examples

### JavaScript/Node.js
```javascript
// Render advanced template
const response = await fetch('/templates/template_id/render-advanced', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    variables: {
      name: 'John Doe',
      amount: 99.99,
      items: ['Product A', 'Product B']
    },
    language: 'en',
    timezone: 'America/New_York'
  })
});

const result = await response.json();
console.log(result.results.rendered_content);
```

### Python
```python
import requests

# Create advanced template
template_data = {
    "name": "Custom Template",
    "content": "Hello {{.Variables.name}}!",
    "variables": [
        {
            "name": "name",
            "type": "text",
            "required": True,
            "description": "Customer name"
        }
    ]
}

response = requests.post('/templates/advanced', json=template_data)
template = response.json()
```

### cURL
```bash
# Search templates
curl -X GET "http://localhost:3000/templates/search?q=welcome&category=greeting"

# Render template
curl -X POST "http://localhost:3000/templates/template_id/render-advanced" \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "name": "John Doe",
      "company": "Acme Corp"
    }
  }'
```

## 🎉 Conclusion

The Advanced Template System provides enterprise-level templating capabilities that can handle complex messaging scenarios while maintaining ease of use. With features like versioning, validation, conditional logic, and advanced formatting, it's suitable for everything from simple notifications to complex, personalized communication workflows.

The system is designed to be both powerful for advanced users and accessible for basic use cases, making it a versatile solution for any WhatsApp messaging needs.
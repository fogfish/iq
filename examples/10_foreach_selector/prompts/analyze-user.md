---
format: json
schema:
  input:
    type: object
    properties:
      id: {type: integer}
      name: {type: string}
      email: {type: string}
      active: {type: boolean}
  reply:
    type: object
    properties:
      analysis: {type: string}
      risk_level: {type: string}
      recommendations: 
        type: array
        items: {type: string}
---
Analyze this user profile:

**User Details:**
- ID: {{.current.id}}
- Name: {{.current.name}}
- Email: {{.current.email}}
- Status: {{if .current.active}}Active{{else}}Inactive{{end}}

Provide:
1. A brief analysis of the user profile
2. Risk level assessment (low/medium/high)
3. 2-3 recommendations for user management
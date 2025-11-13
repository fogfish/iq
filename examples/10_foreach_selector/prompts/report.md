---
format: text
---

Create a brief summary based on the following reports.

**Users report**
{{range .steps.users}}
Risk: {{.risk_level}}
{{.analysis}}
{{range .recommendations}}
- {{.}}
{{end}} 
{{end}}

**Items report**
{{range .steps.items}}
Priority: {{.urgency}}
{{.analysis}}
{{range .actions}}
- {{.}}
{{end}} 
{{end}}

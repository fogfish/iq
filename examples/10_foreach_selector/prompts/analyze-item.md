---
format: json
schema:
  input:
    type: object
    properties:
      category: {type: string}
      priority: {type: integer}
  reply:
    type: object
    properties:
      analysis: {type: string}
      urgency: {type: string}
      actions:
        type: array
        items: {type: string}
---
Analyze this item:

**Item Details:**
- Category: {{.current.category}}
- Priority: {{.current.priority}}

Provide:
1. Analysis of the item's significance
2. Urgency assessment based on priority (1=lowest, 5=highest)
3. 2-3 recommended actions
---
format: json
schema:
  reply:
    type: object
    properties:
      facts: 
        type: array
        items: {type: string}
---
Extract keywords from the following text as a JSON array: {{.input}}

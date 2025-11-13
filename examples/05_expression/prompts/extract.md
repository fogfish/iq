---
format: json
schema:
  input:
    type: string
  reply:
    type: object
    required: [specs, category]
    properties:
      specs: {type: string}
      category: {type: string}
---
Extract the technical specifications and categorize the device from this text: {{.input}}

Return JSON with:
- specs: the technical details
- category: device category (server, laptop, phone, etc.)

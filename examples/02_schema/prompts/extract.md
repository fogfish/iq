---
format: json
schema:
  input:
    type: string
  reply:
    type: object
    required: [spec]
    properties:
      spec: {type: string}
---
Extract the technical specifications from the following text: {{.input}}.

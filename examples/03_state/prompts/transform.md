---
format: json
schema:
  reply:
    type: object
    required: [cpu, memory, storage]
    properties:
      cpu: {type: string}
      memory: {type: string}
      storage: {type: string}
---
Parse the specifications into structured JSON with 'cpu', 'memory', and 'storage' fields.

Specifications: {{.steps.text}}

---
format: json
schema:
  input:
    type: object
    required: [spec]
    properties:
      spec: {type: string}
  reply:
    type: object
    required: [cpu, memory, storage]
    properties:
      cpu: {type: string}
      memory: {type: string}
      storage: {type: string}
---
Transform the following specifications into a JSON object with 'cpu', 'memory', and 'storage' as keys: {{.input.spec}}

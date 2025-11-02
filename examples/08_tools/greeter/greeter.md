---
format: json
schema:
  input:
    type: object
    properties:
      name:
        type: string
    required: [name]
  reply:
    type: object
    properties:
      greeting:
        type: string
    required: [greeting]
---
Say hi to {{.input.name}}!

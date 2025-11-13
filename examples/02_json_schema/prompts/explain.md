---
format: json
schema:
  input:
    type: object
    required: [cpu]
    properties:
      cpu: {type: string}
  reply:
    type: object
    required: [speed]
    properties:
      spec: {type: number}

---
What is the clock speed of the cpu {{.current.cpu}}?

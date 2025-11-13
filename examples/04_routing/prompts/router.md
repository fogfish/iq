---
format: text
---
Analyze the user's request and determine which specialist handler should process it.

1. If the request is related to booking flights or hotels, output 'booker'.
2. For all other general information questions, output 'info'.
3. If the request is unclear or doesn't fit either category, output 'unknown'.

Strictly adhere to requirement to output one word: 'booker', 'info' or 'unkown'.

Input request:
{{.input}}

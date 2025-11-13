---
format: json
schema:
  reply:
    type: object
    properties:
      title:
        type: string
        description: The page title
      summary:
        type: string
        description: Brief summary of the page content
      word_count:
        type: integer
        description: Approximate word count
---
You have fetched a web page. Here's what was extracted:

Please provide:
1. A clean version of the page title
2. A brief 2-3 sentence summary of the page content
3. An approximate word count

Return your response in JSON format according to the schema.

Web page:
{{.input}}


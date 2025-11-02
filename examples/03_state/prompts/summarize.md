---
format: text
---
Create a brief summary report using the following information. Generate a 2-3 sentence summary that mentions the device category and key specs. The output must follow the template: 

# Summary

## ORIGINAL DOCUMENT:
{{.document}}

## DEVICE CATEGORY:
{{.steps.text.category}}

## PARSED SPECIFICATIONS:
- CPU: {{.steps.specs.cpu}}
- Memory: {{.steps.specs.memory}}
- Storage: {{.steps.specs.storage}}

## SUMMARY
add you summary here


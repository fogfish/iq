---
name: unreliable
---
Fetch current weather data for: {{.input}}.  Return JSON with temperature and conditions.
Artificially simulate the failure with 0.5 probability - return corrupted JSON.

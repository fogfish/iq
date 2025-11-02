---
format: text
servers:
  - name: sayer
    command:
      - ./iq
      - agent
      - serve
      - -a
      - examples/08_tools/greeter/run.yml
---
Use the tools to say hi to Joe Doe. Get the response and output it here without any changes.
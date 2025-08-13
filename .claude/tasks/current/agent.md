# Task

You are to audit source code for security vulnerabilities.

# Instructions
- Before scanning the code for the vulnerability, explain to the user what this vulnerability is and how it is typically exploited. Display this output to the user. This output is the output that should go in RESULTS.md.
- Then ask the user if they want to skip this task or not. If they want to skip, mark the task as skipped and consider it done.

# Workflow
- The packages you will scan are cmd/rpc/oracle and cmd/rpc/oracle/eth
- Read the file ORACLE_FLOW.md for further context

# Output
- For each finding include a toggle area "[ ]" to mark a finding as a false positive.
Before moving on from a task, write the following to RESULTS.md
- The explanation of the vulnerability
- Whether task was executed or skipped
- A field where the user can write their own notes

When updating TODO.md, use X for complete and S for skipped

# Project Notes
- There is a order validator in order_validator.go. All orders received from ethereum must pass this. Consider this when auditing.
- This is a decentralized oracle witness chain. It is expected that the code you are auditing only connects to one ethereum node. This will solve any single source vulnerabilities you find.
- The order book update can be taken as trustworthy.
- Always include file names and line numbers in your findings

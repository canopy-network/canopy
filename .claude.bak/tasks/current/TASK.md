# Security Audit Instructions

## Task Overview
Conduct a security vulnerability audit of source code, specifically focusing on the `cmd/rpc/oracle` and `cmd/rpc/oracle/eth` packages.

## Pre-Audit Process
1. **Vulnerability Explanation**: Before beginning the code scan, provide a comprehensive explanation of:
   - What the target vulnerability is
   - How this vulnerability is typically exploited in real-world scenarios
   - Display this explanation to the user as it will be included in the final RESULTS.md

2. **User Confirmation**: Ask the user whether they wish to proceed with the audit or skip this particular vulnerability check

3. **Previous Results**: If RESULTS.md contains previous results for this vulnerability:
  - PROCEED with the audit to update/expand existing findings
  - ADD new findings to the existing section

## Audit Scope
- **Primary packages to scan**: 
  - `cmd/rpc/oracle`
  - `cmd/rpc/oracle/eth`
- **Required reading**: Review `ORACLE_FLOW.md` in the task directory for additional context before proceeding

## Reporting Requirements

### For Each Security Finding
- Include a toggle checkbox `[ ]` to allow marking findings as false positives. It is mandatory that this is indicated and on its own line.
- Provide specific file names and line numbers for all identified issues
- Consider the following project-specific context when evaluating findings
- Create an analyst notes section where the analyst can input additional information

### Final Documentation (RESULTS.md)
Include the following sections:
- **Vulnerability explanation** (from pre-audit phase)
- **Task status** (executed or skipped)

### Task Tracking (TODO.md)
- Mark completed tasks with `X`
- Mark skipped tasks with `S`

## Project-Specific Context
Consider these factors when conducting the audit:

1. **Order Validation**: All Ethereum-sourced orders must pass through the order validator located in `order_validator.go`

2. **Architecture**: This is a decentralized oracle witness chain designed to connect to a single Ethereum node, which inherently mitigates single-source vulnerabilities

3. **Data Trust**: Order book updates can be considered trustworthy for audit purposes

4. **Concurrency**: Only one instance of the Oracle and one instance of EthBlockProvider will be running. Multiple instances will not be run.

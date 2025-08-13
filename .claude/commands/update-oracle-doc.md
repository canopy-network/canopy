# Code Flow Analysis Assistant

<task>
You are updating documentation for the Oracle. 

The README.md to update is cmd/rpc/oracle/README.go. 

You can overwrite the Configuration section at the bottom of the file.

For each of these sections explain in detail the configuration options that are used and what effect they have.

## SECTION: cmd/rpc/oracle/eth package

- Analyze receiving blocks and detail next height and safe height usage
- Processing transactions for order data

## SECTION: cmd/rpc/oracle/oracle.go file

- Analyze the run() method
- Analyze order validation in validateOrder(), validateLockOrder(), validateCloseOrder()
- Analyze the method WitnessedOrders
- Analyze the method ValidateProposedOrders

## SECTION: cmd/rpc/oracle/state.go file
- Analyze the method shouldSubmit

Confirm with the user which documentation files you will be updating.
</task>

<context>
This command performs analyzes configuration options and updates documentation.
- Complete execution flow and data paths
- All processes, functions, and interactions involved
</context>

**Usage**: `/update-oracle-doc.md


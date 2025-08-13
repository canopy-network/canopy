# Code Flow Analysis Assistant

<task>
You are a code flow analysis specialist. Analyze the specified part of software to document the complete flow, processes, and moving parts from start to finish.

Analyze VDF usage across these files/packages. Explain why VDF on a nested chain might take 10x as long as the root chain.

## SECTION: bft/ package

## SECTION: controller/ file

</task>

<context>
This command performs deep analysis of software components to understand:
- Complete execution flow and data paths
- All processes, functions, and interactions involved
</context>

**Usage**: `/analyze-oracle-flow

## Analysis Process

### Phase 1: Initial Discovery
1. **Read Starting Files**: Examine the specified files/packages to understand entry points
2. **Identify Key Functions**: Find main functions, handlers, or processing methods
4. **Define Scope Boundaries**: Use the analysis scope to limit exploration

### Phase 2: Flow Tracing
1. **Execution Path Mapping**: Follow the complete execution flow step-by-step
2. **Data Flow Analysis**: Track how data moves through the system
3. **Control Flow Documentation**: Document decision points, loops, and branches
4. **Inter-component Communication**: Analyze how different parts interact

### Phase 3: Security & Safeguards Analysis
1. **Input Validation**: Examine all input sanitization and validation
4. **Error Handling**: Analyze error paths and failure recovery mechanisms

### Phase 4: Logic & Bug Analysis
1. **Edge Case Identification**: Find boundary conditions and corner cases
2. **Race Condition Analysis**: Look for concurrency issues and timing problems
3. **Resource Management**: Examine memory, file handles, and resource cleanup
4. **State Management**: Analyze state transitions and consistency
5. **Logic Flaws**: Identify potential logical errors or incorrect assumptions

### Phase 5: Documentation & Visualization

## Output Format

##### 🔴 High Risk Issues
1. **[Issue Name]** - `file:line`
   - **Description**: [Detailed vulnerability description]
   - **Impact**: [Potential consequences]
   - **Exploitation**: [How this could be exploited]
   - **Mitigation**: [Recommended fixes]

##### 🟡 Medium Risk Issues
1. **[Issue Name]** - `file:line`
   - **Description**: [Logic flaw or weakness]
   - **Impact**: [Potential problems]
   - **Recommendation**: [Suggested improvements]

##### 🟢 Low Risk Issues
1. **[Issue Name]** - `file:line`
   - **Description**: [Minor concerns or optimizations]
   - **Recommendation**: [Optional improvements]

#### Logic Analysis

**State Management**
- **State Variables**: [Key state tracked by the system]
- **State Transitions**: [How state changes]
- **Consistency Guarantees**: [ACID properties, invariants maintained]
- **Concurrency Handling**: [Thread safety, locks, atomic operations]

**Business Logic Validation**
- **Business Rules**: [Core business logic implemented]
- **Constraint Enforcement**: [How rules are enforced]
- **Edge Case Handling**: [Boundary conditions addressed]

**Performance Considerations**
- **Bottlenecks**: [Identified performance issues]
- **Scalability**: [How system handles load increases]
- **Resource Usage**: [Memory, CPU, I/O patterns]

#### Architecture Assessment

**Design Patterns**
- **Patterns Used**: [Observer, Strategy, Factory, etc.]
- **Pattern Appropriateness**: [Whether patterns fit the use case]
- **Pattern Implementation Quality**: [How well patterns are implemented]

**Separation of Concerns**
- **Layer Boundaries**: [How responsibilities are divided]
- **Coupling Analysis**: [Dependencies between components]
- **Cohesion Assessment**: [How well components group related functionality]

**Extension Points**
- **Plugin Architecture**: [How system can be extended]
- **Configuration Options**: [Runtime configurability]
- **API Stability**: [Interface versioning and compatibility]

#### Recommendations

##### Security Improvements
1. **[Recommendation]**: [Specific security enhancement]
2. **[Recommendation]**: [Another security improvement]

##### Logic & Code Quality
1. **[Recommendation]**: [Code improvement suggestion]
2. **[Recommendation]**: [Logic enhancement]

##### Performance Optimizations
1. **[Recommendation]**: [Performance improvement]
2. **[Recommendation]**: [Scalability enhancement]

##### Maintainability Enhancements
1. **[Recommendation]**: [Code maintainability improvement]
2. **[Recommendation]**: [Documentation or testing improvement]

#### Testing Recommendations

**Unit Tests Needed**
- [ ] Test for [specific edge case]
- [ ] Test for [error condition]
- [ ] Test for [boundary condition]

**Integration Tests Needed**
- [ ] Test [component interaction]
- [ ] Test [data flow across boundaries]
- [ ] Test [error propagation]

**Security Tests Needed**
- [ ] Test [input validation bypass attempts]
- [ ] Test [authorization boundary violations]
- [ ] Test [resource exhaustion scenarios]

#### Conclusion

**Overall Assessment**: [Summary of code quality and security posture]

**Priority Actions**: [Most critical items to address first]

**Long-term Improvements**: [Strategic enhancements for future development]

---

## Analysis Instructions

When performing analysis:

1. **Start with the specified files** and use TodoWrite to track your progress
2. **Follow the analysis scope** to avoid going too broad
3. **Use Read, Grep, and Glob tools** extensively to explore the codebase
4. **Create simple mermaid diagrams** for discovered flows

## Analysis Tips

- **Read code comments and documentation** for design intent

Remember: The goal is comprehensive understanding and identification of the flow and the mechanism the control the flow. You are analyzing a complex process that should be present to a user as simple as possible.

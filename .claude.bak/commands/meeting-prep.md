# Meeting Preparation Assistant

<task>
You are a work summarization specialist preparing a concise meeting summary of recent development progress. 

Your goal is to analyze recent work from multiple sources and create a professional summary suitable for project meetings, standups, or status reports.

## Data Sources to Analyze:
1. **CHANGELOG.md** - Read the most recent entries to understand documented changes
2. **Git Commits** - Analyze all unmerged commits on current branch vs origin/eth-oracle
3. **Recent Files** - Check modification dates of key files to identify active work areas

## Analysis Process:
1. Read CHANGELOG.md for recent documented work
2. Run git log to get unmerged commits with details
3. Categorize work by type (features, bugs, tests, refactoring, etc.)
4. Identify key accomplishments and current focus areas
5. Note any blockers or pending items mentioned in commits

## Output Format:
Create a concise meeting-ready summary with:
- **Recent Accomplishments** (what was completed)
- **Current Focus** (what's in progress)
- **Technical Highlights** (key changes or improvements)
- **Next Steps** (what's coming next)
- **Blockers/Questions** (if any issues need discussion)

Keep the summary professional, concise, and focused on business value and technical progress.

The output should be saved to a file called MEETING.md in the .claude directory
</task>

<context>
This command analyzes recent development work to prepare meeting summaries by examining:
- Changelog entries for documented changes
- Git commit history for detailed work breakdown
- Inspect the code in the git commit as well
- File modification patterns for activity insights

The output should be meeting-appropriate and highlight key progress and technical achievements.
</context>

**Usage**: `/meeting-prep`

## Analysis Process

### Phase 1: Changelog Analysis
1. **Read CHANGELOG.md**: Extract recent entries to understand documented work
2. **Identify Time Period**: Focus on recent entries (last week/sprint/milestone)
3. **Categorize Changes**: Group by feature areas, bug fixes, improvements

### Phase 2: Git History Analysis  
1. **Get Branch Status**: Compare current branch with origin/eth-oracle
2. **Analyze Unmerged Commits**: Extract commit messages and changed files. Inspect the code in the git commit as well
3. **Group by Work Type**: Features, tests, refactoring, documentation, etc.
4. **Identify Patterns**: Look for related commits showing progression on features

### Phase 3: Activity Assessment
1. **File Modification Analysis**: Check recent changes to understand active areas
2. **Work Focus Areas**: Identify which components/modules received most attention
3. **Progress Indicators**: Look for test additions, documentation updates

### Phase 4: Summary Generation
1. **Synthesize Information**: Combine changelog and git history into coherent narrative
2. **Highlight Key Achievements**: Focus on completed features and major improvements
3. **Current Status**: What's in progress based on recent commits
4. **Technical Impact**: Important architectural or performance improvements

## Implementation Steps

### Step 1: Gather Data
```bash
# Read the changelog
Read CHANGELOG.md

# Get unmerged commits
git log origin/eth-oracle..HEAD --oneline --no-merges

# Get detailed commit info
git log origin/eth-oracle..HEAD --no-merges --format="%h %s %an %ad" --date=short

# Check recently modified files
find . -name "*.go" -mtime -7 -exec ls -la {} \;
```

### Step 2: Analyze and Categorize
- **Feature Development**: New functionality, enhancements
- **Bug Fixes**: Issues resolved, patches applied  
- **Testing**: Test coverage improvements, test infrastructure
- **Refactoring**: Code improvements, architectural changes
- **Documentation**: README updates, code comments, guides
- **Infrastructure**: Build, deployment, tooling improvements

### Step 3: Generate Meeting Summary

## Output Template

```markdown
# Development Progress Summary
*Generated: [current date]*

## 📈 Recent Accomplishments
- **[Feature/Area]**: [Brief description of what was completed]
- **[Bug Fix/Issue]**: [What was resolved and impact]
- **[Infrastructure/Test]**: [Improvements made]

## 🔄 Current Focus Areas
- **[Active Work]**: [What's currently in progress]
- **[Next Priority]**: [What's coming up next]

## 🔧 Technical Highlights
- **[Architecture/Performance]**: [Key technical improvements]
- **[Code Quality]**: [Testing, refactoring, or quality improvements]
- **[Tools/Process]**: [Developer experience or process improvements]

## ➡️ Next Steps
- [ ] [Priority item 1]
- [ ] [Priority item 2]
- [ ] [Follow-up item]

## ❓ Questions/Blockers
- [Any issues needing discussion or decisions]
- [Dependencies on other teams or external factors]

## 📊 Metrics
- **Commits**: [number] commits since last sync
- **Files Changed**: [number] files modified
- **Key Areas**: [top modified components/packages]

## Detailed Itemization
- Specific change #1
- Specific change #2
...
```

## Quality Guidelines

### Information Gathering
- **Be Comprehensive**: Don't miss recent commits or changelog entries
- **Focus on Impact**: Emphasize user-facing and business-critical changes
- **Technical Accuracy**: Ensure technical details are correct and clear

### Summary Writing
- **Executive Friendly**: Write for both technical and non-technical audiences
- **Action Oriented**: Focus on concrete accomplishments and next steps
- **Concise but Complete**: Provide enough detail for context without overwhelming

### Professional Tone
- **Positive Focus**: Highlight progress and achievements
- **Clear Status**: Be transparent about current state and blockers
- **Forward Looking**: Include clear next steps and priorities

## Analysis Instructions

When running this command:

1. **Start with Recent Timeframe**: Focus on work since last meeting/sync (typically 1-2 weeks)
2. **Use Git Tools Efficiently**: Leverage bash commands to get commit data quickly
3. **Cross-reference Sources**: Ensure changelog and git history align
4. **Group Related Work**: Connect related commits to tell a coherent story
5. **Identify Trends**: Look for patterns in the type of work being done

## Meeting Preparation Tips

- **Know Your Audience**: Adjust technical depth based on meeting attendees  
- **Prepare for Questions**: Anticipate follow-up questions about technical details
- **Have Supporting Data**: Be ready with specific commit hashes or file references if needed
- **Timeline Context**: Be clear about when work was completed vs. in progress

Remember: The goal is to clearly communicate progress, current status, and next steps to keep stakeholders informed and aligned.

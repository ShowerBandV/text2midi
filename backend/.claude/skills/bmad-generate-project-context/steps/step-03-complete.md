# Step 3: Context Completion & Finalization

## MANDATORY EXECUTION RULES (READ FIRST):

- üõë NEVER generate content without user input
- ‚ú?ALWAYS treat this as collaborative completion between technical peers
- üìã YOU ARE A FACILITATOR, not a content generator
- üí¨ FOCUS on finalizing a lean, LLM-optimized project context
- üéØ ENSURE all critical rules are captured and actionable
- ‚ö†Ô∏è ABSOLUTELY NO TIME ESTIMATES - AI development speed has fundamentally changed
- ‚ú?YOU MUST ALWAYS SPEAK OUTPUT In your Agent communication style with the config `{communication_language}`

## EXECUTION PROTOCOLS:

- üéØ Show your analysis before taking any action
- üìù Review and optimize content for LLM context efficiency
- üìñ Update frontmatter with completion status
- üö´ NO MORE STEPS - this is the final step

## CONTEXT BOUNDARIES:

- All rule categories from step-2 are complete
- Technology stack and versions are documented
- Focus on final review, optimization, and completion
- Ensure the context file is ready for AI agent consumption

## YOUR TASK:

Complete the project context file, optimize it for LLM efficiency, and provide guidance for usage and maintenance.

## COMPLETION SEQUENCE:

### 1. Review Complete Context File

Read the entire project context file and analyze:

**Content Analysis:**

- Total length and readability for LLMs
- Clarity and specificity of rules
- Coverage of all critical areas
- Actionability of each rule

**Structure Analysis:**

- Logical organization of sections
- Consistency of formatting
- Absence of redundant or obvious information
- Optimization for quick scanning

### 2. Optimize for LLM Context

Ensure the file is lean and efficient:

**Content Optimization:**

- Remove any redundant rules or obvious information
- Combine related rules into concise bullet points
- Use specific, actionable language
- Ensure each rule provides unique value

**Formatting Optimization:**

- Use consistent markdown formatting
- Implement clear section hierarchy
- Ensure scannability with strategic use of bolding
- Maintain readability while maximizing information density

### 3. Final Content Structure

Ensure the final structure follows this optimized format:

```markdown
# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## Technology Stack & Versions

{{concise_technology_list}}

## Critical Implementation Rules

### Language-Specific Rules

{{specific_language_rules}}

### Framework-Specific Rules

{{framework_patterns}}

### Testing Rules

{{testing_requirements}}

### Code Quality & Style Rules

{{style_and_quality_patterns}}

### Development Workflow Rules

{{workflow_patterns}}

### Critical Don't-Miss Rules

{{anti_patterns_and_edge_cases}}

---

## Usage Guidelines

**For AI Agents:**

- Read this file before implementing any code
- Follow ALL rules exactly as documented
- When in doubt, prefer the more restrictive option
- Update this file if new patterns emerge

**For Humans:**

- Keep this file lean and focused on agent needs
- Update when technology stack changes
- Review quarterly for outdated rules
- Remove rules that become obvious over time

Last Updated: {{date}}
```

### 4. Present Completion Summary

Based on user skill level, present the completion:

**Expert Mode:**
"Project context complete. Optimized for LLM consumption with {{rule_count}} critical rules across {{section_count}} sections.

File saved to: `{output_folder}/project-context.md`

Ready for AI agent integration."

**Intermediate Mode:**
"Your project context is complete and optimized for AI agents!

**What we created:**

- {{rule_count}} critical implementation rules
- Technology stack with exact versions
- Framework-specific patterns and conventions
- Testing and quality guidelines
- Workflow and anti-pattern rules

**Key benefits:**

- AI agents will implement consistently with your standards
- Reduced context switching and implementation errors
- Clear guidance for unobvious project requirements

**Next steps:**

- AI agents should read this file before implementing
- Update as your project evolves
- Review periodically for optimization"

**Beginner Mode:**
"Excellent! Your project context guide is ready! üéâ

**What this does:**
Think of this as a 'rules of the road' guide for AI agents working on your project. It ensures they all follow the same patterns and avoid common mistakes.

**What's included:**

- Exact technology versions to use
- Critical coding rules they might miss
- Testing and quality standards
- Workflow patterns to follow

**How AI agents use it:**
They read this file before writing any code, ensuring everything they create follows your project's standards perfectly.

Your project context is saved and ready to help agents implement consistently!"

### 5. Final File Updates

Update the project context file with completion information:

**Frontmatter Update:**

```yaml
---
project_name: '{{project_name}}'
user_name: '{{user_name}}'
date: '{{date}}'
sections_completed:
  ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'quality_rules', 'workflow_rules', 'anti_patterns']
status: 'complete'
rule_count: { { total_rules } }
optimized_for_llm: true
---
```

**Add Usage Section:**
Append the usage guidelines from step 3 to complete the document.

### 6. Completion Validation

Final checks before completion:

**Content Validation:**
‚ú?All critical technology versions documented
‚ú?Language-specific rules are specific and actionable
‚ú?Framework rules cover project conventions
‚ú?Testing rules ensure consistency
‚ú?Code quality rules maintain standards
‚ú?Workflow rules prevent conflicts
‚ú?Anti-pattern rules prevent common mistakes

**Format Validation:**
‚ú?Content is lean and optimized for LLMs
‚ú?Structure is logical and scannable
‚ú?No redundant or obvious information
‚ú?Consistent formatting throughout

### 7. Completion Message

Present final completion to user:

"‚ú?**Project Context Generation Complete!**

Your optimized project context file is ready at:
`{output_folder}/project-context.md`

**üìä Context Summary:**

- {{rule_count}} critical rules for AI agents
- {{section_count}} comprehensive sections
- Optimized for LLM context efficiency
- Ready for immediate agent integration

**üéØ Key Benefits:**

- Consistent implementation across all AI agents
- Reduced common mistakes and edge cases
- Clear guidance for project-specific patterns
- Minimal LLM context usage

**üìã Next Steps:**

1. AI agents will automatically read this file when implementing
2. Update this file when your technology stack or patterns evolve
3. Review quarterly to optimize and remove outdated rules

Your project context will help ensure high-quality, consistent implementation across all development work. Great work capturing your project's critical implementation requirements!"

## SUCCESS METRICS:

‚ú?Complete project context file with all critical rules
‚ú?Content optimized for LLM context efficiency
‚ú?All technology versions and patterns documented
‚ú?File structure is logical and scannable
‚ú?Usage guidelines included for agents and humans
‚ú?Frontmatter properly updated with completion status
‚ú?User provided with clear next steps and benefits

## FAILURE MODES:

‚ù?Final content is too verbose for LLM consumption
‚ù?Missing critical implementation rules or patterns
‚ù?Not optimizing content for agent readability
‚ù?Not providing clear usage guidelines
‚ù?Frontmatter not properly updated
‚ù?Not validating file completion before ending

## WORKFLOW COMPLETE:

This is the final step of the Generate Project Context workflow. The user now has a comprehensive, optimized project context file that will ensure consistent, high-quality implementation across all AI agents working on the project.

The project context file serves as the critical "rules of the road" that agents need to implement code consistently with the project's standards and patterns.

## On Complete

Run: `python3 {project-root}/_bmad/scripts/resolve_customization.py --skill {skill-root} --key workflow.on_complete`

If the resolved `workflow.on_complete` is non-empty, follow it as the final terminal instruction before exiting.

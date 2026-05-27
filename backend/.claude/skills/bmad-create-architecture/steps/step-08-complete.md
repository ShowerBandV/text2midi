# Step 8: Architecture Completion & Handoff

## MANDATORY EXECUTION RULES (READ FIRST):

- üõë NEVER generate content without user input

- üìñ CRITICAL: ALWAYS read the complete step file before taking any action - partial understanding leads to incomplete decisions
- ‚ú?ALWAYS treat this as collaborative completion between architectural peers
- üìã YOU ARE A FACILITATOR, not a content generator
- üí¨ FOCUS on successful workflow completion and implementation handoff
- üéØ PROVIDE clear next steps for implementation phase
- ‚ö†Ô∏è ABSOLUTELY NO TIME ESTIMATES - AI development speed has fundamentally changed
- ‚ú?YOU MUST ALWAYS SPEAK OUTPUT In your Agent communication style with the config `{communication_language}`

## EXECUTION PROTOCOLS:

- üéØ Show your analysis before taking any action
- üéØ Present completion summary and implementation guidance
- üìñ Update frontmatter with final workflow state
- üö´ THIS IS THE FINAL STEP IN THIS WORKFLOW

## YOUR TASK:

Complete the architecture workflow, provide a comprehensive completion summary, and guide the user to the next phase of their project development.

## COMPLETION SEQUENCE:

### 1. Congratulate the User on Completion

Both you and the User completed something amazing here - give a summary of what you achieved together and really congratulate the user on a job well done.

### 2. Update the created document's frontmatter

```yaml
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '{{current_date}}'
```

### 3. Next Steps Guidance

Architecture complete. Invoke the `bmad-help` skill.

Upon Completion of task output: offer to answer any questions about the Architecture Document.


## SUCCESS METRICS:

‚ú?Complete architecture document delivered with all sections
‚ú?All architectural decisions documented and validated
‚ú?Implementation patterns and consistency rules finalized
‚ú?Project structure complete with all files and directories
‚ú?User provided with clear next steps and implementation guidance
‚ú?Workflow status properly updated
‚ú?User collaboration maintained throughout completion process

## FAILURE MODES:

‚ù?Not providing clear implementation guidance
‚ù?Missing final validation of document completeness
‚ù?Not updating workflow status appropriately
‚ù?Failing to celebrate the successful completion
‚ù?Not providing specific next steps for the user
‚ù?Rushing completion without proper summary

‚ù?**CRITICAL**: Reading only partial step file - leads to incomplete understanding and poor decisions
‚ù?**CRITICAL**: Proceeding with 'C' without fully reading and understanding the next step file
‚ù?**CRITICAL**: Making decisions without complete understanding of step requirements and protocols

## WORKFLOW COMPLETE:

This is the final step of the Architecture workflow. The user now has a complete, validated architecture document ready for AI agent implementation.

The architecture will serve as the single source of truth for all technical decisions, ensuring consistent implementation across the entire project development lifecycle.

## On Complete

Run: `python3 {project-root}/_bmad/scripts/resolve_customization.py --skill {skill-root} --key workflow.on_complete`

If the resolved `workflow.on_complete` is non-empty, follow it as the final terminal instruction before exiting.

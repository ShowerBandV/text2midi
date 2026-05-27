# Step 4: Testing

Display: `Orientation â†?Walkthrough â†?Detail Pass â†?[Testing]`

## Follow Global Step Rules in SKILL.md

- This is **experiential**, not analytical. The detail pass asked "did you think about X?" â€?this says "you could see X with your own eyes."
- Do not prescribe. The human decides whether observing the behavior is worth their time. Frame suggestions as options, not obligations.
- Do not duplicate CI, test suites, or automated checks. Assume those exist and work. This is about manual observation â€?the kind of confidence-building no automated test provides.
- If the change has no user-visible behavior, say so explicitly. Do not invent observations.

## IDENTIFY OBSERVABLE BEHAVIOR

Scan the diff and spec for changes that produce behavior a human could directly observe. Categories to look for:

- **UI changes** â€?new screens, modified layouts, changed interactions, error states
- **CLI/terminal output** â€?new commands, changed output, new flags or options
- **API responses** â€?new endpoints, changed payloads, different status codes
- **State changes** â€?database records, file system artifacts, config effects
- **Error paths** â€?bad input, missing dependencies, edge conditions

For each observable behavior, determine:

1. **What to do** â€?the specific action (command to run, button to click, request to send)
2. **What to expect** â€?the observable result that confirms the change works
3. **Why bother** â€?one phrase connecting this observation to the change's intent (omit if obvious from context)

Target 2â€? suggestions for a typical change. If more than 5 qualify, prioritize by how much confidence the observation provides relative to effort. A change with zero observable behavior is fine â€?do not pad with trivial observations.

## PRESENT

Output as a single message:

```
Orientation â†?Walkthrough â†?Detail Pass â†?[Testing]
```

Then the testing suggestions using this format:

```
### How to See It Working

**{Brief description}**
Do: {specific action}
Expect: {observable result}

**{Brief description}**
Do: {specific action}
Expect: {observable result}
```

Include code blocks for commands or requests where helpful.

If the change has no observable behavior, replace the suggestions with:

```
### How to See It Working

This change is internal â€?no user-visible behavior to observe. The diff and tests tell the full story.
```

### Closing

End the message with:

```
---

You've seen the change and how to verify it. When you're ready to make a call, just say so.
```

## NEXT

When the human signals they're ready to make a decision about this {change_type}, read fully and follow `./step-05-wrapup.md`

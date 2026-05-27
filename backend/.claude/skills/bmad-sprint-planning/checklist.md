# Sprint Planning Validation Checklist

## Core Validation

### Complete Coverage Check

- [ ] Every epic found in epic\*.md files appears in sprint-status.yaml
- [ ] Every story found in epic\*.md files appears in sprint-status.yaml
- [ ] Every epic has a corresponding retrospective entry
- [ ] No items in sprint-status.yaml that don't exist in epic files

### Parsing Verification

Compare epic files against generated sprint-status.yaml:

```
Epic Files Contains:                Sprint Status Contains:
âœ?Epic 1                            âœ?epic-1: [status]
  âœ?Story 1.1: User Auth              âœ?1-1-user-auth: [status]
  âœ?Story 1.2: Account Mgmt           âœ?1-2-account-mgmt: [status]
  âœ?Story 1.3: Plant Naming           âœ?1-3-plant-naming: [status]
                                      âœ?epic-1-retrospective: [status]
âœ?Epic 2                            âœ?epic-2: [status]
  âœ?Story 2.1: Personality Model      âœ?2-1-personality-model: [status]
  âœ?Story 2.2: Chat Interface         âœ?2-2-chat-interface: [status]
                                      âœ?epic-2-retrospective: [status]
```

### Final Check

- [ ] Total count of epics matches
- [ ] Total count of stories matches
- [ ] All items are in the expected order (epic, stories, retrospective)

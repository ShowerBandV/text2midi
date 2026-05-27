---
name: Drift
status: final
sources:
  - {planning_artifacts}/prds/drift-2026-03-12/prd.md
updated: 2026-04-02
---

# Drift â€?Experience Spine

> Illustrative example. Single-surface responsive web. shadcn/ui on Next.js + Tailwind. Paired with `design-example-shadcn.md` (Drift DESIGN.md). Demonstrates: component-library inheritance, keyboard-first interaction primitives, the "shadcn + brand-layer" pattern that covers most modern web SaaS.

## Foundation

Single-surface responsive web. shadcn/ui on Next.js 15+ with Tailwind CSS. The component library does most of the work; brand discipline is "respect the defaults except where the brand layer overrides them." `DESIGN.md` is the visual identity reference and names the override surface; this spine is the experience. Single-tenant per project; users can belong to multiple projects but each project is a self-contained workspace.

## Information Architecture

| Surface | Reached from | Purpose |
|---|---|---|
| Today | App open / `g t` | Current focus, in-progress tasks pulled from all projects |
| Projects | Sidebar / `g p` | List of active and archived projects |
| Project detail | Projects row / `g 1`â€“`g 9` | Tasks in this project, organized by lane |
| Search | `âŒ˜K` / `Ctrl+K` | Command palette â€?surface, navigate, act |
| Settings | Avatar menu | Account, theme, keyboard shortcuts, billing |

Sidebar collapses to icons on `md`; becomes a `Sheet` on `sm`. Modal stacks one level deep (e.g., open `Dialog` on top of a surface, never on top of another dialog).

â†?Composition reference: `mockups/today.html`, `mockups/project-detail.html`, `mockups/command-palette.html`. Spine wins on conflict.

## Voice and Tone

Microcopy. Brand voice and aesthetic posture live in `DESIGN.md`.

| Do | Don't |
|---|---|
| "What are you working on?" | "Let's get productive! ðŸš€" |
| "3 tasks in motion" | "You have 3 active items." |
| "Closed. Nice work." | "Task completed successfully âœ? |
| "Nothing in motion. Pick something." | "No active tasks. Click below to get started!" |
| Manager-facing: counts and verbs. Employee-facing: same. | Different tone per audience â€?Drift talks to everyone the same way. |

## Component Patterns

Behavioral. Visual specs live in `DESIGN.md.Components` (or in shadcn defaults, when inherited).

| Component | Use | Behavioral rules |
|---|---|---|
| Task row | Projects, Today | Click anywhere on row opens edit dialog. Checkbox toggles done state with optimistic update. Hover reveals quick-actions (`focus`, `defer`, `archive`). |
| Focus card | Today, Project detail | At most one focus card per surface â€?the task or project marked with `focus` state. `f` keyboard shortcut sets focus on the active row. |
| Command palette | Global (âŒ˜K) | Fuzzy search across all projects, tasks, and commands. `Enter` fires the highlighted result. `â†’` previews a result. Escape closes. |
| Project header | Project detail | Inline-editable title (click to edit, blur to save). Status pill: active / archived / done. |
| Empty state | Anywhere | shadcn's empty pattern + one Drift-specific sentence. `display-sm` for the headline, body text below, single primary action. |

## State Patterns

| State | Surface | Treatment |
|---|---|---|
| Cold app load | Today | shadcn `Skeleton` rows (4-6) match expected layout. Resolves on data. |
| No focus | Today | `display-sm`: "Nothing in motion. Pick something." Below: list of in-progress tasks from all projects. |
| Empty project | Project detail | `display-sm`: "{Project title} is empty." Body: "Add a first task to get going." Single primary button. |
| Command palette no matches | âŒ˜K | "No matches. Start typing a task or project name, or pick an action below." Followed by 4-5 common commands. |
| Offline | Global (status bar) | shadcn `Toast` once: "You're offline. Changes will sync when you reconnect." Local writes continue. |
| Permission denied | Projects (others' private) | Surface hidden from sidebar. No "blocked" screen. |
| Stale data | Project detail | If background refresh detects changes, shadcn `Toast`: "Updated by {user_name}. Refresh." Manual refresh, no auto. |

## Interaction Primitives

**Keyboard-first.** Drift's primary audience is developers and power users; the keyboard surface is the product, the mouse is fallback.

- `âŒ˜K` / `Ctrl+K` â€?Command palette (universal)
- `g t` / `g p` â€?Go to Today / Projects (vim-style)
- `g 1`â€“`g 9` â€?Go to project by sidebar position
- `f` â€?Set focus on highlighted task/project
- `c` â€?Create new task (context-aware: in the active project)
- `Esc` â€?Close dialogs, exit edit mode, clear command palette
- `/` â€?Focus search in current surface

**Mouse:** click to act, drag deferred to v2. Hover reveals row actions on `md+` (touch users tap to reveal).

**Banned everywhere:** infinite scroll (pagination only), drag-to-reorder in v1, hover-only affordances on `sm` viewports, modal stacks > 1 level deep.

## Accessibility Floor

Behavioral. Visual contrast lives in `DESIGN.md` (inherits shadcn's WCAG AA-compliant defaults; brand overrides verified to maintain ratios).

- WCAG 2.2 AA across the responsive web surface.
- Screen reader announces page surface on navigation: "Today, focus surface" / "Project: {name}, task list, {N} tasks."
- Keyboard shortcuts available without modifier on most surfaces (vim-style `g t` etc.) â€?users with motor-control limitations get the same surface as power users.
- `Tab` order matches reading order on every surface. `Esc` always closes the topmost modal/popover.
- Command palette is fully keyboard-operable; results announce as they update via `aria-live`.
- Focus rings inherit shadcn's `ring` token â€?visible at AA contrast against `background`.

## Responsive & Platform

| Breakpoint | Behavior |
|---|---|
| `â‰?lg` (1024px+) | Sidebar visible. Today is a 2-column layout: focus + in-motion list. |
| `md` (768â€?023px) | Sidebar collapses to icons. Today stacks to single column. |
| `< md` (`sm`) | Sidebar becomes a `Sheet` triggered from top bar. Command palette opens fullscreen. |

Drift is responsive web, not a native mobile app. The product works on phones for read + simple-edit, but the primary surface is desktop / laptop.

## Inspiration & Anti-patterns

- **Lifted from Linear:** the keyboard-first discipline. `âŒ˜K` is the command center; vim-style nav (`g t`); no drag for primary navigation; status pill vocabulary.
- **Lifted from Notion:** inline-editable titles. Click-to-edit on project header, blur to save. No edit/view mode toggle.
- **Lifted from shadcn:** the entire surface vocabulary. Drift's brand is *what we add to shadcn*, not a from-scratch design system. This is a deliberate posture, not a shortcut.
- **Rejected â€?Streaks, badges, achievement notifications:** Drift is a tool, not a habit app. Task closure is its own reward; no celebratory animation, no "ðŸŽ‰ 5-day streak!" toast.
- **Rejected â€?AI-suggested next tasks:** Drift surfaces what's in motion, doesn't tell the user what to work on. The user picks focus; the tool surfaces consequences.
- **Rejected â€?Multi-column kanban as default project view:** lists are linear; kanban hides progress behind columns. Optional v2; not the default.

## Key Flows

### Flow 1 â€?Morning focus (Sarah, solo founder, 8:45am Tuesday)

1. Sarah opens Drift in a browser tab.
2. App loads Today. `display-sm`: "Welcome back, Sarah." Focus card shows yesterday's marked task â€?"Finish landing page hero copy" â€?still in motion.
3. She hits `âŒ˜K`, types "ship hero", sees the matching task and presses Enter to open it.
4. Inline edit: she updates the task description with two new bullets. Tab + Tab triggers save.
5. **Climax:** Sarah closes the dialog. Today re-renders: focus card still shows the hero task, but now with the updated body text visible at a glance. She doesn't have to navigate anywhere â€?the surface that greeted her now reflects the work she just did. She picks up her coffee and starts writing.

Failure: data save fails â†?shadcn `Toast` (destructive variant): "Couldn't save. Trying again." Inline edit retained; another `Enter` retries.

### Flow 2 â€?Async handoff (Devon and Mara, small remote team, mid-afternoon)

1. Devon finishes wiring up the auth flow and marks the task `done`.
2. Mara, time-zoned three hours ahead and online during overlap, opens Drift.
3. Today loads; her focus is on her own front-end work, but the in-motion list shows Devon's auth task now marked done and one task below it newly assigned to her â€?"Wire up post-auth redirect" â€?that Devon set during checkout.
4. She hits `f` on the row to mark it as her focus, then `Enter` to open it.
5. **Climax:** The focus card swaps. Her surface now shows the post-auth redirect task as the live thing she's working on; the projects sidebar shows the Auth project highlighted; the command palette `âŒ˜K` defaults its first result to "Go to Auth project." The state of the team's progress is *embedded in her surface* â€?no Slack thread to scroll, no status doc to read.

Failure: Devon hadn't actually assigned the follow-up â€?Mara mis-assigned to herself. She hits `Esc`, `f` again to unfocus, and reassigns to Devon. No "are you sure" dialog; Drift trusts the user.

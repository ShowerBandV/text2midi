# Domain Research Step 1: Domain Research Scope Confirmation

## MANDATORY EXECUTION RULES (READ FIRST):

- üõë NEVER generate content without user confirmation

- üìñ CRITICAL: ALWAYS read the complete step file before taking any action - partial understanding leads to incomplete decisions
- üîÑ CRITICAL: When loading next step with 'C', ensure the entire file is read and understood before proceeding
- ‚ú?FOCUS EXCLUSIVELY on confirming domain research scope and approach
- üìã YOU ARE A DOMAIN RESEARCH PLANNER, not content generator
- üí¨ ACKNOWLEDGE and CONFIRM understanding of domain research goals
- üîç This is SCOPE CONFIRMATION ONLY - no web research yet
- ‚ú?YOU MUST ALWAYS SPEAK OUTPUT In your Agent communication style with the config `{communication_language}`

## EXECUTION PROTOCOLS:

- üéØ Show your analysis before taking any action
- ‚ö†Ô∏è Present [C] continue option after scope confirmation
- üíæ ONLY proceed when user chooses C (Continue)
- üìñ Update frontmatter `stepsCompleted: [1]` before loading next step
- üö´ FORBIDDEN to load next step until C is selected

## CONTEXT BOUNDARIES:

- Research type = "domain" is already set
- **Research topic = "{{research_topic}}"** - discovered from initial discussion
- **Research goals = "{{research_goals}}"** - captured from initial discussion
- Focus on industry/domain analysis with web research
- Web search is required to verify and supplement your knowledge with current facts

## YOUR TASK:

Confirm domain research scope and approach for **{{research_topic}}** with the user's goals in mind.

## DOMAIN SCOPE CONFIRMATION:

### 1. Begin Scope Confirmation

Start with domain scope understanding:
"I understand you want to conduct **domain research** for **{{research_topic}}** with these goals: {{research_goals}}

**Domain Research Scope:**

- **Industry Analysis**: Industry structure, market dynamics, and competitive landscape
- **Regulatory Environment**: Compliance requirements, regulations, and standards
- **Technology Patterns**: Innovation trends, technology adoption, and digital transformation
- **Economic Factors**: Market size, growth trends, and economic impact
- **Supply Chain**: Value chain analysis and ecosystem relationships

**Research Approach:**

- All claims verified against current public sources
- Multi-source validation for critical domain claims
- Confidence levels for uncertain domain information
- Comprehensive domain coverage with industry-specific insights

### 2. Scope Confirmation

Present clear scope confirmation:
"**Domain Research Scope Confirmation:**

For **{{research_topic}}**, I will research:

‚ú?**Industry Analysis** - market structure, key players, competitive dynamics
‚ú?**Regulatory Requirements** - compliance standards, legal frameworks
‚ú?**Technology Trends** - innovation patterns, digital transformation
‚ú?**Economic Factors** - market size, growth projections, economic impact
‚ú?**Supply Chain Analysis** - value chain, ecosystem, partnerships

**All claims verified against current public sources.**

**Does this domain research scope and approach align with your goals?**
[C] Continue - Begin domain research with this scope

### 3. Handle Continue Selection

#### If 'C' (Continue):

- Document scope confirmation in research file
- Update frontmatter: `stepsCompleted: [1]`
- Load: `./step-02-domain-analysis.md`

## APPEND TO DOCUMENT:

When user selects 'C', append scope confirmation:

```markdown
## Domain Research Scope Confirmation

**Research Topic:** {{research_topic}}
**Research Goals:** {{research_goals}}

**Domain Research Scope:**

- Industry Analysis - market structure, competitive landscape
- Regulatory Environment - compliance requirements, legal frameworks
- Technology Trends - innovation patterns, digital transformation
- Economic Factors - market size, growth projections
- Supply Chain Analysis - value chain, ecosystem relationships

**Research Methodology:**

- All claims verified against current public sources
- Multi-source validation for critical domain claims
- Confidence level framework for uncertain information
- Comprehensive domain coverage with industry-specific insights

**Scope Confirmed:** {{date}}
```

## SUCCESS METRICS:

‚ú?Domain research scope clearly confirmed with user
‚ú?All domain analysis areas identified and explained
‚ú?Research methodology emphasized
‚ú?[C] continue option presented and handled correctly
‚ú?Scope confirmation documented when user proceeds
‚ú?Proper routing to next domain research step

## FAILURE MODES:

‚ù?Not clearly confirming domain research scope with user
‚ù?Missing critical domain analysis areas
‚ù?Not explaining that web search is required for current facts
‚ù?Not presenting [C] continue option
‚ù?Proceeding without user scope confirmation
‚ù?Not routing to next domain research step

‚ù?**CRITICAL**: Reading only partial step file - leads to incomplete understanding and poor decisions
‚ù?**CRITICAL**: Proceeding with 'C' without fully reading and understanding the next step file
‚ù?**CRITICAL**: Making decisions without complete understanding of step requirements and protocols

## NEXT STEP:

After user selects 'C', load `./step-02-domain-analysis.md` to begin industry analysis.

Remember: This is SCOPE CONFIRMATION ONLY - no actual domain research yet, just confirming the research approach and scope!

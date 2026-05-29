# Chord Progression Knowledge

Edit this file to add new chord templates. The song planner reads this at startup.

## Major key templates (convert degrees to concrete chords in selected key)

bright_pop:      I - V - vi - IV
warm_pop:        I - vi - IV - V
simple_major:    I - IV - V - I
open_folk:       I - V - IV - I
emotional_major: vi - IV - I - V

## Minor key templates

dark_loop:       i - VI - VII - i
epic_minor:      i - VI - III - VII
dark_descending: i - VII - VI - VII
boss_battle:     i - VI - VII - V
classical_minor: i - iv - V - i
tragic_minor:    i - V - VI - iv

## Atmosphere / special

horror_drone:     i
horror_phrygian:  i - bII
lofi_soft:        I - vi - ii - V
lofi_resolution:  ii - V - I - I
eastern_dark:     i - VII - VI - VII
eastern_fantasy:  i - III - VII - i

## Emo / Melancholy

emo_sad:         i - VI - III - VII
emo_emotional:   i - iv - VII - III
emo_pop_punk:    vi - IV - I - V
emo_tension:     i - V - VI - iv
emo_dark:        i - VII - VI - VII

## Quick selection by mood

Bright/happy:       major -> I - V - vi - IV (BPM 120-160)
Warm/healing:       major -> I - vi - IV - V (BPM 80-120)
Sad/lyrical:        minor -> i - VI - III - VII, or vi - IV - I - V in major (BPM 60-100)
Emo/melancholy:     minor -> i - VI - III - VII or i - iv - VII - III (BPM 60-90)
Dark/dungeon:       minor -> i - VI - VII - i (BPM 80-130)
Battle/boss:        minor -> i - VI - VII - V (BPM 130-180)
Epic/adventure:     minor -> i - VI - III - VII (BPM 100-160)
Horror/suspense:    minor -> i or i - bII (BPM 50-90)
Lo-fi/chill:        major -> I - vi - ii - V or ii - V - I - I (BPM 70-100)
East-Asian fantasy: minor -> i - VII - VI - VII (BPM 60-120)

## Rules

1. Decide major or minor first based on mood
2. Use 4-bar loop by default, one chord per bar
3. In loopable music, final chord must naturally return to first chord
4. V -> I (major) or V -> i / VII -> i (minor) for strong return
5. Do NOT output diminished chords, slash chords, or extended chords
6. Output only root-position major/minor triads (e.g. C, Dm, Bb, F#m)
7. For intro: use I/i alone or first 2 chords, slower rhythm
8. For climax: same progression, higher energy, denser arrangement
9. BPM MUST match intent.tempo_preference AND the style's mood character

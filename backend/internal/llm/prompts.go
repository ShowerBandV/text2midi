// Package llm provides prompt templates for the LLM agents.
// Ported from music_agent/prompts/templates.py.
package llm

import (
	"fmt"
	"strings"
)

// noteToSemitoneStr and semitoneToNoteStr are compact maps for the prompts
// to constrain chord root choices.
const noteMapStr = `{"C":0,"C#":1,"Db":1,"D":2,"D#":3,"Eb":3,"E":4,"F":5,"F#":6,"Gb":6,"G":7,"G#":8,"Ab":8,"A":9,"A#":10,"Bb":10,"B":11}`
const noteNamesStr = `["C","C#","D","D#","E","F","F#","G","G#","A","A#","B"]`

// chordKnowledge is an embedded reference for the song planner.
// Condensed from knowledges/chords.md --essential progression templates + rules.
const chordKnowledge = `# Chord Progression Knowledge

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

## Emo / Melancholy --忧郁、悲伤、情感化 (slow tempo, minor key)
emo_sad:         i - VI - III - VII        # 最忧郁的进行，小调+大三度色�?emo_emotional:   i - iv - VII - III        # 更暗的忧郁，iv增加悲伤�?emo_pop_punk:    vi - IV - I - V           # 流行emo，大调关系小调起
emo_tension:     i - V - VI - iv           # 悲剧感，强烈的情感张�?emo_dark:        i - VII - VI - VII        # 压抑下行，适合intro/verse

## Quick selection by mood

Bright/happy:       major ->I - V - vi - IV (BPM 120-160)
Warm/healing:       major ->I - vi - IV - V (BPM 80-120)
Sad/lyrical:        minor ->i - VI - III - VII, or vi - IV - I - V in major (BPM 60-100)
Emo/melancholy:     minor ->i - VI - III - VII or i - iv - VII - III (BPM 60-90)
Dark/dungeon:       minor ->i - VI - VII - i (BPM 80-130)
Battle/boss:        minor ->i - VI - VII - V (BPM 130-180)
Epic/adventure:     minor ->i - VI - III - VII (BPM 100-160)
Horror/suspense:    minor ->i or i - bII (BPM 50-90)
Lo-fi/chill:        major ->I - vi - ii - V or ii - V - I - I (BPM 70-100)
East-Asian fantasy: minor ->i - VII - VI - VII (BPM 60-120)

## Rules
1. Decide major or minor first based on mood
2. Use 4-bar loop by default, one chord per bar
3. In loopable music, final chord must naturally return to first chord
4. V ->I (major) or V ->i / VII ->i (minor) for strong return
5. Do NOT output diminished chords, slash chords, or extended chords
6. Output only root-position major/minor triads (e.g. C, Dm, Bb, F#m)
7. Chord roots must come from this map: ` + "`" + noteMapStr + "`" + `
8. Canonical note names: ` + "`" + noteNamesStr + "`" + `
9. For intro: use I/i alone or first 2 chords, slower rhythm
10. For climax: same progression, higher energy, denser arrangement
11. BPM MUST match intent.tempo_preference AND the style's mood character:
    - melancholic/sad/emo/忧郁 ->BPM 60-90 (slow, do NOT use fast tempo)
    - happy/bright ->BPM 120-160
    - medium/calm ->BPM 80-120
    - epic/battle ->BPM 130-180
`

// instrumentKnowledge is a condensed reference for the arrangement planner.
const instrumentKnowledge = `# Instrument Knowledge

## GM Program Numbers (for midi.program)
Acoustic Grand Piano=0, Electric Piano=5, Vibraphone=11, Pipe Organ=19
Acoustic Guitar(nylon)=24, Acoustic Guitar(steel)=25, Electric Guitar(jazz)=26
Electric Guitar(clean)=27, Electric Guitar(muted)=28, Overdriven Guitar=29
Distortion Guitar=30, Electric Bass(finger)=34, Electric Bass(pick)=35
Slap Bass=36, Synth Bass 1=38, Synth Bass 2=39
Violin=40, Viola=41, Cello=42, Contrabass=43
Tremolo Strings=44, Pizzicato Strings=45, Orchestral Harp=46, Timpani=47
String Ensemble 1=48, String Ensemble 2=49, Synth Strings 1=50
Synth Strings 2=51, Choir Aahs=52, Voice Oohs=53
Trumpet=56, Trombone=57, Tuba=58, French Horn=60
Brass Ensemble=61, Synth Brass 1=62, Synth Brass 2=63
Soprano Sax=64, Alto Sax=65, Tenor Sax=66
Oboe=68, English Horn=69, Bassoon=70, Clarinet=71
Piccolo=72, Flute=73, Recorder=74, Pan Flute=75, Shakuhachi=77
Whistle=78, Lead 1(saw)=80, Lead 2(square)=81, Pad 1(warm)=89
Pad 3(polysynth)=90, Pad 4(choir)=91, Pad 5(bowed)=92
Pad 6(metallic)=93, Pad 7(halo)=94, Pad 8(sweep)=95
FX 1(rain)=96, FX 2(soundtrack)=97, FX 3(crystal)=98
FX 4(atmosphere)=99, FX 5(brightness)=100, FX 6(goblins)=101
FX 7(echoes)=102, FX 8(sci-fi)=103
Kalimba=108, Bagpipe=109, Fiddle=110, Shanai=111
Taiko Drum=116, Melodic Tom=117, Synth Drum=118

## Track role, register, and program suggestions

Drums: channel=9 always, program=null, GM drum pitches (kick=36, snare=38, closed_hat=42, open_hat=46, crash=49)
Bass: register C1-C3. Electric Bass(finger)=34, Synth Bass=38/39, Acoustic Bass=43
Chords/Pad: register C3-C5. Pad(warm)=89, Pad(polysynth)=90, EP=5, Strings=48/49, Guitar=24/25/26
Lead: register C4-C6. Lead(saw)=80, Lead(square)=81, Flute=73, Violin=40, Trumpet=56
Arpeggio: register C3-C6. Harp=46, EP=5, Synth=90/98
Strings: register variable. Ensemble=48/49, Cello=42, Viola=41
Brass: register C3-C5. Ensemble=61, Trumpet=56, Horn=60
Ethnic: Shakuhachi=77, Pan Flute=75, Kalimba=108, Taiko=116

## Arrangement rules
- drums must use channel=9 and program=null
- non-drum tracks must NOT use channel=9
- core tracks (drums, bass, chords, lead) should be present by default
- extra tracks (pad, arp, strings, brass, fx) are allowed when style demands
- do NOT force all 4 core tracks for ambient or sparse styles
`

// noteKnowledge is a condensed reference for note generation.
const noteKnowledge = `# Note Generation Knowledge

## Per-current-chord logic
For each bar, identify the active chord, then generate notes per role:

Bass: root at chord start. Register C1-C3. Durations: 0.25-1 beat (pop/electronic), 1-4 beats (slow/ambient).
Chords: root+third+fifth. Register C3-C5. Durations: block=1-4 beats, arp=0.25-0.5 beats.
Pad: chord tones, long 2-8 beats. Register C3-C6.
Lead: chord tones on strong beats, in-key passing tones on weak beats. Register C4-C6. 0.25-2 beats.
Drums: no chord pitches. Kick=36 on 1&3, Snare=38 on 2&4, Hat=42 on 8ths.

## Register by role
Bass: C1-C3
Low strings/brass: C2-C4
Chords/Pad: C3-C5
Lead: C4-C6
High accents: C5-C7

## Duration by role
Drums: 0.05-0.25 beats
Bass: 0.25-1 beat (pop), 1-4 beats (slow)
Chords: 0.5-4 beats
Pad: 2-8 beats
Lead: 0.25-2 beats
Arp: 0.25-0.5 beats

## Section energy
Intro: low density, long durations, fewer instruments
Main: complete patterns, stable density
Climax: higher register, stronger velocity, denser rhythm, more short values
Ending/Loop: final chord returns to opening chord naturally

## Avoid
- All instruments playing full chords (crowding)
- Bass not following chord roots
- Same rhythmic density across all tracks
- Continuous large leaps in lead
- Pad being too short or choppy
`

// arrangementKnowledge provides section-specific arrangement rules per genre.
// Injected into genre prompts for structured verse/pre-chorus/chorus/bridge handling.
const arrangementKnowledge = `
# Arrangement Rules Per Genre

## Pop Arrangement
- Chord progressions: I-V-vi-IV (bright), vi-IV-I-V (sad), I-IV-vi-V (open)
- Arpeggios: verse = 8th note broken chords, pre-chorus = 16th note ascending, chorus = high register rapid arpeggio background
- Inversions: use for smooth bass lines - C - G/B - Am - Am/G - F - C/E - Dm - G
  Pre-chorus: use G/B ascending
  Chorus: keep bass moving with F/A, G/B
- Non-chord tones: verse = passing/auxiliary tones (safe), pre-chorus = anticipation/suspension (build tension), chorus = appoggiatura (emotional peak)
- Section energy: verse=low, pre-chorus=medium, chorus=high, bridge=unstable

## Rock Arrangement
- Chords: power chords (root + fifth, no third). Progressions: I-IV-V, I-bVII-IV (Mixolydian), vi-IV-I-V
- Riffs: guitar riffs use root-fifth-octave patterns with palm muting
- Blues elements: b3, b5, b7 (blue notes) are STYLE-DEFINING, not "wrong notes"
- Inversions: rare for guitar. Bass can do inversions: C - G/B - Am - F/A
- Drums: kick on 1&3, snare on 2&4 = ESSENCE of rock
- Section: verse = palm-muted riffs, chorus = open power chords, solo = pentatonic runs

## Chinese Style (国风) Arrangement
- Scales: pentatonic (宫商角徵�?. C�?= C D E G A. 羽调 (minor): A C D E G
- Avoid functional Western harmony. Use open fifths, sus2, sus4 chords instead of triads
- Chord substitutions: replace F with Dm7/F or Fsus2 (F-G-C) to keep pentatonic feel
- Arpeggios: pentatonic glissandi, pipa-style rolling chords, guzheng "sweep" patterns
- Inversions: use for pentatonic bass lines - C - G/B - Am - Em/G (bass: C-B-A-G)
- Non-chord tones / 偏音: verse = pentatonic only (avoid 4 and 7). Pre-chorus = add 变宫 (7) as passing tone. Chorus = add 雅乐 #4 or 燕乐 b7 for color, on weak beats only
- Instruments: pentatonic ornaments, slides, vibrato. Voice leading prioritizes pentatonic flow
`

// styleBPMInfo provides BPM ranges for each style (used in prompt hints).
var styleBPMInfo = map[string]struct {
	minBPM, maxBPM int
}{
	"trap":       {130, 160},
	"boomBap":    {80, 100},
	"drill":      {130, 145},
	"lofi":       {70, 90},
	"westCoast":  {90, 105},
	"jerseyClub": {130, 140},
}

// BuildBeatPrompt builds a single-shot prompt that directly generates
// drum patterns (16-step), bass patterns, and melody notes for a beat.
// randomSeed ensures each call produces different musical ideas.
func BuildBeatPrompt(style, styleDescription, userPrompt string, bpm, bars int, key, randomSeed string) string {
	bpmRange := ""
	if info, ok := styleBPMInfo[style]; ok {
		bpmRange = fmt.Sprintf(" (valid BPM range: %d-%d)", info.minBPM, info.maxBPM)
	}

	return fmt.Sprintf(`You are a world-class hip-hop music producer. The user will describe a musical idea, and you need to convert it into a strict JSON object that drives a MIDI generation engine.

You must follow these rules:
1. style must be one of: trap, boomBap, drill, lofi, westCoast, jerseyClub.
2. key must be in standard format like "C minor".
3. drumPattern is an object containing kick, snare, hihat. Each value is an array of length 16, representing a 16th-note step sequence for one bar. 1 = trigger, 0 = no trigger.
4. bassPattern is an array of length 16, representing the root note offset for each 16th-note step of one bar. 0 = root, +7 = fifth, +12 = octave root.
5. melodyNotes is an array of note objects. Each note has: start (16th-note position), duration (number of 16th-notes), pitch (MIDI pitch 21-108 in key scale), velocity (1-127, varying velocities create expression), articulation (optional: "legato" for smooth connected, "staccato" for short detached, "accent" for emphasized, "normal" default).
6. performance (optional): an object with expression data applied across the whole track. Fields:
   - expressionCurve: array of {bar, value} where value is CC11 expression (0-127). Controls overall loudness contour (crescendo/diminuendo).
   - sustainPedal: array of {bar, on (bool)} for CC64 sustain pedal events.
   - pitchBend: array of {bar, value} where value is pitch bend 0-16383 (8192=center). Positive values bend up, negative bend down for expression.
   - globalDynamics: number 0.0-1.0 for overall intensity scaling.
7. melody velocity must vary --not all notes at same velocity. Use higher velocity (100-120) for accented/strong beats, lower (60-85) for passing notes.
8. Ending bars (last 2 bars) should have progressively lower velocity for a natural fade-out.
9. Random seed: %s. Each seed produces unique patterns.
10. All output must be valid JSON. Do NOT include any extra text, explanation, or Markdown.

Style: %s
Style description: %s
BPM: %d%s
Key: %s
Bars: %d

User input: %s

Return JSON only, in this exact format:
{
  "style": "...",
  "drumPattern": {
    "kick": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
    "snare": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
    "hihat": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
  },
  "bassPattern": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
  "melodyNotes": [
    {"start": 0, "duration": 2, "pitch": 64, "velocity": 100, "articulation": "accent"},
    {"start": 4, "duration": 1, "pitch": 67, "velocity": 76, "articulation": "legato"},
    {"start": 8, "duration": 3, "pitch": 71, "velocity": 72, "articulation": "legato"}
  ],
  "performance": {
    "expressionCurve": [{"bar": 0, "value": 60}, {"bar": 4, "value": 80}, {"bar": 8, "value": 100}, {"bar": 14, "value": 50}, {"bar": 15, "value": 25}],
    "sustainPedal": [{"bar": 0, "on": true}, {"bar": 15.5, "on": false}],
    "globalDynamics": 0.85
  }
}`, randomSeed, style, styleDescription, bpm, bpmRange, key, bars, userPrompt)
}

// BuildRockPatternPrompt builds a single-shot prompt for rock/metal style generation.
// Outputs the same JSON structure as BuildBeatPrompt but with 4/4 rock-appropriate patterns:
// kick on 1&3, snare on 2&4, crash on fills, power-chord guitar, blues-scale lead.
func BuildRockPatternPrompt(style, styleDescription, userPrompt string, bpm, bars int, key, randomSeed string) string {
	return fmt.Sprintf(`You are a world-class rock musician (drummer + guitarist + bassist). The user will describe a musical idea, and you need to convert it into a strict JSON object that drives a MIDI generation engine.

Rock arrangement knowledge:
- Power chords (root + fifth, no third) are the foundation. Progressions: I-IV-V, I-bVII-IV (Mixolydian), vi-IV-I-V.
- Blues elements: b3, b5, b7 are STYLE-DEFINING --use them freely on strong beats.
- Verse: palm-muted riffs, tight drumming. Chorus: open power chords, full energy.
- Bass inversions for flow: C - G/B - Am - F/A. Guitar stays on root-position power chords.
- Riffs should be rhythmic and memorable, using root-fifth-octave patterns.

You must follow these rules:
1. Key must be in standard format like "E minor".
2. DrumPattern is an object containing kick, snare, hihat. Each value is an array of length 16, representing a 16th-note step sequence for one bar. 1 = trigger, 0 = no trigger.
3. Kick must hit on beats 1 AND 3 (steps 0 and 8). Add extra kicks for fills and energy.
4. Snare must hit on beats 2 AND 4 (steps 4 and 12). That is the ESSENCE of rock drumming.
5. Hi-hat plays steady 8th notes (steps 0,2,4,6,8,10,12,14). Crash cymbal can replace the hi-hat on beat 1 of section changes by setting step 0 to 2.
6. bassPattern is an array of length 16, representing the root note offset for each 16th-note step. 0 = root, +7 = fifth, +12 = octave root. Use root-fifth-root patterns for driving rock feel.
7. melodyNotes is an array of note objects. Each note has: start (16th-note position), duration (# of 16th-notes), pitch (MIDI pitch 21-108 in key, using pentatonic/blues scales), velocity (1-127), articulation ("power_chord" for rhythm guitar, "bend" for lead guitar, "legato", "staccato", "accent", "normal").
8. melody velocity must vary --use higher velocity (110-127) for accented downbeats, lower (70-90) for fills and verses.
9. Random seed: %s. Each seed produces unique patterns.
10. All output must be valid JSON. Do NOT include any extra text, explanation, or Markdown.

Style: %s
Style description: %s
BPM: %d
Key: %s
Bars: %d

User input: %s

Return JSON only, in this exact format:
{
  "style": "...",
  "drumPattern": {
    "kick": [1,0,0,0,1,0,0,0,1,0,0,0,1,0,0,0],
    "snare": [0,0,0,0,1,0,0,0,0,0,0,0,1,0,0,0],
    "hihat": [1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0]
  },
  "bassPattern": [0,0,0,0,7,0,0,0,0,0,0,0,7,0,0,0],
  "melodyNotes": [
    {"start": 0, "duration": 4, "pitch": 64, "velocity": 115, "articulation": "power_chord"},
    {"start": 4, "duration": 2, "pitch": 71, "velocity": 90, "articulation": "bend"},
    {"start": 8, "duration": 4, "pitch": 67, "velocity": 100, "articulation": "accent"}
  ],
  "performance": {
    "expressionCurve": [{"bar": 0, "value": 80}, {"bar": 4, "value": 110}, {"bar": 7, "value": 127}, {"bar": 14, "value": 90}],
    "sustainPedal": [{"bar": 0, "on": true}, {"bar": 15.5, "on": false}],
    "globalDynamics": 0.9
  }
}`, randomSeed, style, styleDescription, bpm, key, bars, userPrompt)
}

// BuildMetalPatternPrompt builds a single-shot prompt for metal style generation.
// Double bass, blast beats, down-tuned power chords, Phrygian/harmonic minor aggression.
func BuildMetalPatternPrompt(style, styleDescription, userPrompt string, bpm, bars int, key, randomSeed string) string {
	return fmt.Sprintf(`You are a metal musician (drummer + guitarist + bassist). The user describes a musical idea; convert it into strict JSON for a MIDI generation engine.

Rules:
1. DrumPattern has kick, snare, hihat arrays of length 16 (16th-note steps). 1=trigger, 0=no trigger.
2. Double bass: kick plays steady 16th notes (steps 0-15 all 1) or rapid alternating patterns. Kick density must be HIGH (8-16 hits per bar).
3. Snare: either on beats 2&4 (steps 4,12) for groove, or constant 16th-note blast beats (steps 0-15). Use 2 for accented blast hits.
4. Hi-hat: steady 8th notes or ride cymbal. Crash on downbeats (step 0 and 8).
5. bassPattern is 16 elements: root note offsets. 0=root, +7=fifth, +12=octave. Use root-fifth power motion, often following kick rhythm.
6. melodyNotes array: each note has start (16th position), duration (# of 16ths), pitch (21-108, use Phrygian/harmonic minor scales), velocity (1-127), articulation ("power_chord", "tremolo", "palm_mute", "legato", "staccato", "accent", "normal").
7. High velocity (110-127) for accents and downbeats. Aggressive, high energy throughout.
8. GM Instruments: Distortion Guitar=30, Overdriven Guitar=29, Electric Bass(pick)=35, Synth Bass 1=38, String Ensemble=49.
9. Optional sections field: include energy/pattern variations across sections (intro, verse, chorus, breakdown).
10. Random seed: %s. Each seed produces unique patterns.
11. Valid JSON only. No extra text.

Style: %s | %s
BPM: %d | Key: %s | Bars: %d
User: %s

JSON format:
{
  "style": "...",
  "drumPattern": {
    "kick": [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1],
    "snare": [0,0,0,0,1,1,1,1,0,0,0,0,1,1,1,1],
    "hihat": [1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0]
  },
  "bassPattern": [0,0,0,0,7,0,0,0,0,0,0,0,7,0,0,0],
  "melodyNotes": [
    {"start": 0, "duration": 2, "pitch": 52, "velocity": 120, "articulation": "power_chord"},
    {"start": 4, "duration": 1, "pitch": 57, "velocity": 110, "articulation": "palm_mute"},
    {"start": 8, "duration": 4, "pitch": 55, "velocity": 115, "articulation": "tremolo"}
  ],
  "performance": {
    "expressionCurve": [{"bar": 0, "value": 100}, {"bar": 4, "value": 120}, {"bar": 8, "value": 127}, {"bar": 14, "value": 100}],
    "globalDynamics": 0.95
  }
}`, randomSeed, style, styleDescription, bpm, key, bars, userPrompt)
}

// BuildPopPatternPrompt builds a single-shot prompt for pop music generation.
// Four-on-the-floor, catchy melodies, bright chords, danceable groove.
func BuildPopPatternPrompt(style, styleDescription, userPrompt string, bpm, bars int, key, randomSeed string) string {
	return fmt.Sprintf(`You are a pop music producer (drummer + keyboardist + bassist). The user describes a musical idea; convert it into strict JSON for a MIDI generation engine.

Arrangement knowledge reference:
- STRUCTURE: Intro(2bars, sparse) -> Verse(4-8bars, low energy) -> Pre-Chorus(2-4bars, build) -> Chorus(4-8bars, full) -> Bridge(4bars, contrast) -> Final Chorus(out)。
- DYNAMIC CONTRAST: Verse uses 30%% energy, Chorus uses 100%%. Instruments enter GRADUALLY: piano first, then bass, then drums, then strings.
- Verse: 8th note broken chords, low velocity, passing/auxiliary tones. Keep space for vocal.
- Pre-chorus: ascending 16th notes, G/B inversion, anticipation/suspension, hi-hat opens up.
- Chorus: ALL instruments. Block chords, loud, appoggiatura for emotional peak, crash cymbal on beat 1.
- Bridge: strip back to piano/vocal, change chord progression for contrast.
- Chord progressions: I-V-vi-IV (bright), vi-IV-I-V (sad), I-IV-vi-V (open).
- Inversions for smooth bass: C - G/B - Am - Am/G - F - C/E - Dm - G.
- Drums: kick on 1&3, snare on 2&4 (backbeat), hi-hat 8th notes. This is ESSENTIAL for pop.
- Bass locks with kick drum root on beat 1.

Rules:
1. DrumPattern has kick, snare, hihat arrays of length 16 (16th-note steps). 1=trigger, 0=no trigger.
2. Kick: four-on-the-floor pattern (steps 0,4,8,12 = all 1). Add occasional extra kicks for variation.
3. Snare: beats 2 and 4 (steps 4 and 12). Crisp and consistent.
4. Hi-hat: 8th notes (steps 0,2,4,6,8,10,12,14) with occasional open hat (step value 2) on offbeats for groove.
5. bassPattern: 16 elements. 0=root, +7=fifth, +12=octave. Simple root motion, often locking with kick.
6. melodyNotes array: each note has start (16th position), duration (# of 16ths), pitch (21-108, use major/lydian scales, bright tones), velocity (1-127), articulation ("lead", "chord", "pluck", "legato", "staccato", "accent", "normal").
7. Melodies should be catchy and singable. Use varied velocity (80-110) with higher on chorus beats.
8. Keep it bright and uplifting. Avoid dissonance.
9. Random seed: %s. Each seed produces unique patterns.
10. Valid JSON only. No extra text.

Style: %s | %s
BPM: %d | Key: %s | Bars: %d
User: %s

JSON format:
{
  "style": "...",
  "drumPattern": {
    "kick": [1,0,0,0,1,0,0,0,1,0,0,0,1,0,0,0],
    "snare": [0,0,0,0,1,0,0,0,0,0,0,0,1,0,0,0],
    "hihat": [1,0,1,0,1,1,1,0,1,0,1,0,1,1,1,0]
  },
  "bassPattern": [0,0,0,0,0,0,0,0,7,0,0,0,7,0,0,0],
  "melodyNotes": [
    {"start": 0, "duration": 2, "pitch": 72, "velocity": 100, "articulation": "lead"},
    {"start": 4, "duration": 1, "pitch": 76, "velocity": 85, "articulation": "staccato"},
    {"start": 8, "duration": 4, "pitch": 79, "velocity": 95, "articulation": "legato"}
  ],
  "performance": {
    "expressionCurve": [{"bar": 0, "value": 70}, {"bar": 4, "value": 90}, {"bar": 8, "value": 110}, {"bar": 14, "value": 80}],
    "sustainPedal": [{"bar": 0, "on": true}, {"bar": 15.5, "on": false}],
    "globalDynamics": 0.85
  }
}`, randomSeed, style, styleDescription, bpm, key, bars, userPrompt)
}

// BuildIntentParserPrompt builds the prompt for the intent parser agent.
func BuildIntentParserPrompt(userPrompt string, enforceCoreTracks bool, maxDurationSeconds *int) string {
	var trackRule string
	if enforceCoreTracks {
		trackRule = "- requested_tracks must include at least drums,bass,chords,lead."
	} else {
		trackRule = "- requested_tracks should be decided by musical intent; do not force fixed instrument sets."
	}

	var durationRule string
	if maxDurationSeconds != nil {
		durationRule = fmt.Sprintf("- intent.duration_seconds must be <= %d. If user asks longer, clamp to %d.", *maxDurationSeconds, *maxDurationSeconds)
	} else {
		durationRule = "- intent.duration_seconds should be reasonable for loopable BGM and default to 30 when unclear."
	}

	return fmt.Sprintf(`You are an Intent Parser for a MIDI music generation pipeline.
Return JSON only. No markdown, no explanations.

User prompt: %s

Output fields:
- task_type
- user_prompt
- intent.style (list)
- intent.mood (list)
- intent.use_case
- intent.duration_seconds (number)
- intent.loopable (bool)
- intent.complexity
- intent.requested_tracks (list)
- intent.tempo_preference (slow|medium|fast|very_fast)
- intent.must_have (list)
- intent.avoid (list)
- intent.feature_vector (object, see dimensions below)

Feature Vector (all 0.0-1.0 scale):
  - darkness: 0=bright/major/high, 1=dark/minor/low
  - energy: 0=calm/sparse, 1=explosive/dense
  - acousticness: 0=synthetic/electronic, 1=acoustic/organic
  - density: 0=sparse, 1=dense (notes per bar, layers)
  - rhythmic_complexity: 0=steady 4/4, 1=complex syncopation
  - tension: 0=consonant (triads), 1=dissonant (aug/dim/extended)
  - lo_fi: 0=clean/precise, 1=lo-fi/warped/vintage

Rules:
- Output must follow this exact nested JSON shape (no dotted keys):
{
  "task_type": "generate_music",
  "user_prompt": "...",
  "intent": {
    "style": ["..."],
    "mood": ["..."],
    "use_case": "...",
    "duration_seconds": 30,
    "loopable": true,
    "complexity": "medium",
    "requested_tracks": ["drums", "bass", "chords", "lead"],
    "tempo_preference": "fast",
    "must_have": ["drums", "bass", "chords", "lead"],
    "avoid": [],
    "feature_vector": {
      "darkness": 0.5,
      "energy": 0.5,
      "acousticness": 0.5,
      "density": 0.5,
      "rhythmic_complexity": 0.5,
      "tension": 0.5,
      "lo_fi": 0.5
    }
  }
}
- Never output keys like "intent.style" or "intent.mood".
- feature_vector must always be present. Infer each dimension from the user's description.
%s
- %s
- Tempo preference mapping:
  - emo / melancholy / 忧郁 / 悲伤 / 抒情 ->"slow" (BPM 60-90)
  - sad / lyrical / ballad ->"slow" or "medium"
  - happy / upbeat / bright ->"fast" or "very_fast"
  - calm / relaxed / chill ->"medium"
  - epic / battle / intense ->"fast" or "very_fast"
  - default loopable BGM ->"medium"
- Fill missing values with reasonable defaults.`, userPrompt, trackRule, durationRule)
}

// BuildSongPlannerPrompt builds the prompt for the song planner agent.
func BuildSongPlannerPrompt(intentJSON string) string {
	return fmt.Sprintf(`You are a Song Planner for a MIDI music generation pipeline.
Return JSON only. No markdown, no explanations.

Input intent: %s

Chord progression knowledge:
%s

Output top-level key: song_plan
Required song_plan fields:
title,bpm,time_signature,key,total_bars,estimated_duration_seconds,
loopable,global_style,sections,chord_progression

Constraints:
- time_signature must be 4/4 in MVP
- bpm should match tempo_preference in intent
- total_bars must be one of 8,12,16,24,32
- chord_progression should contain at least 4 bars and be loopable
- Do not repeat exactly the same melodic contour template across runs.
- Create a fresh composition each run while keeping style consistency.
- Vary harmonic rhythm, section contrast, and motif development naturally.
- Chord symbol compatibility:
  - Allowed: triads (C, Dm, Bb), 7th chords (Cmaj7, Dm7, G7, Am7), 9th chords (Cmaj9, Dm9, G9)
  - Allowed: suspended chords (Csus4, Dsus2), diminished (Cdim, Cdim7), augmented (Caug), half-dim (Cm7b5)
  - Allowed: slash chords for bass movement (C/G, Dm/F, Am/C, G/B)
  - NOT allowed: chords with altered tensions in parentheses (e.g. C7(#9) — write C7#9 instead)
  - Chord roots must come from this map: %s
  - Canonical note names used by the system: %s
- Use the chord progression knowledge above as the primary decision guide:
  - infer major/minor mode from intent style, mood, and use_case
  - choose a chord-degree template that matches the requested style and emotion
  - convert roman-numeral degrees into concrete chord symbols in the selected key
  - keep chord_progression section-aware and loopable
  - Use extended chords (7ths, 9ths, sus) naturally when the style calls for them (jazz/pop/R&B = rich harmony, rock = power chords)
- chord_progression must contain concrete chord symbols only, not roman numerals.
- Output must follow this exact JSON shape:
{
  "song_plan": {
    "title": "...",
    "bpm": 140,
    "time_signature": {"numerator": 4, "denominator": 4},
    "key": {"root": "D", "mode": "minor", "scale": "natural_minor"},
    "total_bars": 16,
    "estimated_duration_seconds": 27.4,
    "loopable": true,
    "global_style": {"primary": "cyberpunk", "energy": 0.8, "darkness": 0.7, "brightness": 0.3},
    "sections": [{"id": "intro", "name": "Intro", "start_bar": 0, "length_bars": 4, "energy": 0.4}],
    "chord_progression": [{"bar": 0, "chord": "Dm"}, {"bar": 1, "chord": "Bb"}, {"bar": 2, "chord": "C"}, {"bar": 3, "chord": "A"}]
  }
}`, intentJSON, chordKnowledge, noteMapStr, noteNamesStr)
}

// BuildArrangementPlannerPrompt builds the prompt for the arrangement planner agent.
func BuildArrangementPlannerPrompt(intentJSON, songPlanJSON string, enforceCoreTracks bool) string {
	var trackRule string
	if enforceCoreTracks {
		trackRule = "- tracks must include core tracks: drums,bass,chords,lead"
	} else {
		trackRule = "- tracks should be decided by style and user intent; do not force fixed core instruments"
	}
	return fmt.Sprintf(`Instrument knowledge:
%s

You are an Arrangement Planner for a MIDI music generation pipeline.
Return JSON only. No markdown, no explanations.

Input intent: %s
Input song_plan: %s

Output format:
{"arrangement": {"tracks": {...} } }

Rules:
%s
- extra tracks are allowed when needed by style or user request
- drums must use channel=9 and program=null
- non-drum tracks must not use channel=9
- each track must include:
  id,name,role,enabled,is_core_track,generation_strategy,midi,mix,style,sections
- Encourage compositional variation across runs:
  - Do not lock to one fixed melodic direction.
  - Prefer different section-level density and role interaction per run.
  - Keep requested style, but allow creative arrangement decisions.
- Output must follow this exact nested shape:
{
  "arrangement": {
    "tracks": {
      "drums": {
        "id": "drums",
        "name": "Drums",
        "role": "rhythm",
        "enabled": true,
        "is_core_track": true,
        "generation_strategy": "drum_generator",
        "midi": {"channel": 9, "program": null},
        "mix": {"volume": 105, "pan": 64},
        "style": {"pattern_type": "driving_electronic", "density": "high"},
        "sections": {"intro": {"active": true, "density": "low"}, "main_a": {"active": true, "density": "medium"}, "main_b": {"active": true, "density": "high"}}
      }
    }
  }
}`, instrumentKnowledge, intentJSON, songPlanJSON, trackRule)
}

// BuildTrackNoteGeneratorPrompt builds the prompt for per-track note generation.
// This is for the "llm" note generation mode (future use; currently rule-based).
func BuildTrackNoteGeneratorPrompt(songPlanStr, trackJSON string) string {
	return fmt.Sprintf(`You are a professional music producer generating note events for a single instrument track.

%s

Use the note generation knowledge above as your PRIMARY DECISION GUIDE.

Input song_plan: %s
Input track: %s

ROLE-SPECIFIC RULES:
- If role is bass: play chord roots on strong beats, use octave jumps, stay in MIDI 28-48.
- If role is chords: play full triad at chord changes, arpeggiated or block voicing, MIDI 48-72.
- If role is pad: sustain long notes, overlap chord tones, MIDI 48-72.
- If role is lead: SINGABLE melody, stepwise motion, varied phrasing, MIDI 60-84.
- If role is drums: GM drum map (kick=36, snare=38, hi-hat=42), lock with tempo.

CRITICAL:
- Derive notes from the active chord at each beat/bar.
- Vary durations: mix short(0.125) medium(0.5-0.75) long(1.0-2.0).
- Use RESTS. Leave 20-40%% empty space. Do NOT fill every beat.
- End phrases with cadence. Make loop endings lead naturally back to beginning.
- Avoid all tracks using same rhythmic density simultaneously.

Output EXACTLY this JSON shape, nothing else:
{
  "events": [
    {"type":"note","pitch":60,"start_beat":0.0,"duration_beat":0.5,"velocity":96}
  ]
}`, noteKnowledge, songPlanStr, trackJSON)
}

// BuildNoteSequencePrompt builds a prompt for direct note-level generation.
// Unlike BuildBeatPrompt (16-step grid), this allows arbitrary fractional beat positions,
// enabling natural phrasing, rests, and expressive timing.
// Used for melody and bass lines that need musicality.
//
// Parameters:
//   - instrument: "lead" / "bass" / "pipa" / "guzheng" etc.
//   - chordProg: JSON array of chord changes
//   - totalBeats: total duration in beats
//   - styleDesc: natural language style description
//   - featureVec: JSON of the feature vector
func BuildNoteSequencePrompt(instrument, key, scale, chordProg, styleDesc, featureVec string, bpm, totalBeats int, randomSeed string) string {
	prompt := `You are a professional composer writing a single instrument track.
Return a JSON object with an "events" array. No markdown, no explanations.

Instrument: __INSTRUMENT__
Key: __KEY__
Scale: __SCALE__
BPM: __BPM__
Total duration: __BEATS__ beats
Chord progression: __CHORDS__
Style: __STYLE__
Feature vector: __FEATURES__
Random seed: __SEED__

CRITICAL — Compose like a professional songwriter:
1. VOCAL MELODY FIRST: Write a SINGABLE melody. Imagine a singer performing it. Melodies should have a clear emotional arc: start in a comfortable range, build tension, resolve.
2. PHRASE STRUCTURE: 4-bar phrases. Shape = statement(bar1) -> repeat with variation(bar2) -> contrast(bar3) -> resolve(bar4). This A-B-A-C structure creates memorability.
3. STEPWISE MOTION: Most intervals should be 1-2 semitones (stepwise). Limit leaps to 5 semitones max. After a leap, immediately step back in the opposite direction.
4. RHYTHMIC IDENTITY: Each phrase should have a recognizable rhythm pattern. Repeat the rhythm of bar1 in bar2, but change the pitches. This is how hooks are made.
5. RESTS ARE MUSIC: Do NOT fill every beat. Leave 20-40% of each bar as silence. Breathe. Let the listener anticipate the next note.
6. NOTE DURATIONS: Mix long notes (1.0-2.0 beats, for important syllables) with short notes (0.125-0.25, for passing). The longest note in each phrase should fall on the climax.
7. PITCH RANGE: Stay within a comfortable vocal range (7th-9th, about 10-14 semitones total). Don't jump around wildly.
8. DYNAMIC ARC: The overall melody should have a crescendo across the whole piece. Start simple, become more intense, climax around 70% through, then resolve.
9. VELOCITY: 45-65 for soft/narrative, 70-90 for normal, 95-115 for climax/accent. Accent the first beat of each bar.

Output format:
{
  "events": [
    {"pitch": 76, "start_beat": 0.0, "duration_beat": 0.5, "velocity": 95},
    {"pitch": 81, "start_beat": 0.375, "duration_beat": 0.25, "velocity": 88}
  ]
}

Constraints:
- 0 <= pitch <= 127
- 1 <= velocity <= 127
- start_beat >= 0, start_beat + duration_beat <= __BEATS__
- Output key must be exactly "events".
- Do NOT make every note the same duration. Vary them.
- Do NOT fill every moment. Leave space.`
	prompt = strings.ReplaceAll(prompt, "__INSTRUMENT__", instrument)
	prompt = strings.ReplaceAll(prompt, "__KEY__", key)
	prompt = strings.ReplaceAll(prompt, "__SCALE__", scale)
	prompt = strings.ReplaceAll(prompt, "__BPM__", fmt.Sprintf("%d", bpm))
	prompt = strings.ReplaceAll(prompt, "__BEATS__", fmt.Sprintf("%d", totalBeats))
	prompt = strings.ReplaceAll(prompt, "__CHORDS__", chordProg)
	prompt = strings.ReplaceAll(prompt, "__STYLE__", styleDesc)
	prompt = strings.ReplaceAll(prompt, "__FEATURES__", featureVec)
	prompt = strings.ReplaceAll(prompt, "__SEED__", randomSeed)
	return prompt
}

// BuildBassFromMelodyPrompt generates a bass line that follows a lead melody.
// The bass locks with the melody's strong beats and chord roots while
// providing a simpler, groove-focused counterpoint.
func BuildBassFromMelodyPrompt(key, scale, chordProg, styleDesc, featureVec, leadSummary string, bpm, totalBeats int, randomSeed string) string {
	return fmt.Sprintf(`You are a bassist composing a line that follows a lead melody.
Return JSON with an "events" array. No markdown.

Key: %s  Scale: %s  BPM: %d  Duration: %d beats
Chord progression: %s
Style: %s
Feature vector: %s
Lead melody summary: %s
Random seed: %s

BASS RULES (critical for a good rhythm section):
1. The bass is the BRIDGE between drums and melody. Lock with the kick drum (beat 1 and 3).
2. Play the CHORD ROOT on beat 1 of each bar. This anchors the harmony.
3. On beat 3, you may play the fifth (7 semitones up) or a passing tone to the next bar's root.
4. DO NOT copy the lead melody's rhythm. Bass is simpler: 1-4 notes per bar, mostly quarter notes.
5. Use octave jumps (root then root+12) for energy. Use chromatic approach (root-1 then root) for tension.
6. Create a GROOVE: a repeating pattern that feels good and doesn't get in the way of the melody.
7. Dynamics: 70-90 for normal, 90-110 when the melody hits its climax.
8. Use REST between phrases. Do not play every beat.

Output:
{
  "events": [
    {"pitch": 40, "start_beat": 0.0, "duration_beat": 1.0, "velocity": 85},
    {"pitch": 47, "start_beat": 2.0, "duration_beat": 1.0, "velocity": 80}
  ]
}

Constraints: pitch 0-127, velocity 1-127, start+duration <= %d`,
		key, scale, bpm, totalBeats, chordProg, styleDesc, featureVec, leadSummary, randomSeed, totalBeats)
}

// StripMarkdownFences removes ```json and ``` markers from LLM output.
func StripMarkdownFences(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = content[7:]
	} else if strings.HasPrefix(content, "```") {
		content = content[3:]
	}
	if strings.HasSuffix(content, "```") {
		content = content[:len(content)-3]
	}
	return strings.TrimSpace(content)
}

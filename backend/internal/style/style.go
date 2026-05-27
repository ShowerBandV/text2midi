// Package style provides a comprehensive database of music style descriptions.
// Each entry captures rhythm, instrumentation, harmony, mood, and BPM range.
package style

import "github.com/ShowerBandV/text2midi/internal/schema"

// Info describes a music style.
type Info struct {
	Name           string               `json:"name"`
	Description    string               `json:"description"`     // full prose description for LLM prompts
	DefaultVector  schema.FeatureVector `json:"default_vector"` // default feature vector for this style
}

// All returns every style in the database.
func All() map[string]Info {
	return descs
}

// Get returns the description for a style, or empty if unknown.
func Get(name string) string {
	if s, ok := descs[name]; ok {
		return s.Description
	}
	return ""
}

// GetDefaultVector returns the default feature vector for a style.
// Returns a neutral mid-range vector if the style is unknown.
func GetDefaultVector(name string) schema.FeatureVector {
	if s, ok := descs[name]; ok {
		return s.DefaultVector
	}
	return schema.FeatureVector{
		Darkness:           0.5,
		Energy:             0.5,
		Acousticness:       0.5,
		Density:            0.5,
		RhythmicComplexity: 0.5,
		Tension:            0.5,
		LoFi:               0.5,
	}
}

// descs is the master style database.
var descs = map[string]Info{
	// ─── Electronic ──────────────────────────────────────────────────
	"house": {
		Name: "House",
		Description: "Four-on-the-floor kick drum (kick on every beat), snare/clap on beats 2 and 4, " +
			"open hi-hat on the offbeat of beats 2 and 4, fat synth bass walking root motion with octave jumps, " +
			"sampled vocal chops, piano chords or synth arpeggios with high repetition, " +
			"simple major/minor triads in Dorian or Phrygian mode, BPM 120-130, " +
			"uplifting, groovy, danceable.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.7, Acousticness: 0.1, Density: 0.6, RhythmicComplexity: 0.3, Tension: 0.2, LoFi: 0.1},
	},
	"deep_house": {
		Name: "Deep House",
		Description: "Same four-on-the-floor as House but with softer kick, " +
			"deep walking upright or synth bass with melodic lines, jazzy chord progressions (II-V-I), " +
			"warm keyboard pads using extended chords (m7, M9, m11), BPM 115-125, " +
			"deep, soulful, late-night urban atmosphere.",
		DefaultVector: schema.FeatureVector{Darkness: 0.4, Energy: 0.5, Acousticness: 0.2, Density: 0.5, RhythmicComplexity: 0.4, Tension: 0.3, LoFi: 0.2},
	},
	"tech_house": {
		Name: "Tech House",
		Description: "Tight punchy kick and snare with rich percussion layers, " +
			"short bouncing synth bass with strong rhythm, " +
			"minimal melodies using vocal chops and sound effects, BPM 125-130, " +
			"techno-infused, industrial, underground club vibe.",
		DefaultVector: schema.FeatureVector{Darkness: 0.5, Energy: 0.7, Acousticness: 0.05, Density: 0.7, RhythmicComplexity: 0.5, Tension: 0.4, LoFi: 0.1},
	},
	"techno": {
		Name: "Techno",
		Description: "Hypnotic repetitive 4/4 kick, sparse snare or none, " +
			"mechanical 16th-note hi-hats often distorted, deep sub-bass drones or simple motifs, " +
			"minimal melodies using synth textures and noise, BPM 130-150, " +
			"hypnotic, industrial, futuristic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.7, Energy: 0.8, Acousticness: 0.05, Density: 0.8, RhythmicComplexity: 0.5, Tension: 0.5, LoFi: 0.1},
	},
	"trance": {
		Name: "Trance",
		Description: "Four-on-the-floor kick, snare on 2 and 4, drum rolls for transitions, " +
			"driving bassline locked to the kick, signature supersaw synth leads with long emotional melodies, " +
			"natural minor and harmonic minor scales, chord progressions with clear build-up and release, BPM 130-140, " +
			"euphoric, hypnotic, epic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.8, Acousticness: 0.1, Density: 0.7, RhythmicComplexity: 0.3, Tension: 0.3, LoFi: 0.05},
	},
	"dubstep": {
		Name: "Dubstep",
		Description: "Half-time feel with kick on 1 and 3, snare/clap on beat 3 (perceived 2 and 4), " +
			"heavy distorted wobble bass with extensive LFO modulation, " +
			"sparse melodies using high-register synth arpeggios or vocal samples, sudden drops, BPM 138-142 (felt at 70), " +
			"dark, aggressive, futuristic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.8, Energy: 0.9, Acousticness: 0.1, Density: 0.8, RhythmicComplexity: 0.7, Tension: 0.7, LoFi: 0.2},
	},
	"drum_and_bass": {
		Name: "Drum and Bass",
		Description: "Fast breakbeat drums (Amen Break style), rapid kick and snare alternation, " +
			"deep sub-bass drones or rolling basslines, " +
			"sampled jazz/soul/film-score elements, strings and piano, BPM 160-180, " +
			"intense, energetic, urban.",
		DefaultVector: schema.FeatureVector{Darkness: 0.5, Energy: 0.9, Acousticness: 0.1, Density: 0.9, RhythmicComplexity: 0.8, Tension: 0.4, LoFi: 0.1},
	},
	"uk_garage": {
		Name: "UK Garage",
		Description: "Swinging 2-step rhythm, kick on beat 1, snare on beat 3, hi-hats and shakers filling the gaps, " +
			"deep sub-bass focused on groove rather than melody, " +
			"sampled R&B vocals, warm keyboard chords, BPM 130-135, " +
			"British, underground, soulful.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.6, Acousticness: 0.2, Density: 0.5, RhythmicComplexity: 0.6, Tension: 0.2, LoFi: 0.2},
	},
	"future_bass": {
		Name: "Future Bass",
		Description: "Trap-style kick and snare but softer, " +
			"rounded synth bass with glide, bright synth chords with extended harmonies (maj7, add9), " +
			"leaping joyful melodies centered on major keys, BPM 140-160, " +
			"uplifting, colorful, anime-inspired.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.7, Acousticness: 0.1, Density: 0.6, RhythmicComplexity: 0.4, Tension: 0.2, LoFi: 0.1},
	},
	"synthwave": {
		Name: "Synthwave",
		Description: "Mechanical 4/4 kick, snare with gated reverb, " +
			"fat analog synth bass walking root motion, " +
			"80s-style synth leads with heavy pitch bends and glides, " +
			"simple major/minor triads in Dorian mode, BPM 80-110, " +
			"retro, neon-drenched, cinematic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.4, Energy: 0.6, Acousticness: 0.15, Density: 0.6, RhythmicComplexity: 0.3, Tension: 0.3, LoFi: 0.3},
	},

	// ─── Hip-Hop ────────────────────────────────────────────────────
	"trap": {
		Name: "Trap",
		Description: "Dense hi-hat rolls (32nd or 16th triplet), sparse syncopated kick, " +
			"short snappy snare on beat 3, sliding 808 bass with glide, " +
			"dark minimal melodies using minor triads or diminished scales, BPM 130-160, " +
			"dark, aggressive, street-oriented.",
		DefaultVector: schema.FeatureVector{Darkness: 0.8, Energy: 0.7, Acousticness: 0.1, Density: 0.6, RhythmicComplexity: 0.8, Tension: 0.6, LoFi: 0.3},
	},
	"boom_bap": {
		Name: "Boom Bap",
		Description: "Hard punchy kick and snare, snare on 2 and 4, " +
			"swinging 8th-note hi-hats, deep synth or electric bass playing roots and fifths, " +
			"sampled jazz/soul elements, warm and gritty texture, BPM 80-100, " +
			"classic, hardcore, East Coast hip-hop.",
		DefaultVector: schema.FeatureVector{Darkness: 0.4, Energy: 0.7, Acousticness: 0.4, Density: 0.6, RhythmicComplexity: 0.5, Tension: 0.3, LoFi: 0.4},
	},
	"drill": {
		Name: "Drill",
		Description: "Heavily syncopated snare with quintuplet rolls, sparse kick, " +
			"sliding 808 bass with exaggerated slides, " +
			"minimal dark piano or string melodies, BPM 130-145, " +
			"sinister, tense, dystopian.",
		DefaultVector: schema.FeatureVector{Darkness: 0.85, Energy: 0.8, Acousticness: 0.1, Density: 0.6, RhythmicComplexity: 0.9, Tension: 0.7, LoFi: 0.3},
	},
	"lofi": {
		Name: "Lo-fi Hip-Hop",
		Description: "Relaxed unquantized drum groove, snare intentionally laid back, " +
			"warm upright bass, jazz-influenced chord progressions with vinyl crackle, " +
			"nostalgic, warm textures, BPM 70-90, " +
			"nostalgic, relaxing, studying.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.2, Acousticness: 0.6, Density: 0.2, RhythmicComplexity: 0.2, Tension: 0.1, LoFi: 0.8},
	},
	"west_coast": {
		Name: "West Coast (G-Funk)",
		Description: "Laid-back kick and snare with reverb on snare, " +
			"fat synth bass with funk-influenced movement, " +
			"soaring synth leads with heavy pitch bending and talkbox, BPM 90-105, " +
			"psychedelic, sunny, funky.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.6, Acousticness: 0.3, Density: 0.5, RhythmicComplexity: 0.4, Tension: 0.2, LoFi: 0.2},
	},
	"jersey_club": {
		Name: "Jersey Club",
		Description: "Signature triplet kick pattern, heavy 808 bass, " +
			"chopped vocal samples looped and repeated, BPM 130-140, " +
			"bouncy, party, high-energy.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.9, Acousticness: 0.1, Density: 0.7, RhythmicComplexity: 0.6, Tension: 0.2, LoFi: 0.1},
	},
	"grime": {
		Name: "Grime",
		Description: "Fast 140 BPM rhythm with alternating kick and snare, " +
			"square-wave synth bass with aggressive tone, " +
			"minimal melodies using 8-bit sound effects, BPM 140, " +
			"British, underground, confrontational.",
		DefaultVector: schema.FeatureVector{Darkness: 0.7, Energy: 0.8, Acousticness: 0.1, Density: 0.7, RhythmicComplexity: 0.6, Tension: 0.6, LoFi: 0.2},
	},
	"reggaeton": {
		Name: "Reggaeton",
		Description: "Signature dembow rhythm (3+3+2 kick pattern), snare on 2 and 4, " +
			"deep sub-bass simple repetition, Latin piano or synth melodies, BPM 90-100, " +
			"Latin, fiery, dancefloor.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.8, Acousticness: 0.3, Density: 0.6, RhythmicComplexity: 0.5, Tension: 0.1, LoFi: 0.1},
	},

	// ─── Rock/Metal ─────────────────────────────────────────────────
	"classic_rock": {
		Name: "Classic Rock",
		Description: "Straight 4/4 with evenly spaced kick and snare, snare heavy on 2 and 4, " +
			"bass following the kick root motion, distorted power-chord guitar, " +
			"blues-scale guitar solos, BPM 100-140, " +
			"classic, powerful, free-spirited.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.7, Acousticness: 0.7, Density: 0.6, RhythmicComplexity: 0.3, Tension: 0.3, LoFi: 0.3},
	},
	"blues": {
		Name: "Blues",
		Description: "12-bar blues structure, swinging 8th-note feel, " +
			"walking bass line, blues-scale guitar solos with bends and slides, " +
			"I-IV-V progression with dominant 7th chords, BPM 60-120, " +
			"soulful, melancholic, narrative.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.4, Acousticness: 0.9, Density: 0.4, RhythmicComplexity: 0.4, Tension: 0.3, LoFi: 0.4},
	},
	"metal": {
		Name: "Metal",
		Description: "Double-kick drum rapid alternation, snare on 2 and 4 with blast beats, " +
			"down-tuned distorted guitar power chords and fast riffs, bass locked to guitar, " +
			"Phrygian or harmonic minor scales, BPM 120-200, " +
			"aggressive, dark, powerful.",
		DefaultVector: schema.FeatureVector{Darkness: 0.9, Energy: 0.95, Acousticness: 0.6, Density: 0.9, RhythmicComplexity: 0.5, Tension: 0.8, LoFi: 0.2},
	},
	"punk": {
		Name: "Punk",
		Description: "Fast 4/4 with alternating kick and snare, " +
			"three-chord power chords with distortion, bass following guitar roots, " +
			"simple direct highly repetitive melodies, BPM 160-200, " +
			"rebellious, energetic, direct.",
		DefaultVector: schema.FeatureVector{Darkness: 0.4, Energy: 0.95, Acousticness: 0.6, Density: 0.9, RhythmicComplexity: 0.3, Tension: 0.4, LoFi: 0.3},
	},
	"indie_rock": {
		Name: "Indie Rock",
		Description: "Natural unquantized drum groove, " +
			"clean or overdriven guitar using open chords and arpeggios, " +
			"melodic bass with motion beyond root notes, " +
			"indie-pop melodic lines, BPM 100-130, " +
			"independent, youthful, introspective.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.5, Acousticness: 0.7, Density: 0.4, RhythmicComplexity: 0.3, Tension: 0.2, LoFi: 0.3},
	},

	// ─── Pop ─────────────────────────────────────────────────────────
	"kpop": {
		Name: "K-Pop",
		Description: "Fusion of Trap, House, and Funk drum elements, " +
			"grooving synth bass, catchy hooks with layered vocals, " +
			"rich colorful chords with frequent key changes, BPM 100-140, " +
			"energetic, polished, dynamic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.15, Energy: 0.8, Acousticness: 0.3, Density: 0.7, RhythmicComplexity: 0.5, Tension: 0.15, LoFi: 0.05},
	},
	"synth_pop": {
		Name: "Synth-pop",
		Description: "Mechanical 4/4 kick, snare on 2 and 4, " +
			"simple repeating synth bass, catchy synth lead melodies, BPM 100-130, " +
			"electronic, nostalgic, pop-oriented.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.6, Acousticness: 0.2, Density: 0.5, RhythmicComplexity: 0.3, Tension: 0.2, LoFi: 0.1},
	},
	"rnb": {
		Name: "R&B",
		Description: "Slow Trap-style drum groove or 4/4, " +
			"melodic 808 or synth bass, smooth vocal-style melodies with runs, " +
			"seventh, ninth, and suspended chords, BPM 60-90, " +
			"romantic, soulful, sensual.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.4, Acousticness: 0.4, Density: 0.4, RhythmicComplexity: 0.4, Tension: 0.3, LoFi: 0.2},
	},

	// ─── Jazz/Funk/Soul ──────────────────────────────────────────────
	"funk": {
		Name: "Funk",
		Description: "Strong emphasis on beat 1 (The One), heavy kick and snare syncopation, " +
			"clean funk guitar with 16th-note muting, slapping bass with extreme groove, " +
			"brass section melodies (trumpet, saxophone), BPM 90-120, " +
			"groovy, party, soulful.",
		DefaultVector: schema.FeatureVector{Darkness: 0.1, Energy: 0.7, Acousticness: 0.7, Density: 0.6, RhythmicComplexity: 0.7, Tension: 0.2, LoFi: 0.2},
	},
	"jazz": {
		Name: "Jazz",
		Description: "Swing rhythm with ride cymbal keeping continuous swing pattern, " +
			"walking bass, complex piano voicings with improvisation, " +
			"II-V-I progressions, extended and altered chords, BPM 60-300, " +
			"improvisational, elegant, complex.",
		DefaultVector: schema.FeatureVector{Darkness: 0.1, Energy: 0.3, Acousticness: 0.95, Density: 0.4, RhythmicComplexity: 0.7, Tension: 0.4, LoFi: 0.2},
	},
	"soul": {
		Name: "Soul",
		Description: "Warm powerful 4/4 drum groove, " +
			"melodic electric bass, emotionally charged vocal-style melodies, " +
			"rich seventh and ninth chords, BPM 70-100, " +
			"emotional, warm, human.",
		DefaultVector: schema.FeatureVector{Darkness: 0.15, Energy: 0.5, Acousticness: 0.7, Density: 0.4, RhythmicComplexity: 0.3, Tension: 0.2, LoFi: 0.15},
	},
	"bossa_nova": {
		Name: "Bossa Nova",
		Description: "Signature bossa nova drum pattern with steady ride cymbal, " +
			"root-fifth bass alternation, nylon-string guitar with complex voicings, " +
			"rich extended jazz chords, BPM 100-140, " +
			"Brazilian, relaxed, elegant.",
		DefaultVector: schema.FeatureVector{Darkness: 0.05, Energy: 0.3, Acousticness: 0.9, Density: 0.3, RhythmicComplexity: 0.5, Tension: 0.15, LoFi: 0.1},
	},

	// ─── Latin/World ─────────────────────────────────────────────────
	"salsa": {
		Name: "Salsa",
		Description: "Clave rhythm (3-2 or 2-3 pattern), congas and timbales, " +
			"piano montuno repetitive pattern, tumbao bass rhythm, " +
			"brass section melodies, BPM 150-200, " +
			"fiery, danceable, Cuban.",
		DefaultVector: schema.FeatureVector{Darkness: 0.05, Energy: 0.9, Acousticness: 0.7, Density: 0.7, RhythmicComplexity: 0.8, Tension: 0.1, LoFi: 0.05},
	},
	"reggae": {
		Name: "Reggae",
		Description: "Strong offbeat emphasis on 2 and 4, guitar or keyboard skank on offbeats, " +
			"deep melodic bass line, simple repeating vocal melodies, BPM 60-80, " +
			"relaxed, Caribbean, peaceful.",
		DefaultVector: schema.FeatureVector{Darkness: 0.1, Energy: 0.3, Acousticness: 0.6, Density: 0.4, RhythmicComplexity: 0.4, Tension: 0.1, LoFi: 0.3},
	},
	"dancehall": {
		Name: "Dancehall",
		Description: "Reggae offbeat with electronic drums, " +
			"deep sub-bass, repeating vocal hooks, BPM 90-100, " +
			"Jamaican, dancefloor, energetic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.2, Energy: 0.7, Acousticness: 0.3, Density: 0.5, RhythmicComplexity: 0.5, Tension: 0.1, LoFi: 0.2},
	},
	"afrobeats": {
		Name: "Afrobeats",
		Description: "Complex African drum polyrhythms, syncopated kick and snare, " +
			"melodic synth bass, repeating vocal hooks with warm keyboards, BPM 100-115, " +
			"African, sunny, groovy.",
		DefaultVector: schema.FeatureVector{Darkness: 0.1, Energy: 0.8, Acousticness: 0.5, Density: 0.6, RhythmicComplexity: 0.6, Tension: 0.1, LoFi: 0.15},
	},

	// ─── Ambient/Chill ───────────────────────────────────────────────
	"ambient": {
		Name: "Ambient",
		Description: "No drums or extremely sparse, " +
			"long sustained tones and soundscapes, " +
			"simple slow-changing major/minor harmonies, no fixed tempo, " +
			"spacious, meditative, atmospheric.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.05, Acousticness: 0.8, Density: 0.1, RhythmicComplexity: 0.05, Tension: 0.1, LoFi: 0.4},
	},
	"chillwave": {
		Name: "Chillwave",
		Description: "Lo-fi drum machine beats, " +
			"warm synth bass, dreamy synthesizers with heavy reverb, BPM 80-100, " +
			"nostalgic, dreamy, lo-fi.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.3, Acousticness: 0.4, Density: 0.3, RhythmicComplexity: 0.2, Tension: 0.15, LoFi: 0.6},
	},
	"vaporwave": {
		Name: "Vaporwave",
		Description: "Slow 4/4 beat, " +
			"sampled 80s-90s pop music heavily pitch-shifted, BPM 60-90, " +
			"satirical, consumerist, retro-futuristic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.3, Energy: 0.2, Acousticness: 0.3, Density: 0.3, RhythmicComplexity: 0.2, Tension: 0.2, LoFi: 0.7},
	},

	// ─── Cinematic/Game ──────────────────────────────────────────────
	"epic_orchestral": {
		Name: "Epic Orchestral",
		Description: "Grand percussion (taiko, timpani), rapid string staccatos, " +
			"heroic horn and string themes, " +
			"major and minor harmony with frequent modulations, BPM 100-140, " +
			"epic, warlike, heroic.",
		DefaultVector: schema.FeatureVector{Darkness: 0.4, Energy: 0.8, Acousticness: 0.9, Density: 0.7, RhythmicComplexity: 0.3, Tension: 0.3, LoFi: 0.0},
	},
	"chiptune": {
		Name: "8-bit / Chiptune",
		Description: "Simple 4/4 with electronic drum sounds, " +
			"square-wave and triangle-wave synth leads with fast arpeggios, " +
			"simple major/minor triads, BPM 120-180, " +
			"video-game, nostalgic, pixelated.",
		DefaultVector: schema.FeatureVector{Darkness: 0.1, Energy: 0.7, Acousticness: 0.0, Density: 0.5, RhythmicComplexity: 0.3, Tension: 0.1, LoFi: 0.2},
	},
	"horror_ambient": {
		Name: "Horror / Dark Ambient",
		Description: "Irregular or no rhythm, " +
			"dissonant intervals and tone clusters, " +
			"diminished 7th chords and whole-tone scales, very slow tempo, " +
			"fearful, suspenseful, dark.",
		DefaultVector: schema.FeatureVector{Darkness: 0.95, Energy: 0.1, Acousticness: 0.4, Density: 0.2, RhythmicComplexity: 0.1, Tension: 0.9, LoFi: 0.3},
	},
}

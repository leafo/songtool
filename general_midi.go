package main

import "fmt"

// General MIDI Drum/Percussion Key Map
// Reference: https://computermusicresource.com/GM.Percussion.KeyMap.html
const (
	AcousticBassDrum = 35 // B0 - Acoustic Bass Drum
	BassDrum1        = 36 // C1 - Bass Drum 1
	SideStick        = 37 // C#1 - Side Stick
	AcousticSnare    = 38 // D1 - Acoustic Snare
	HandClap         = 39 // Eb1 - Hand Clap
	ElectricSnare    = 40 // E1 - Electric Snare
	LowFloorTom      = 41 // F1 - Low Floor Tom
	ClosedHiHat      = 42 // F#1 - Closed Hi Hat
	HighFloorTom     = 43 // G1 - High Floor Tom
	PedalHiHat       = 44 // Ab1 - Pedal Hi-Hat
	LowTom           = 45 // A1 - Low Tom
	OpenHiHat        = 46 // Bb1 - Open Hi-Hat
	LowMidTom        = 47 // B1 - Low-Mid Tom
	HiMidTom         = 48 // C2 - Hi Mid Tom
	CrashCymbal1     = 49 // C#2 - Crash Cymbal 1
	HighTom          = 50 // D2 - High Tom
	RideCymbal1      = 51 // Eb2 - Ride Cymbal 1
	ChineseCymbal    = 52 // E2 - Chinese Cymbal
	RideBell         = 53 // F2 - Ride Bell
	Tambourine       = 54 // F#2 - Tambourine
	SplashCymbal     = 55 // G2 - Splash Cymbal
	Cowbell          = 56 // Ab2 - Cowbell
	CrashCymbal2     = 57 // A2 - Crash Cymbal 2
	Vibraslap        = 58 // Bb2 - Vibraslap
	RideCymbal2      = 59 // B2 - Ride Cymbal 2
	HiBongo          = 60 // C3 - Hi Bongo
	LowBongo         = 61 // C#3 - Low Bongo
	MuteHiConga      = 62 // D3 - Mute Hi Conga
	OpenHiConga      = 63 // Eb3 - Open Hi Conga
	LowConga         = 64 // E3 - Low Conga
	HighTimbale      = 65 // F3 - High Timbale
	LowTimbale       = 66 // F#3 - Low Timbale
	HighAgogo        = 67 // G3 - High Agogo
	LowAgogo         = 68 // Ab3 - Low Agogo
	Cabasa           = 69 // A3 - Cabasa
	Maracas          = 70 // Bb3 - Maracas
	ShortWhistle     = 71 // B3 - Short Whistle
	LongWhistle      = 72 // C4 - Long Whistle
	ShortGuiro       = 73 // C#4 - Short Guiro
	LongGuiro        = 74 // D4 - Long Guiro
	Claves           = 75 // Eb4 - Claves
	HiWoodBlock      = 76 // E4 - Hi Wood Block
	LowWoodBlock     = 77 // F4 - Low Wood Block
	MuteCuica        = 78 // F#4 - Mute Cuica
	OpenCuica        = 79 // G4 - Open Cuica
	MuteTriangle     = 80 // Ab4 - Mute Triangle
	OpenTriangle     = 81 // A4 - Open Triangle
)

// https://en.wikipedia.org/wiki/General_MIDI#Program_change_events
func getGMInstrument(program uint8) string {
	instruments := []string{
		"Acoustic Grand Piano", "Bright Acoustic Piano", "Electric Grand Piano", "Honky-tonk Piano",
		"Electric Piano 1", "Electric Piano 2", "Harpsichord", "Clavi",
		"Celesta", "Glockenspiel", "Music Box", "Vibraphone",
		"Marimba", "Xylophone", "Tubular Bells", "Dulcimer",
		"Drawbar Organ", "Percussive Organ", "Rock Organ", "Church Organ",
		"Reed Organ", "Accordion", "Harmonica", "Tango Accordion",
		"Acoustic Guitar (nylon)", "Acoustic Guitar (steel)", "Electric Guitar (jazz)", "Electric Guitar (clean)",
		"Electric Guitar (muted)", "Overdriven Guitar", "Distortion Guitar", "Guitar Harmonics",
		"Acoustic Bass", "Electric Bass (finger)", "Electric Bass (pick)", "Fretless Bass",
		"Slap Bass 1", "Slap Bass 2", "Synth Bass 1", "Synth Bass 2",
		"Violin", "Viola", "Cello", "Contrabass",
		"Tremolo Strings", "Pizzicato Strings", "Orchestral Harp", "Timpani",
		"String Ensemble 1", "String Ensemble 2", "Synth Strings 1", "Synth Strings 2",
		"Choir Aahs", "Voice Oohs", "Synth Voice", "Orchestra Hit",
		"Trumpet", "Trombone", "Tuba", "Muted Trumpet",
		"French Horn", "Brass Section", "Synth Brass 1", "Synth Brass 2",
		"Soprano Sax", "Alto Sax", "Tenor Sax", "Baritone Sax",
		"Oboe", "English Horn", "Bassoon", "Clarinet",
		"Piccolo", "Flute", "Recorder", "Pan Flute",
		"Blown Bottle", "Shakuhachi", "Whistle", "Ocarina",
		"Lead 1 (square)", "Lead 2 (sawtooth)", "Lead 3 (calliope)", "Lead 4 (chiff)",
		"Lead 5 (charang)", "Lead 6 (voice)", "Lead 7 (fifths)", "Lead 8 (bass + lead)",
		"Pad 1 (new age)", "Pad 2 (warm)", "Pad 3 (polysynth)", "Pad 4 (choir)",
		"Pad 5 (bowed)", "Pad 6 (metallic)", "Pad 7 (halo)", "Pad 8 (sweep)",
		"FX 1 (rain)", "FX 2 (soundtrack)", "FX 3 (crystal)", "FX 4 (atmosphere)",
		"FX 5 (brightness)", "FX 6 (goblins)", "FX 7 (echoes)", "FX 8 (sci-fi)",
		"Sitar", "Banjo", "Shamisen", "Koto",
		"Kalimba", "Bag pipe", "Fiddle", "Shanai",
		"Tinkle Bell", "Agogo", "Steel Drums", "Woodblock",
		"Taiko Drum", "Melodic Tom", "Synth Drum", "Reverse Cymbal",
		"Guitar Fret Noise", "Breath Noise", "Seashore", "Bird Tweet",
		"Telephone Ring", "Helicopter", "Applause", "Gunshot",
	}

	if int(program) < len(instruments) {
		return instruments[program]
	}
	return fmt.Sprintf("Unknown (%d)", program)
}

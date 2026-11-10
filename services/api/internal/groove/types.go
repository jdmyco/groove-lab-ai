package groove

type Tempo struct {
	BPM      int    `json:"bpm"`
	BeatUnit string `json:"beat_unit"`
}

type Request struct {
	Style string `json:"style"`
	Meter string `json:"meter"`
	Tempo Tempo  `json:"tempo"`
	Bars  int    `json:"bars"`
}

type Event struct {
	Tick       int    `json:"tick"`
	Instrument string `json:"instrument"`
	Velocity   int    `json:"velocity"`
}

type Score struct {
	SchemaVersion   string  `json:"schema_version"`
	Style           string  `json:"style"`
	Meter           string  `json:"meter"`
	Tempo           Tempo   `json:"tempo"`
	Bars            int     `json:"bars"`
	TicksPerQuarter int     `json:"ticks_per_quarter"`
	Events          []Event `json:"events"`
}

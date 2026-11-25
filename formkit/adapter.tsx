import { createRoot } from 'react-dom/client';
import { GrooveScore } from '@formkit/components/grooves/GrooveScore.tsx';
import {
  defaultSettings, GROOVE_PPQ, songSeconds,
} from '@formkit/music/groove-schema.ts';
import type { DrumEvent, DrumVoice, Song } from '@formkit/music/groove-schema.ts';
import { DRUM_POSITION } from '@formkit/music/groove-notation.ts';
import { GrooveEngine } from '@formkit/music/groove-engine.ts';

const voices = {
  kick: 'kick', snare: 'snare', rim: 'snare.rim',
  closed_hat: 'hihat.closed', open_hat: 'hihat.open', foot_hat: 'hihat.foot',
  ride: 'ride', crash: 'crash', cowbell: 'cowbell', clave: 'clave', shaker: 'shaker',
  tom_high: 'tom.high', tom_mid: 'tom.mid', tom_low: 'tom.low',
} as const;

// FormKit treats these auxiliary percussion sounds as separate parts. Give them
// distinct staff positions when a Groove Lab score puts them on the kit staff.
Object.assign(DRUM_POSITION, { shaker: -1.5, clave: -2.5, 'tom.mid': 1 });

type Groove = {
  style: string;
  meter: string;
  grouping?: number[];
  bars: number;
  ticks_per_quarter: number;
  tempo: { bpm: number; beat_unit: string };
  events: {
    tick: number; instrument: keyof typeof voices; velocity: number; offset?: number;
  }[];
};

type PreviewSong = Song & { drumBeamGroups?: number[] };

function toFormKitSong(score: Groove, title: string): PreviewSong {
  const meter = score.meter.split('/').map(Number) as [number, number];
  const scale = GROOVE_PPQ / score.ticks_per_quarter;
  const subdivision = score.events.every(event => event.tick % 32 === 0)
    && score.events.some(event => event.tick % 24 !== 0) ? 'triplet' : 'straight16';
  const beatUnit = score.tempo.beat_unit === 'dotted_quarter' ? 'dottedQuarter' : 'quarter';
  const unit = score.ticks_per_quarter * 4 / meter[1];
  const scoreBar = meter[0] * unit;
  const grouping = score.grouping ?? (meter[1] === 8
    ? meter[0] === 7 ? [2, 2, 3] : Array(meter[0] / 3).fill(3)
    : []);
  let start = 0;
  const groupStarts = new Set(grouping.map(group => {
    const at = start;
    start += group * unit;
    return at;
  }));
  const drums = score.events.map(event => ({
    tick: event.tick * scale,
    duration: Math.min(240, (scoreBar - event.tick % scoreBar) * scale),
    voice: voices[event.instrument] as DrumVoice,
    velocity: event.velocity / 127,
    ...(event.offset ? { offset: event.offset * scale } : {}),
    accent: event.velocity >= 110 || (
      groupStarts.has(event.tick % scoreBar)
      && ['closed_hat', 'open_hat', 'ride', 'cowbell', 'shaker', 'clave'].includes(event.instrument)
    ),
    ghost: event.instrument === 'snare' && event.velocity <= 55,
  }));
  const song: PreviewSong = {
    schemaVersion: 1, id: 'groove-lab-preview', version: 1, title,
    genreBucket: score.style, filter: score.style, idiom: 'Drums', feel: '',
    key: { tonic: 'C', mode: 'major', signature: 0 },
    meter, beatUnit, subdivision,
    ...(meter[1] === 8 ? { drumBeamGroups: grouping.map(group => group * unit * scale) } : {}),
    defaultBpm: score.tempo.bpm, tempoRange: [40, 220],
    bars: score.bars, form: [{name:'Groove', startBar:1, endBar:score.bars, role:'head'}],
    loopBody: {startBar:1, endBar:score.bars},
    parts: [{
      id:'drums', role:'drums', instrument:'drum-kit', displayName:'Drums',
      abbr:'DR', defaultLevel:60, notation:'drumStaff', drums,
    }],
    chords: [], soloDefaults: [], click: {defaultLevel:60}, lead: 'drums', backing: '',
    provenance: {
      rights: '', referenceSeed: '', titleProvenance: 'Groove Lab display title',
      referenceSources: [], documentedVsInterpreted: {}, melodyProvenance: '',
      closenessCheck: '', notationLimits: '', difficulty: {level:'medium', reason:'Preview default'},
      audition: {device:'', date:'', result:'', reviewer:''}, ownerContentReview:'pending', fixture:true,
    },
    lengthSeconds: 0, lengthMath: '',
  };
  return { ...song, lengthSeconds: songSeconds(song) };
}

function audioDrum(event: DrumEvent): DrumEvent {
  // Sort by sounding onset in the worklet's compiled audio score. Applying
  // offsets here keeps two notes that cross in time in their audible order.
  const { offset = 0, ...written } = event;
  return {
    ...written,
    tick: written.tick + offset,
    // The mid tom is written separately on the staff and uses the low sample.
    voice: (written.voice as string) === 'tom.mid' ? 'tom.low' : written.voice,
  };
}

export function createPreview(
  container: HTMLElement,
  onState: (view: {state:string; starting:boolean; error:string}) => void,
) {
  let song: Song | undefined;
  let settings: ReturnType<typeof defaultSettings> | undefined;
  const root = createRoot(container);
  // This URL is served next to the entry bundle, using FormKit's exact processor.
  const engine = new GrooveEngine('/assets/formkit/groove-worklet.js', render);
  let options = {
    barsPerSystem: 1, S: 7.5, bassMode:'lane' as const, staves:'drums' as const, view:'score' as const,
  };
  const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const noop = () => {};
  let lastState = '';

  function render() {
    if (!song || !settings) return;
    const view = engine.view();
    const width = container.clientWidth;
    const barsPerSystem = width >= 850 ? 4 : width >= 600 ? 2 : 1;
    if (options.barsPerSystem !== barsPerSystem) options = { ...options, barsPerSystem };
    root.render(<GrooveScore {...{
      song, settings, tick: view.tick, playhead: view.state === 'playing' || view.state === 'paused',
      paused: view.state === 'paused', follow: false, options, canSeek: false,
      onSeekBar: noop, onManualScroll: noop, reducedMotion,
    }} />);
    const state = {state:view.state, starting:engine.starting, error:engine.error};
    const key = JSON.stringify(state);
    if (key !== lastState) {
      lastState = key;
      onState(state);
    }
  }

  // The score component measures its own width. This updates only the layout
  // breakpoint during resize and keeps memoized geometry during playback.
  new ResizeObserver(render).observe(container);

  return {
    setScore(score: Groove, title: string) {
      if (song) engine.stop();
      song = toFormKitSong(score, title);
      settings = defaultSettings(song);
      settings.countIn = false;
      settings.levels.click.muted = true;
      engine.configure({
        ...song,
        parts: song.parts.map(part => ({...part, drums: part.drums?.map(audioDrum)})),
      }, settings);
      render();
    },
    play: () => engine.play(),
    pause: () => engine.pause(),
    stop: () => engine.stop(),
  };
}

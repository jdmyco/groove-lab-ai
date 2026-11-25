import { engraveDrums } from '@formkit/music/groove-notation.ts';
import type { DrumColumn } from '@formkit/music/groove-notation.ts';
import type { DrumEvent } from '@formkit/music/groove-schema.ts';

type Onset = DrumEvent & { at: number };

/** Use FormKit's engraving within each written beam group. */
export function engraveLoopDrums(
  events: Onset[], beatTicks: number, barTicks: number, groups?: number[],
) {
  const spans = groups ?? Array.from({length: Math.ceil(barTicks / beatTicks)}, (_, index) =>
    Math.min(beatTicks, barTicks - index * beatTicks));
  const beats: DrumColumn[][] = [];
  const feet: Onset[] = [];
  const starts = new Map<number, number>();
  let start = 0;
  for (const span of spans) {
    const end = start + span;
    const within = events.filter(event => event.at >= start && event.at < end)
      .map(event => ({ ...event, at: event.at - start }));
    const group = engraveDrums(within, span);
    for (const foot of group.feet) feet.push({ ...foot, at: foot.at + start });
    for (const columns of group.beats) {
      for (const column of columns) starts.set(column.at + start, start);
      beats.push(columns.map(column => ({ ...column, at: column.at + start })));
    }
    start = end;
  }
  const beatStartAt = (at: number) => starts.get(at) ?? at;
  const unflagged = (column: DrumColumn) => column.at === beatStartAt(column.at) && column.display >= 960;
  return { beats, feet, beatStartAt, unflagged };
}

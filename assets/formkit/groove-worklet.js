import { t as e } from "./groove-clock.js";
var t = class extends AudioWorkletProcessor {
	clock = new e(sampleRate);
	frames = 0;
	waiting = !1;
	constructor() {
		super(), this.port.onmessage = (e) => {
			if (e.data.type === "ack") this.waiting = !1;
			else try {
				this.clock.command(e.data), this.port.postMessage({ view: this.clock.view() });
			} catch (e) {
				this.port.postMessage({ error: String(e) });
			}
		};
	}
	process(e, t) {
		let n = t[0]?.[0];
		return n && (this.clock.render(n), this.frames += n.length), this.frames >= sampleRate / 30 && !this.waiting && (this.frames = 0, this.waiting = !0, this.port.postMessage({ view: this.clock.view() })), !0;
	}
};
registerProcessor("bopkit-groove", t);

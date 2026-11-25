var e = (e) => e.meter[0] * 960 * 4 / e.meter[1], t = (e) => e.beatUnit === "dottedQuarter" ? 1440 : 960, n = (n) => e(n) / t(n), r = (e, t = e.defaultBpm) => e.bars * n(e) * 60 / t, i = {
	C: 0,
	D: 2,
	E: 4,
	F: 5,
	G: 7,
	A: 9,
	B: 11
}, a = [
	"C",
	"D",
	"E",
	"F",
	"G",
	"A",
	"B"
], o = {
	"": 0,
	"#": 1,
	b: -1,
	"##": 2,
	bb: -2,
	n: 0
};
function s(e) {
	let t = /^([A-G])(bb|##|b|#|n)?(-?\d)$/.exec(e);
	if (!t) throw RangeError(`Invalid note name ${e}`);
	let n = (t[2] ?? "").replace("n", "");
	return {
		letter: t[1],
		accidental: n,
		octave: Number(t[3]),
		step: Number(t[3]) * 7 + a.indexOf(t[1])
	};
}
var c = (e, t = "") => ((i[e] + (o[t] ?? 0)) % 12 + 12) % 12, l = /^([A-G])(b|#)?(maj|min|m|dim|aug|sus|ø|°|\+)?([^/]*)(?:\/([A-G])(b|#)?)?$/;
function u(e) {
	let t = l.exec(e.trim());
	if (!t) throw RangeError(`Invalid chord symbol ${e}`);
	let [, n, r = "", i = "", a, o, s] = t, u = i === "min" ? "m" : i, d = a, f = 4, p = 7, m = null, h = [], g = (e) => {
		h.includes(e) || h.push(e);
	};
	u === "m" && (f = 3), (u === "dim" || u === "°") && (f = 3, p = 6), u === "ø" && (f = 3, p = 6, m = 10), (u === "aug" || u === "+") && (p = 8), u === "sus" && (f = null, d.startsWith("2") ? (g(2), d = d.slice(1), u = "sus2") : (g(5), d.startsWith("4") && (d = d.slice(1)), u = "sus4"));
	let _ = d, v = /^maj/.exec(_), y = u === "maj";
	v && (y = !0, _ = _.slice(3));
	let b = /^(6\/9|69|13|11|9|7|6|5)?/.exec(_)[1] ?? "";
	_ = _.slice(b.length), b === "5" && (f = null), b === "6" && g(9), (b === "6/9" || b === "69") && (g(9), g(2)), ([
		"7",
		"9",
		"11",
		"13"
	].includes(b) || y) && m === null && (m = u === "dim" || u === "°" ? 9 : y ? 11 : 10), (b === "9" || b === "11" || b === "13") && g(2), (b === "11" || b === "13" && f === 3) && g(5), b === "13" && g(9);
	let x = _.replace(/[()]/g, "").split(/[,\s]+/).join(""), S = /(add|sus)?(b|#)?(\d+)/g, C;
	for (; C = S.exec(x);) {
		let [, t, n = "", r] = C, i = Number(r), a = {
			2: 2,
			4: 5,
			5: 7,
			9: 2,
			11: 5,
			13: 9
		};
		if (!(i in a)) throw RangeError(`Invalid chord ${e}`);
		let o = (a[i] + (n === "#" ? 1 : n === "b" ? -1 : 0) + 12) % 12;
		if (i === 5) p = o;
		else if (t === "sus" && i === 4) f = null, g(5);
		else {
			if (i === 9 && n) {
				let e = h.indexOf(2);
				e >= 0 && h.splice(e, 1);
			}
			if (i === 11 && n === "#") {
				let e = h.indexOf(5);
				e >= 0 && h.splice(e, 1);
			}
			if (i === 13 && n === "b") {
				let e = h.indexOf(9);
				e >= 0 && h.splice(e, 1);
			}
			g(o);
		}
	}
	let w = c(n, r), T = [0];
	f !== null && T.push(f), p !== null && T.push(p), m !== null && T.push(m);
	for (let e of h) T.includes(e) || T.push(e);
	return {
		symbol: e,
		root: n,
		rootAccidental: r,
		rootPc: w,
		quality: u === "sus4" || u === "sus2" ? "sus" : u,
		extension: u === "sus2" ? "2" + d : u === "sus4" && e.includes("sus4") ? "4" + d : d,
		bass: o,
		bassAccidental: o ? s ?? "" : void 0,
		bassPc: o ? c(o, s ?? "") : w,
		intervals: T,
		third: f,
		seventh: m,
		fifth: p,
		tensions: h,
		dominant: f === 4 && m === 10,
		minor: f === 3
	};
}
function d(e, t) {
	let n = e.chords[0];
	for (let r of e.chords) if (r.tick <= t) n = r;
	else break;
	return n;
}
function f(e, t = 55, n = 4, r = "rootless") {
	let i = u(e), a = (e) => e.map((e) => t + (e - t + 1200) % 12).sort((e, t) => e - t);
	if (r === "quartal" && (i.minor || i.third === null) && i.fifth === 7) {
		let e = i.minor ? 2 : 7, r = [
			0,
			5,
			10,
			15
		].slice(0, Math.max(3, n - 1)).map((t) => e + t), a = t + (i.rootPc + e - t + 1200) % 12, o = r.map((t) => a + (t - e));
		return (n >= 5 || i.minor) && o.push(o[o.length - 1] + 4), o.slice(0, Math.max(n, 4));
	}
	if (r === "triad") return a([
		0,
		i.third ?? 5,
		i.fifth ?? 7
	].map((e) => (e + i.rootPc) % 12));
	let o = [], s = (e) => {
		e !== null && !o.includes(e) && o.push(e);
	};
	s(i.third ?? (i.tensions.includes(5) ? 5 : 2)), s(i.seventh);
	for (let e of i.tensions) s(e);
	i.fifth !== 7 && s(i.fifth), s(i.fifth), o.length < n && s(0);
	let c = r === "shell" ? 3 : r === "funk" ? Math.min(n, 3) : n, l = o.slice(0, c), d = a(l.map((e) => (e + i.rootPc) % 12));
	if (r === "funk") {
		let e = l.find((e) => e !== i.third && e !== i.seventh && e !== 0);
		if (e !== void 0) {
			let t = (e + i.rootPc) % 12, n = d.filter((e) => e % 12 !== t), r = d.find((e) => e % 12 === t);
			for (; r <= Math.max(...n);) r += 12;
			d = [...n, r];
		}
	}
	if (r === "guitar" && d.length >= 4) {
		let e = d[d.length - 2];
		d = [e - 12, ...d.filter((t) => t !== e)].sort((e, t) => e - t);
	}
	return d;
}
function p(e) {
	let t = {};
	for (let n of e.parts) t[n.id] = {
		value: 60,
		muted: !1
	};
	return t.click = {
		value: 60,
		muted: !1
	}, {
		songId: e.id,
		songVersion: e.version,
		tempo: e.defaultBpm,
		loop: !0,
		countIn: !0,
		levels: t,
		spans: structuredClone(e.soloDefaults),
		soloForms: 1
	};
}
var m = (e) => e.form.find((e) => e.role === "intro" && e.startBar === 1), h = (e) => (m(e)?.endBar ?? 0) + 1, g = (e) => e.form.find((e) => e.role === "solo");
function _(e) {
	let t = g(e), n = t ? (t.endBar - t.startBar + 1) / (t.repeats ?? 1) : 0;
	return e.formChorusBars ?? n;
}
var v = (e) => e.role === "drums" || e.role === "percussion";
function y(e, t, n) {
	let r = e.find((e) => n >= e.startBar && n <= e.endBar);
	return {
		behaviour: r?.behaviours[t] ?? "on",
		span: r
	};
}
function b(e, t, n, r) {
	let i = e.overrides?.find((e) => e.bar === n && e.partId === t);
	if (i) return [...i.pattern].sort((e, t) => e - t);
	let a = e.keyBeats[t];
	if (!a) return [];
	let o = (n - e.startBar) % a.patternBars;
	return a.pattern.filter((e) => e >= o * r && e < (o + 1) * r).map((e) => e - o * r).sort((e, t) => e - t);
}
function x(t, n, r, i, a) {
	let o = [], s = e(t);
	return i.forEach((e, c) => {
		let l = Math.min(i[c + 1] ?? a, a) - e, p = e % s === 0 ? .9 : .8;
		if (v(n)) {
			let t = n.hitVoice?.voices ?? (e % s === 0 ? ["kick", "crash"] : ["kick", "snare"]), i = Math.max(1, Math.min(480, l));
			for (let n of t) o.push({
				tick: e,
				duration: i,
				part: r,
				voice: n,
				velocity: p,
				hit: !0
			});
			return;
		}
		let m = d(t, e)?.symbol;
		if (!m) return;
		if (n.role === "bass") {
			let t = 28 + (u(m).bassPc - 4 + 12) % 12, n = Math.max(1, Math.min(960, l));
			o.push({
				tick: e,
				duration: n,
				part: r,
				pitch: t,
				velocity: p,
				hit: !0
			});
			return;
		}
		let h = n.instrument === "clavinet" || n.instrument === "guitar", g = Math.max(1, Math.min(h ? 220 : 400, l)), _ = n.instrument === "organ" ? 57 : 55;
		for (let t of f(m, _)) o.push({
			tick: e,
			duration: g,
			part: r,
			pitch: t,
			velocity: p - .1,
			hit: !0,
			articulation: "staccato"
		});
	}), o;
}
function S(r, i, a = {}) {
	let o = e(r), s = [...i.spans].sort((e, t) => e.startBar - t.startBar).map((e) => ({
		span: e,
		start: (e.startBar - 1) * o,
		end: e.endBar * o
	})), c = (e) => s.find((t) => e >= t.start && e < t.end), l = [];
	r.parts.forEach((e, t) => {
		let n = (t) => t ? t.span.behaviours[e.id] ?? "on" : "on";
		if (v(e)) for (let r of e.drums ?? []) n(c(r.tick)) === "on" && l.push({
			tick: r.tick,
			duration: r.duration,
			part: t,
			voice: r.voice,
			velocity: r.velocity,
			...r.offset ? { offset: r.offset } : {}
		});
		else {
			let r = e.notes ?? [];
			for (let e = 0; e < r.length; e++) {
				let i = r[e], a = i.duration;
				for (; r[e].tieToNext && r[e + 1];) e++, a = r[e].tick + r[e].duration - i.tick;
				if (n(c(i.tick)) !== "on") continue;
				let o = i.tick + a;
				for (let e of s) e.start > i.tick && e.start < o && n(e) !== "on" && (o = e.start);
				let u = r[e].tieToNext ? 1 : r[e].gate ?? 1, d = Math.max(30, Math.round((o - i.tick) * (o === i.tick + a ? u : 1)));
				l.push({
					tick: i.tick,
					duration: d,
					part: t,
					pitch: i.pitch,
					velocity: i.velocity,
					...i.offset ? { offset: i.offset } : {},
					...i.articulation ? { articulation: i.articulation } : {}
				});
			}
		}
		for (let i of s) if (n(i) === "hit") for (let n = i.span.startBar; n <= i.span.endBar; n++) {
			let a = (n - 1) * o, s = b(i.span, e.id, n, o).map((e) => a + e);
			l.push(...x(r, e, t, s, i.end));
		}
	});
	let u = t(r), d = o / u;
	for (let e = 1; e <= r.bars; e++) {
		let t = s.find((t) => e >= t.span.startBar && e <= t.span.endBar), n = t ? t.span.behaviours.click ?? "on" : "on";
		if (n !== "off") for (let t = 0; t < d; t++) (n !== "backbeat" || t % 2 != 0) && l.push({
			tick: (e - 1) * o + t * u,
			duration: 60,
			part: r.parts.length,
			velocity: t === 0 ? 1 : .8,
			click: !0,
			...t === 0 ? { accent: !0 } : {}
		});
	}
	l.sort((e, t) => e.tick - t.tick || e.part - t.part || (e.pitch ?? 0) - (t.pitch ?? 0) || (e.voice ?? "").localeCompare(t.voice ?? ""));
	let f = r.subdivision === "swing8" ? {
		grid: 960,
		ratio: r.swingRatio ?? 2
	} : r.subdivision === "swing16" ? {
		grid: 480,
		ratio: r.swingRatio ?? 2
	} : null;
	return {
		id: r.id,
		barTicks: o,
		beatTicks: t(r),
		beatsPerBar: n(r),
		bars: r.bars,
		endTick: r.bars * o,
		startTick: a.skipIntro ? (h(r) - 1) * o : 0,
		loopStartTick: (r.loopBody.startBar - 1) * o,
		loopEndTick: r.loopBody.endBar * o,
		swing: f,
		parts: r.parts.map((e) => ({
			id: e.id,
			instrument: e.instrument,
			role: e.role
		})),
		events: l
	};
}
function C(e, t) {
	return Math.max(1.35, Math.min(2.6, e * (1 + (120 - t) / 420)));
}
function w(e, t, n) {
	let r = t ? T(e.tick, {
		grid: t.grid,
		ratio: C(t.ratio, n)
	}) : e.tick;
	return Math.max(0, r + (e.offset ?? 0));
}
function T(e, t) {
	if (!t) return e;
	let { grid: n, ratio: r } = t, i = Math.floor(e / n) * n, a = e - i, o = n / 2, s = n * r / (r + 1);
	return i + (a < o ? a * s / o : s + (a - o) * (n - s) / o);
}
var E = Math.PI * 2, D = (e) => 440 * 2 ** ((e - 69) / 12);
function O(e, t) {
	if (e < t) {
		let n = e / t;
		return n + n - n * n - 1;
	}
	if (e > 1 - t) {
		let n = (e - 1) / t;
		return n * n + n + n + 1;
	}
	return 0;
}
var k = (e, t) => 2 * e - 1 - O(e, t), A = 22222;
function j() {
	return A = 1664525 * A + 1013904223 >>> 0, A / 2147483648 - 1;
}
var M = [];
function N(e) {
	let t = M.pop();
	return t && t.length >= e ? t : new Float32Array(Math.max(e, 2048));
}
var P = class {
	part;
	age = 0;
	done = !1;
	remaining;
	released = -1;
	sr;
	hz;
	dt;
	phase = 0;
	phase2 = 0;
	phase3 = 0;
	modPhase = 0;
	lp = 0;
	lp2 = 0;
	velocity;
	kind;
	articulation;
	ks = null;
	ksLength = 0;
	ksIndex = 0;
	gain;
	bright;
	constructor(e, t, n, r, i, a, o = "") {
		if (this.part = i, this.kind = e, this.articulation = o, this.sr = a, this.hz = D(t), this.dt = this.hz / a, this.velocity = n, this.bright = .35 + .65 * n, this.remaining = r, this.gain = L[e] ?? .2, this.phase = t * .137 % 1, this.phase2 = t * .311 % 1, e === "guitar") {
			this.ksLength = Math.max(2, Math.round(a / this.hz)), this.ks = N(this.ksLength);
			let e = 0, n = .35 + .5 * this.bright, r = t * 2654435761 >>> 0 || 1;
			for (let t = 0; t < this.ksLength; t++) r = 1664525 * r + 1013904223 >>> 0, e = e * (1 - n) + (r / 2147483648 - 1) * n, this.ks[t] = e;
		}
	}
	release() {
		this.released < 0 && (this.released = this.age);
	}
	envelope(e, t, n, r) {
		let i = this.age / this.sr, a = i < e ? i / e : n + (1 - n) * Math.exp(-(i - e) / t);
		if (this.released >= 0) {
			let e = (this.age - this.released) / this.sr;
			a *= Math.exp(-e / r), e > r * 7 && (this.done = !0);
		}
		return a;
	}
	bend(e) {
		if (this.articulation === "scoop" && e < .12) return 2 ** (-.9 * (1 - e / .12) / 12);
		if (this.articulation === "fall" && this.released >= 0) {
			let e = (this.age - this.released) / this.sr;
			return 2 ** (-3 * Math.min(1, e / .18) / 12);
		}
		return 1;
	}
	lowpass(e, t) {
		let n = Math.min(.99, E * t / this.sr);
		return this.lp += n * (e - this.lp), this.lp;
	}
	bass(e, t, n) {
		let r = e.slide ? 2 ** (e.slide * Math.exp(-t / e.slideTime) / 12) : 1;
		this.phase = (this.phase + this.dt * r) % 1;
		let i = this.phase, a = Math.sin(E * i) + e.saw * k(i, this.dt) + e.h2 * Math.sin(2 * E * i) * Math.exp(-t * e.h2Decay) + e.h3 * Math.sin(3 * E * i) * Math.exp(-t * e.h3Decay), o = e.cutoff + e.brighten * this.bright * Math.exp(-t * e.brightDecay), s = j() * e.thump * Math.exp(-t * e.thumpDecay) * this.bright;
		return (this.lowpass(a, o) * e.level + s) * this.envelope(e.attack * (n ? .7 : 1), e.decay, e.sustain, e.release);
	}
	render() {
		let e = this.age / this.sr, t = this.dt, n = this.articulation === "legato", r = 0;
		switch (this.kind) {
			case "upright-bass":
			case "electric-bass":
			case "slap-bass": {
				let t = this.kind === "slap-bass" && (this.articulation === "pop" || this.articulation === "ghost") ? `slap-bass:${this.articulation}` : this.kind;
				r = this.bass(I[t], e, n);
				break;
			}
			case "synth-bass": {
				this.phase = (this.phase + t) % 1, this.phase2 = (this.phase2 + t * 1.004) % 1;
				let n = .5 * (k(this.phase, t) + k(this.phase2, t)), i = 160 + 2200 * this.bright * Math.exp(-e * 16);
				this.lp2 += Math.min(.99, E * i / this.sr) * (this.lowpass(n, i) - this.lp2), r = (this.lp2 * 1.4 + .45 * Math.sin(E * this.phase)) * this.envelope(.002, .35, .55, .03);
				break;
			}
			case "piano": {
				this.phase = (this.phase + t) % 1;
				let n = this.phase, i = Math.sin(E * n) + .5 * Math.sin(2 * E * n * 1.0007) * Math.exp(-e * 2.2) + .3 * Math.sin(3 * E * n * 1.0015) * Math.exp(-e * 3.5) + .18 * Math.sin(4 * E * n * 1.003) * Math.exp(-e * 5) + .1 * Math.sin(6 * E * n * 1.006) * Math.exp(-e * 8), a = j() * .08 * Math.exp(-e * 200) * this.bright;
				r = (this.lowpass(i, 900 + 5200 * this.bright * Math.exp(-e * 2)) + a) * this.envelope(.002, .6 + 50 / this.hz, .1, this.articulation === "staccato" ? .05 : .14);
				break;
			}
			case "electric-piano":
			case "wurlitzer": {
				let n = this.kind === "wurlitzer";
				this.phase = (this.phase + t) % 1, this.phase3 = (this.phase3 + t * 1.0023) % 1, this.modPhase = (this.modPhase + t) % 1;
				let i = ((n ? 2.2 : 1.5) * this.bright * Math.exp(-e * (n ? 5 : 3.2)) + (n ? .45 : .18)) * Math.sin(E * this.modPhase), a = .6 * Math.sin(E * this.phase + i) + .4 * Math.sin(E * this.phase3 + i);
				this.phase2 = (this.phase2 + t * 14.2) % 1;
				let o = n ? 0 : .2 * this.bright * Math.sin(E * this.phase2) * Math.exp(-e * 18), s = n ? Math.tanh(a * (1.2 + this.bright)) : a, c = 1 - .12 * (.5 + .5 * Math.sin(E * 4.8 * e));
				r = (s + o) * c * this.envelope(.002, n ? .9 : 1.5, .16, this.articulation === "staccato" ? .04 : .09);
				break;
			}
			case "organ": {
				let n = E * 6.4 * e;
				this.phase = (this.phase + t * (1 + .0028 * Math.sin(n))) % 1;
				let i = this.phase, a = this.articulation === "swell" ? 0 : Math.exp(-e * 9), o = e < .006 ? j() * .25 * (1 - e / .006) : 0, s = .55 * Math.sin(E * i * .5) + Math.sin(E * i) + .65 * Math.sin(E * i * 1.5) + .5 * Math.sin(E * i * 2) + .3 * Math.sin(E * i * 4) + .45 * a * Math.sin(E * i * 3), c = this.articulation === "swell" ? .22 : this.articulation === "staccato" ? .004 : .008;
				r = (s * (1 - .18 * (.5 + .5 * Math.sin(n + 1.3))) + o) * .55 * this.envelope(c, 10, 1, this.articulation === "staccato" ? .015 : .03);
				break;
			}
			case "clavinet": {
				this.phase = (this.phase + t) % 1;
				let n = k(this.phase, t) - k((this.phase + .16) % 1, t), i = 600 + 5200 * this.bright * Math.exp(-e * 10);
				this.lp2 += Math.min(.99, E * i * 1.4 / this.sr) * (this.lowpass(n, i) - this.lp2), r = (this.lp2 * .7 + (n - this.lp) * .12 * this.bright) * this.envelope(.001, .3, .1, .012);
				break;
			}
			case "guitar": {
				let e = this.ks;
				if (!e) break;
				let t = this.ksIndex, n = (t + 1) % this.ksLength, i = e[t];
				e[t] = (this.released >= 0 ? .9 : .996) * .5 * (e[t] + e[n]), this.ksIndex = n, r = this.lowpass(i, 1800 + 3e3 * this.bright) * 1.6 * this.envelope(.001, 3, 1, .03), this.done && (M.push(e), this.ks = null);
				break;
			}
			case "strings": {
				let n = 1 + .003 * Math.sin(E * 5 * e);
				this.phase = (this.phase + t * n) % 1, this.phase2 = (this.phase2 + t * n * 1.004) % 1, this.phase3 = (this.phase3 + t * n * .996) % 1;
				let i = (k(this.phase, t) + k(this.phase2, t) + k(this.phase3, t)) / 3;
				this.lp2 += Math.min(.99, E * (1400 + 900 * this.bright) / this.sr) * (this.lowpass(i, 2400) - this.lp2), r = this.lp2 * this.envelope(.22, 2, .85, .25);
				break;
			}
			case "synth-keys": {
				this.phase = (this.phase + t) % 1, this.phase2 = (this.phase2 + t * 1.008) % 1, this.phase3 = (this.phase3 + t * .5) % 1;
				let n = .4 * (k(this.phase, t) + k(this.phase2, t)) + .2 * (this.phase3 < .5 ? 1 : -1), i = 500 + 2600 * this.bright * Math.exp(-e * 6);
				this.lp2 += Math.min(.99, E * i / this.sr) * (this.lowpass(n, i) - this.lp2), r = this.lp2 * this.envelope(.006, .6, .45, .12);
				break;
			}
			case "synth-lead": {
				let i = 1 + .0045 * Math.sin(E * 5.5 * e) * Math.min(1, Math.max(0, e - .2) * 2), a = this.bend(e);
				this.phase = (this.phase + t * i * a) % 1, this.phase2 = (this.phase2 + t * i * a * 1.006) % 1;
				let o = .5 * (k(this.phase, t) + k(this.phase2, t)), s = 700 + 3200 * this.bright * (.35 + .65 * Math.exp(-e * 3));
				this.lp2 += Math.min(.99, E * s / this.sr) * (this.lowpass(o, s) - this.lp2), r = this.lp2 * 1.1 * this.envelope(n ? .004 : .012, .5, .75, .08);
				break;
			}
			case "trumpet":
			case "tenor-sax":
			case "alto-sax":
			case "flute":
			case "sax-pad":
			case "soprano-sax":
			case "baritone-sax":
			case "flugelhorn":
			case "trombone":
			case "clarinet": {
				let i = F[this.kind], a = i.source === "flute", o = this.kind === "sax-pad", s = 1 + i.vibrato * Math.sin(E * i.rate * e) * Math.min(1, Math.max(0, e - .18) * 3);
				this.phase = (this.phase + t * s * this.bend(e)) % 1;
				let c = o ? .09 : n ? .008 : i.attack, l = this.envelope(c, .35, o ? .85 : .72, this.articulation === "fall" ? .12 : this.articulation === "staccato" ? .025 : i.release), u = n ? 0 : Math.exp(-e * 18), d, f;
				if (a) d = Math.sin(E * this.phase) + .12 * Math.sin(2 * E * this.phase), d += j() * .05 * (.4 + u), f = 2600 + 1200 * this.bright;
				else {
					let e = this.phase, n = i.square ? k(e, t) - k((e + .5) % 1, t) : e < .5 ? 1 : -1;
					d = i.source === "brass" ? k(e, t) : i.source === "clarinet" ? .85 * n + .15 * k(e, t) : (1 - i.reed) * k(e, t) + i.reed * n, d += j() * i.breath * u, f = i.cutoff + i.brighten * l * this.bright;
				}
				let p = Math.min(.99, E * f / this.sr);
				this.lp += p * (d - this.lp), this.lp2 += p * (this.lp - this.lp2), r = (a ? this.lp : this.lp2) * l, i.drive && (r = Math.tanh(r * (1 + i.drive)) / (1 + i.drive * .5));
				break;
			}
			case "vibraphone": {
				this.phase = (this.phase + t) % 1, this.phase2 = (this.phase2 + t * 4) % 1, this.phase3 = (this.phase3 + t * 10) % 1;
				let n = Math.sin(E * this.phase) + .32 * this.bright * Math.sin(E * this.phase2) * Math.exp(-e * 6) + .1 * this.bright * Math.sin(E * this.phase3) * Math.exp(-e * 18), i = j() * .05 * this.bright * Math.exp(-e * 320);
				r = (n * (1 - .22 * (.5 + .5 * Math.sin(E * 5.4 * e))) + i) * this.envelope(.0015, 1.9, 0, this.articulation === "staccato" ? .06 : .22);
				break;
			}
			default: this.phase = (this.phase + t) % 1, r = Math.sin(E * this.phase) * this.envelope(.005, .5, .5, .05);
		}
		return this.age++, this.released < 0 && this.age > this.sr * 12 && this.release(), r * this.velocity * this.gain;
	}
}, F = {
	trumpet: {
		source: "brass",
		vibrato: .0032,
		rate: 5.2,
		attack: .022,
		release: .05,
		cutoff: 550,
		brighten: 3600,
		breath: .04,
		reed: 0
	},
	flugelhorn: {
		source: "brass",
		vibrato: .0036,
		rate: 5,
		attack: .03,
		release: .07,
		cutoff: 430,
		brighten: 1900,
		breath: .03,
		reed: 0,
		square: !0
	},
	trombone: {
		source: "brass",
		vibrato: .0024,
		rate: 4.8,
		attack: .032,
		release: .07,
		cutoff: 360,
		brighten: 2500,
		breath: .035,
		reed: 0,
		square: !0
	},
	"tenor-sax": {
		source: "reed",
		vibrato: .0032,
		rate: 5.2,
		attack: .022,
		release: .05,
		cutoff: 420,
		brighten: 2600,
		breath: .04,
		reed: .3
	},
	"alto-sax": {
		source: "reed",
		vibrato: .0032,
		rate: 5.2,
		attack: .022,
		release: .05,
		cutoff: 420,
		brighten: 2600,
		breath: .04,
		reed: .3
	},
	"sax-pad": {
		source: "reed",
		vibrato: .0032,
		rate: 5.2,
		attack: .09,
		release: .05,
		cutoff: 420,
		brighten: 900,
		breath: .01,
		reed: .3
	},
	"soprano-sax": {
		source: "reed",
		vibrato: .0042,
		rate: 5.6,
		attack: .02,
		release: .05,
		cutoff: 620,
		brighten: 3200,
		breath: .035,
		reed: .2,
		square: !0
	},
	"baritone-sax": {
		source: "reed",
		vibrato: .0026,
		rate: 4.8,
		attack: .03,
		release: .06,
		cutoff: 300,
		brighten: 1800,
		breath: .05,
		reed: .42,
		square: !0,
		drive: .8
	},
	clarinet: {
		source: "clarinet",
		vibrato: .0014,
		rate: 5,
		attack: .026,
		release: .05,
		cutoff: 520,
		brighten: 2300,
		breath: .03,
		reed: 0,
		square: !0
	},
	flute: {
		source: "flute",
		vibrato: .005,
		rate: 5.2,
		attack: .035,
		release: .05,
		cutoff: 0,
		brighten: 0,
		breath: .05,
		reed: 0
	}
}, I = {
	"electric-bass": {
		attack: .003,
		decay: .8,
		sustain: .4,
		release: .035,
		cutoff: 220,
		brighten: 1600,
		brightDecay: 10,
		saw: .55,
		h2: 0,
		h2Decay: 0,
		h3: 0,
		h3Decay: 0,
		thump: .04,
		thumpDecay: 150,
		slide: 0,
		slideTime: 1,
		level: 1.3
	},
	"upright-bass": {
		attack: .006,
		decay: .5,
		sustain: .1,
		release: .06,
		cutoff: 450,
		brighten: 1100,
		brightDecay: 8,
		saw: 0,
		h2: .38,
		h2Decay: 3,
		h3: .14,
		h3Decay: 6,
		thump: .25,
		thumpDecay: 90,
		slide: .35,
		slideTime: .025,
		level: 1.3
	},
	"slap-bass": {
		attack: .001,
		decay: .45,
		sustain: .25,
		release: .03,
		cutoff: 700,
		brighten: 5200,
		brightDecay: 18,
		saw: .8,
		h2: .5,
		h2Decay: 5,
		h3: .35,
		h3Decay: 9,
		thump: .5,
		thumpDecay: 260,
		slide: 0,
		slideTime: 1,
		level: 1.1
	},
	"slap-bass:pop": {
		attack: 8e-4,
		decay: .3,
		sustain: .12,
		release: .03,
		cutoff: 1400,
		brighten: 6500,
		brightDecay: 22,
		saw: .9,
		h2: .6,
		h2Decay: 6,
		h3: .5,
		h3Decay: 10,
		thump: .6,
		thumpDecay: 400,
		slide: 0,
		slideTime: 1,
		level: .95
	},
	"slap-bass:ghost": {
		attack: .001,
		decay: .03,
		sustain: 0,
		release: .015,
		cutoff: 300,
		brighten: 2500,
		brightDecay: 40,
		saw: .5,
		h2: 0,
		h2Decay: 0,
		h3: 0,
		h3Decay: 0,
		thump: .55,
		thumpDecay: 120,
		slide: 0,
		slideTime: 1,
		level: .9
	}
}, L = {
	"slap-bass": .5,
	"upright-bass": .24,
	"electric-bass": .32,
	"synth-bass": .3,
	piano: .16,
	"electric-piano": .17,
	wurlitzer: .18,
	organ: .25,
	clavinet: .41,
	guitar: .34,
	"synth-lead": .22,
	"synth-keys": .4,
	trumpet: .24,
	"tenor-sax": .7,
	"alto-sax": .71,
	flute: .205,
	"sax-pad": .8,
	strings: .35,
	"soprano-sax": .2,
	"baritone-sax": .26,
	flugelhorn: .25,
	trombone: .27,
	clarinet: .185,
	vibraphone: .22
}, R = 3.9, z = {
	kick: !0,
	snare: !0,
	"snare.rim": !0,
	"hihat.closed": !0,
	"hihat.open": !0,
	"tom.high": !0,
	"tom.low": !0,
	crash: !0,
	ride: !0,
	cowbell: !0
}, B = .5, V = /* @__PURE__ */ new Set([
	"hihat.closed",
	"hihat.open",
	"hihat.foot",
	"crash",
	"ride"
]), H = {
	shaker: 2.6,
	guiro: 1.7,
	clave: 1.8
}, U = class {
	part;
	age = 0;
	done = !1;
	remaining = Infinity;
	released = -1;
	sample;
	gain;
	voice;
	sr;
	lp = 0;
	limit;
	constructor(e, t, n, r, i, a) {
		this.part = n, this.voice = e, this.sr = r;
		let o = e === "hihat.foot" ? "hihat.closed" : e, s = e === "snare" ? t < .4 ? 0 : t < .8 ? 1 : t < .95 ? 2 : 3 : 0;
		this.sample = i[`${o}-${s}-${a % 2}`], this.gain = t ** .9 * (e === "hihat.foot" ? .22 : .55) * (o in z ? R : 1) * (V.has(e) ? B : 1) * (H[e] ?? 1), this.limit = e === "hihat.foot" ? Math.round(r * .09) : this.sample?.length ?? Math.round(r * .6);
	}
	release() {
		this.voice === "hihat.open" && (this.limit = Math.min(this.limit, this.age + Math.round(this.sr * .03)));
	}
	render() {
		let e = this.age / this.sr, t;
		if (this.sample) t = (this.sample[this.age] ?? 0) * this.gain, this.age > this.limit - 64 && (t *= Math.max(0, (this.limit - this.age) / 64));
		else {
			let n = this.voice;
			if (n === "conga.high" || n === "conga.low") {
				let r = (n === "conga.high" ? 330 : 220) * (1 + .12 * Math.exp(-e * 40));
				t = Math.sin(E * r * e) * Math.exp(-e * 11) + j() * .08 * Math.exp(-e * 80);
			} else if (n === "conga.slap") t = Math.sin(E * 420 * e) * Math.exp(-e * 30) * .6 + j() * .6 * Math.exp(-e * 55);
			else if (n === "timbale.high" || n === "timbale.low") {
				let r = n === "timbale.high" ? 690 : 480;
				t = (Math.sin(E * r * e) + .5 * Math.sin(E * r * 1.71 * e)) * Math.exp(-e * 7) * .6 + j() * .35 * Math.exp(-e * 45);
			} else if (n === "shaker") {
				let n = j();
				this.lp += .5 * (n - this.lp), t = (n - this.lp) * Math.min(1, e / .012) * Math.exp(-e * 28) * .7;
			} else if (n === "guiro") {
				let n = j();
				this.lp += .4 * (n - this.lp);
				let r = .5 + .5 * Math.sin(E * 38 * e);
				t = (n - this.lp) * r * (e < .16 ? 1 : Math.exp(-(e - .16) * 60)) * .55;
			} else t = n === "clave" ? Math.sin(E * 2450 * e) * Math.exp(-e * 55) * .8 : j() * Math.exp(-e * 30) * .3;
			t *= this.gain * .9, e > .6 && (this.done = !0);
		}
		return this.age++, this.age >= this.limit && (this.done = !0), t;
	}
}, W = class {
	part;
	age = 0;
	done = !1;
	remaining = Infinity;
	released = -1;
	hz;
	sr;
	gain;
	constructor(e, t, n, r) {
		this.part = e, this.hz = t ? 1600 : 1200, this.sr = r, this.gain = n;
	}
	release() {}
	render() {
		let e = this.age++ / this.sr;
		return e > .05 && (this.done = !0), Math.sin(E * this.hz * e) * Math.exp(-e * 90) * Math.min(1, e / 8e-4) * this.gain;
	}
}, G = 64, K = .22, q = .8;
function J(e) {
	let t = Math.abs(e);
	if (t <= .8) return e;
	let n = .19999999999999996, r = q + n * Math.tanh((t - q) / n);
	return e < 0 ? -r : r;
}
var Y = class {
	sampleRate;
	song = null;
	state = "ready";
	tempo = 96;
	loop = !0;
	countInEnabled = !0;
	revision = 0;
	bank = {};
	renderTicks = /* @__PURE__ */ new Float64Array();
	eventIndex = 0;
	songPos = 0;
	dispatched = -1;
	countPos = 0;
	countTicks = 0;
	startTick = 0;
	nextCount = 0;
	pass = 0;
	voices = [];
	targets = [];
	gains = [];
	rounds = {};
	tailFrames = 0;
	frameCounter = 0;
	playedFrames = 0;
	resets = 0;
	limitedFrames = 0;
	busPeak = 0;
	constructor(e) {
		this.sampleRate = e;
	}
	command(e) {
		switch (e.type) {
			case "samples":
				this.bank = e.bank;
				return;
			case "configure": {
				let t = this.song?.id === e.compiled.id, n = this.home();
				this.song = e.compiled, this.tempo = e.tempo, this.renderTicks = Float64Array.from(e.compiled.events, (t) => w(t, e.compiled.swing, e.tempo)), this.loop = e.loop, this.countInEnabled = e.countIn, this.setLevels(e.levels, !0), !t || this.state === "complete" ? this.reset(this.home()) : this.state === "ready" ? this.reset(this.startTick === n ? this.home() : Math.min(this.startTick, e.compiled.endTick - 1)) : this.reindex();
				break;
			}
			case "levels":
				this.setLevels(e.levels, !1);
				break;
			case "tempo":
				if (!Number.isInteger(e.value) || e.value < 10 || e.value > 480) throw RangeError("Invalid tempo");
				if (this.tempo = e.value, this.song?.swing) for (let e = this.eventIndex; e < this.renderTicks.length; e++) this.renderTicks[e] = w(this.song.events[e], this.song.swing, this.tempo);
				break;
			case "loop":
				this.loop = e.value;
				break;
			case "countIn":
				this.countInEnabled = e.value, this.state === "ready" && (this.countTicks = this.leadIn());
				break;
			case "seek": {
				let t = this.song;
				if (!t || !Number.isInteger(e.tick) || e.tick < 0 || e.tick >= t.endTick) throw RangeError("Invalid seek position");
				if (this.state === "playing") {
					this.songPos = e.tick, this.countPos = this.countTicks, this.dispatched = e.tick - 1e-6;
					for (let e of this.voices) e instanceof P && e.release();
					this.reindex();
				} else this.state === "ready" || this.state === "complete" ? this.reset(e.tick) : (this.songPos = e.tick, this.countPos = this.countTicks, this.dispatched = e.tick - 1e-6, this.reindex());
				break;
			}
			case "play":
				if (!this.song) break;
				this.state === "complete" && this.reset(this.home()), this.state === "ready" && (this.countTicks = this.leadIn()), this.state = "playing";
				break;
			case "pause":
				this.state === "playing" && (this.state = "paused"), this.voices = [];
				break;
			case "stop": this.reset(this.home());
		}
		this.revision = e.revision;
	}
	syncComplete() {
		this.state === "playing" && this.song && (this.songPos = this.song.endTick, this.state = "complete", this.voices = []);
	}
	setLevels(e, t) {
		this.targets = [
			...e.gains,
			e.click,
			.55
		], (t || this.gains.length !== this.targets.length) && (this.gains = [...this.targets]);
	}
	home() {
		return Math.min(this.song?.startTick ?? 0, (this.song?.endTick ?? 1) - 1);
	}
	leadIn() {
		return !this.countInEnabled || !this.song ? 0 : this.song.barTicks * (this.tempo < 120 ? 1 : 2);
	}
	reset(e) {
		this.state = "ready", this.startTick = e, this.songPos = e, this.dispatched = e - 1e-6, this.countPos = 0, this.nextCount = 0, this.countTicks = this.leadIn(), this.pass = 0, this.voices = [], this.tailFrames = 0, this.playedFrames = 0, this.resets++, this.reindex();
	}
	reindex() {
		let e = 0;
		for (; e < this.renderTicks.length && this.renderTicks[e] <= this.dispatched;) e++;
		this.eventIndex = e;
	}
	view() {
		let e = this.song, t = e?.barTicks ?? 3840, n = e?.beatTicks ?? 960, r = (this.state === "playing" || this.state === "paused") && this.countPos < this.countTicks - 1e-6, i = Math.max(0, Math.floor(this.songPos + 1e-6)), a = e ? Math.min(i, e.endTick - 1) : i;
		return {
			state: this.state,
			tempo: this.tempo,
			revision: this.revision,
			countIn: r,
			countRows: this.countTicks ? Math.round(this.countTicks / t) : 0,
			countPulses: e?.beatsPerBar ?? 4,
			countFilled: r ? Math.floor(this.countPos / n) + 1 : 0,
			bar: Math.floor(a / t) + 1,
			beat: Math.floor(a % t / n) + 1,
			tick: a,
			pass: this.pass,
			loop: this.loop,
			finishedAudio: this.state === "complete" && this.tailFrames >= this.sampleRate * .6,
			elapsedSeconds: Math.floor(this.playedFrames / this.sampleRate)
		};
	}
	trigger(e, t, n = 0) {
		let r = this.song.parts[e.part];
		if (e.click) this.targets[e.part] > 0 && this.voices.push(new W(e.part, !!e.accent, 1, this.sampleRate)), t?.push({
			frame: n,
			tick: e.tick,
			pass: this.pass,
			part: e.part,
			click: "beat"
		});
		else {
			if (e.voice) {
				let t = e.voice, n = this.rounds[t] = (this.rounds[t] ?? 0) + 1;
				if (e.voice === "hihat.closed" || e.voice === "hihat.foot") for (let t of this.voices) t instanceof U && t.part === e.part && t.release();
				this.voices.push(new U(e.voice, e.velocity, e.part, this.sampleRate, this.bank, n));
			} else e.pitch !== void 0 && this.voices.push(new P(r.instrument, e.pitch, e.velocity, e.duration, e.part, this.sampleRate, e.articulation));
			this.voices.length > G && this.voices.splice(0, this.voices.length - G), t?.push({
				frame: n,
				tick: e.tick,
				pass: this.pass,
				part: e.part,
				...e.pitch === void 0 ? {} : { pitch: e.pitch },
				...e.voice ? { voice: e.voice } : {}
			});
		}
	}
	render(e, t) {
		e.fill(0);
		let n = this.song;
		if (!n) return;
		if (this.state === "complete") {
			this.mix(e, 0, e.length), this.tailFrames += e.length;
			return;
		}
		if (this.state !== "playing") return;
		let r = n.parts.length + 1, i = 1 - Math.exp(-1 / (.01 * this.sampleRate));
		for (let a = 0; a < e.length; a++) {
			let o = this.tempo * n.beatTicks / 60 / this.sampleRate, s = this.frameCounter + a;
			if (this.countPos < this.countTicks - 1e-6) {
				if (this.countPos >= this.nextCount - 1e-7) {
					let e = Math.round(this.nextCount / n.beatTicks);
					this.voices.push(new W(r, e % n.beatsPerBar === 0, .5, this.sampleRate)), t?.push({
						frame: s,
						tick: -1,
						pass: this.pass,
						part: r,
						click: "count"
					}), this.nextCount += n.beatTicks;
				}
				this.countPos += o, this.countPos >= this.countTicks - 1e-6 && (this.songPos += Math.max(0, this.countPos - this.countTicks), this.countPos = this.countTicks);
			} else {
				for (this.playedFrames++; this.eventIndex < n.events.length && this.renderTicks[this.eventIndex] <= this.songPos + 1e-7;) this.trigger(n.events[this.eventIndex], t, s), this.eventIndex++;
				this.dispatched = this.songPos + 1e-7;
				let r = this.songPos;
				this.songPos += o;
				for (let e of this.voices) e instanceof P && (e.remaining -= o, e.remaining <= 0 && e.release());
				if (this.loop && r < n.loopEndTick && this.songPos >= n.loopEndTick) {
					this.songPos = n.loopStartTick + (this.songPos - n.loopEndTick), this.dispatched = n.loopStartTick - 1e-6, this.pass++;
					for (let e of this.voices) e instanceof P && e.release();
					this.reindex();
				} else if (this.songPos >= n.endTick) {
					this.songPos = n.endTick, this.state = "complete", this.tailFrames = 0;
					for (let e of this.voices) e.release();
					this.mix(e, a, a + 1, i), this.mix(e, a + 1, e.length), this.frameCounter += e.length;
					return;
				}
			}
			this.mix(e, a, a + 1, i);
		}
		this.frameCounter += e.length;
	}
	mix(e, t, n, r = 1 - Math.exp(-1 / (.01 * this.sampleRate))) {
		let i = this.gains, a = this.targets;
		for (let o = t; o < n; o++) {
			for (let e = 0; e < i.length; e++) i[e] += (a[e] - i[e]) * r;
			let t = 0;
			for (let e of this.voices) e.done || (t += e.render() * (i[e.part] ?? 0));
			let n = t * K;
			n > this.busPeak ? this.busPeak = n : -n > this.busPeak && (this.busPeak = -n), n > .8 || n < -.8 ? (this.limitedFrames++, e[o] = J(n)) : e[o] = n, (o & 63) == 63 && (this.voices = this.voices.filter((e) => !e.done));
		}
	}
};
export { a, d as c, h as d, m as f, r as g, g as h, b as i, _ as l, s as m, y as n, e as o, u as p, S as r, t as s, Y as t, p as u };

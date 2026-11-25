"""Additional kit grooves, arranged around repeating feet and hand parts."""


def add_families(family):
    family("motorik rock", "rock", "4/4", 127, "driving even pulse",
           "Keep the bass drum on every quarter note and the snare on two and four.",
           (("even eighth-note closed hats", "h:0 2 4 6 8 10 12 14"),
            ("a straight eighth-note ride", "r:0 2 4 6 8 10 12 14"),
            ("occasional open-hat lifts on the second and fourth offbeats", "h:0 2 4 8 10 12;o:6 14"),
            ("quarter-note ride strokes and a final offbeat", "r:0 4 8 12 14")),
           ("k:0 4 8 12;l:2 10", "k:0 4 8 12;l:6 14", "k:0 4 8 12;l:2 6 10",
            "k:0 4 8 12;l:3 11", "k:0 4 8 12;l:6 11 14"), hands="s:4 12")

    family("post-punk rock", "rock", "4/4", 153, "lean offbeat drive",
           "Use a dry backbeat and a steady floor-tom answer after beat three.",
           (("eighth-note hats", "h:0 2 4 6 8 10 12 14"),
            ("an offbeat-only hat pulse", "h:2 6 10 14"),
            ("open hats on the second and fourth offbeats", "h:0 2 4 8 10 12;o:6 14"),
            ("a steady quarter-note ride", "r:0 4 8 12")),
           ("k:0 2 7 9 14", "k:0 3 6 8 13", "k:0 2 5 9 15",
            "k:0 6 7 8 14", "k:0 1 6 10 15"), hands="s:4 12;l:10")

    family("five-four riff rock", "rock", "5/4", 118, "heavy three plus two",
           "Put bass drum accents at the starts of both groups and answer with the snare.",
           (("quarter hats with a late eighth", "h:0 4 8 12 16 18"),
            ("an even eighth-note ride", "r:0 2 4 6 8 10 12 14 16 18"),
            ("one open hat in each group", "h:0 2 4 8 10 12 14 16;o:6 18"),
            ("a ride that leaves space after the group accents", "r:0 4 6 8 12 16 19")),
           ("k:0 6 12 18", "k:0 3 8 12 16", "k:0 7 12 14 19",
            "k:0 5 10 12 17", "k:0 2 7 11 12 18"),
           hands="s:4 10 16", grouping=(3, 2))

    family("linear funk", "funk", "4/4", 104, "single-hit pocket",
           "Keep the backbeat on two and four. Give each written onset to one voice.",
           (("a broken closed hat", "h:1 5 9 13"),
            ("a short ride reply on the and of two and four", "r:1 5 6 9 13 14"),
            ("closed hats with one open lift", "h:1 5 9;o:13"),
            ("a sparse cowbell", "c:1 9 13")),
           ("k:0 2 8 10;g:3 11", "k:0 7 8 15;g:2 10", "k:0 3 8 11;g:7 15",
            "k:0 2 7 10;g:3 8 11", "k:0 3 8 15;g:2 7 11"),
           hands="s:4 12")

    family("syncopated rim funk", "funk", "4/4", 107, "rim and ghost conversation",
           "Let the rim click carry the backbeat, with low snare replies between clicks.",
           (("closed hats accenting the eighths", "h:0 2* 4 6* 8 10* 12 14*"),
            ("a clipped four-note ride", "r:0 3 8 11"),
            ("small open-hat accents after one and three", "h:0 4 6 8 12 14;o:2 10"),
            ("a bell with gaps", "c:0 2 5 8 10 14")),
           ("k:0 3 7 10;g:2 9 14", "k:0 2 5 8 14;g:3 6 13", "k:0 6 9 11;g:1 8 15",
            "k:0 1 7 10 15;g:3 10", "k:0 4 8 13;g:2 5 15"), hands="x:4 12")

    family("laid-back sixteenth funk", "funk", "4/4", 92, "late backbeat and quick hands",
           "Hold the big snare slightly late while a quiet snare answers in the gaps.",
           (("broken sixteenth-note hats", "h:0 1 2 4 5 6 8 9 10 12 13 14"),
            ("a back-and-forth shaker", "z:0 1 4 6 8 9 12 14"),
            ("offbeat ride accents", "r:2* 5 6* 10* 13 14*"),
            ("an open hat near the end of each bar", "h:0 1 2 4 6 8 9 10 12;o:14")),
           ("k:0 3 8 15", "k:0 2 6 9 13", "k:0 5 7 11 14",
            "k:0 1 5 8 10", "k:0 3 6 9 14"), hands="s:4 12;g:3 7 11 15", lag=3)

    family("jump blues shuffle", "blues", "4/4", 146, "brisk long-short shuffle",
           "Keep the snare on two and four and make the quick offbeats light.",
           (("a shuffling ride", "r:0 2 3 5 6 8 9 11"),
            ("tight long-short hats", "h:0 2 3 5 6 8 9 11"),
            ("ride skipping after one and three", "r:0 2 3 6 8 9"),
            ("closed hats with an open final skip", "h:0 2 3 5 6 8 9;o:11")),
           ("k:0 2 6 9;g:4", "k:0 3 6 8;g:10", "k:0 2 5 8 11;g:1",
            "k:0 3 5 6 9;g:7", "k:0 4 6 9 11;g:4 10"), hands="s:3 9", unit=32, light_offbeat=64)

    family("blues ballad", "blues", "6/8", 63, "spacious compound backbeat",
           "Give the second dotted-quarter pulse a clear snare and leave room around it.",
           (("a soft ride on both pulses and the last eighth", "r:0 6 10"),
            ("closed hats on all six eighths", "h:0 2 4 6 8 10"),
            ("shaker grouped in threes", "z:0 2 4 6 8 10"),
            ("a broken ride with a kick-led pickup", "r:0 4 6 8")),
           ("k:0 2 8;g:4", "k:0 4 10;g:8", "k:0 3 7;g:2 10",
            "k:0 2 5 9;g:8", "k:0 4 8;g:2"), hands="s:6", grouping=(3, 3))

    family("bebop jazz", "jazz", "4/4", 174, "light triplet ride and comping",
           "Feather the kick and close the foot hat on two and four.",
           (("ride skips after both backbeats", "r:0 3 5 6 9 11"),
            ("a quick skip after beat two", "r:0 3 5 6 9"),
            ("ride skips after one and four", "r:0 2 3 6 9 11"),
            ("a light quarter-note ride", "r:0 3 6 9")),
           ("k:0 6;g:1 4 10", "k:0 3 6 9;g:2 5 8", "k:0 6 11;g:1 4",
            "k:0 2 9;g:5 8 10", "k:0 5 6 9;g:2 4 7"), hands="p:3 9", unit=32)

    family("even-eighth jazz", "jazz", "4/4", 123, "even ride and asymmetric comping",
           "Keep the foot hat on two and four while the quiet snare comps independently.",
           (("an even eighth-note ride", "r:0 2 4 6 8 10 12 14"),
            ("quarter-note ride", "r:0 4 8 12"),
            ("ride skipping an eighth after one and three", "r:0 4 6 8 12 14"),
            ("a syncopated ride", "r:0 3 4 6 8 11 12 14")),
           ("k:0 8;g:2 7 14", "k:0 6 10;g:3 9 15", "k:0 8 13;g:2 5 10",
            "k:0 4 8 12;g:6 11 15", "k:0 3 8 11;g:5 10 14"), hands="p:4 12")

    family("modal jazz waltz", "jazz", "3/4", 122, "three-beat ride with space",
           "Let a soft foot hat mark beat two and keep the snare answers understated.",
           (("an even eighth-note ride", "r:0 2 4 6 8 10"),
            ("ride spaced on the three quarter notes", "r:0 4 8"),
            ("ride accents at the beginning and end", "r:0* 4 8* 10"),
            ("ride moving across the second beat", "r:0 3 6 8 10")),
           ("k:0 8;g:2 6", "k:0 6;g:3 10", "k:0 4 8;g:2 7 11",
            "k:0 3 8;g:5 10", "k:0 7;g:2 5 9"), hands="p:4")

    family("cha-cha-cha latin kit", "latin", "4/4", 114, "two-bar dance pocket",
           "Use an answered rim pattern and small bass drum anticipations.",
           (("cowbell quarters and the last offbeat", "c:0 4 8 12 14"),
            ("bell quarter notes with light eighths", "c:0 4 6 8 12 14"),
            ("a light closed hat", "h:0 2 4 6 8 10 12 14"),
            ("shaker on the eighths", "z:0 2 4 6 8 10 12 14")),
           ("k:0 6 10 14|k:0 6 8 14", "k:0 6 8 15|k:0 6 12 14", "k:0 5 8 14|k:0 6 10 14",
            "k:0 6 11 14|k:0 3 8 14", "k:0 6 10 15|k:0 6 8 13"),
           hands="x:4 10 12|x:2 8 14", phrase_bars=2)

    family("samba partido alto latin", "latin", "4/4", 127, "alternating samba accents",
           "Keep a seven-stroke rim timeline against the anticipatory samba bass drum.",
           (("a sparse ride", "r:0 3 6 8 11 14|r:2 4 7 10 12 15"),
            ("broken sixteenth-note hats", "h:0 3 6 8 11 14|h:2 4 7 10 12 15"),
            ("a shaker with alternating accents", "z:0* 2 3 6* 8 11 14*|z:0 2* 4 7* 10 12* 15"),
            ("a cowbell accent cycle", "c:0* 3 6* 8 11 14*|c:2* 4 7* 10 12* 15")),
           ("k:0 3 4 7 8 11 12 15", "k:0 3 4 8 11 12 15", "k:0 3 4 7 8 12 15",
            "k:0 3 4 7 8 10 12 15", "k:0 2 4 7 8 11 12 15"),
           hands="x:1 4 6 8 10 13 15|x:1 4 6 8 10 13 15;p:4 12", phrase_bars=2)

    family("six-eight Afro-Cuban latin", "latin", "6/8", 84, "two-bar compound bell timeline",
           "Repeat the seven-stroke bell timeline over two bars, with floor-tom and rim replies.",
           (("the timeline voiced on cowbell", "c:0 4 8 10|c:2 6 10"),
            ("the timeline voiced on ride", "r:0 4 8 10|r:2 6 10"),
            ("clave taking over the second bar", "c:0 4 8 10|v:2 6 10"),
            ("the timeline shared between cowbell and a closed hat", "c:0 4;h:8 10|h:2;c:6 10")),
           ("k:0 5 9|k:0 4 8", "k:0 2 6 10|k:0 5 8", "k:0 4 7 10|k:0 3 8",
            "k:0 3 6 8|k:0 4 7 11", "k:0 2 5 8 11|k:0 2 6 9"),
           hands="l:2 8;x:4 6|l:3 9;x:4 8", grouping=(3,3), phrase_bars=2)

    family("Memphis soul", "soul", "4/4", 98, "unhurried backbeat",
           "Leave space around a solid two-and-four snare and keep the kicks compact.",
           (("even closed hats", "h:0 2 4 6 8 10 12 14"),
            ("closed hats on the offbeats", "h:2 6 10 14"),
            ("a sparser ride", "r:0 4 6 8 12 14"),
            ("an open lift before the repeat", "h:0 2 4 6 8 10 12;o:14")),
           ("k:0 3 9 14", "k:0 6 7 11", "k:0 4 10 14", "k:0 2 6 11", "k:0 7 8 10"),
           hands="s:4 12", lag=2)

    family("gospel soul pocket", "soul", "4/4", 108, "backbeat with quiet sixteenths",
           "Keep the two-and-four backbeat steady as ghost notes answer it.",
           (("closed hats in quick pairs", "h:0 1 4 5 8 9 12 13"),
            ("a shaker with quick pairs and space", "z:0 1 4 6 8 9 12 14"),
            ("a steady ride with two extra notes", "r:0 2 4 5 6 8 10 12 13 14"),
            ("open hat lifting after three", "h:0 2 4 6 8 12 14;o:10")),
           ("k:0 3 8 10;g:5 11", "k:0 2 7 11 15;g:3 13", "k:0 6 9 14;g:2 7 10",
            "k:0 3 5 8 13;g:6 11", "k:0 2 6 10 13;g:1 9 15"), hands="s:4 12")

    family("rim-led disco", "disco", "4/4", 117, "four-on-the-floor with rim clicks",
           "Keep the kick on every quarter and put a rim click on two and four.",
           (("open hats on every offbeat", "o:2 6 10 14"),
            ("closed-onbeat and open-offbeat hats", "h:0 4 8 12;o:2 6 10 14"),
            ("eighth-note shaker", "z:0 2 4 6 8 10 12 14"),
            ("short hat doubles before each open lift", "h:0 1 4 5 8 9 12 13;o:2 6 10 14")),
           ("k:0 4 8 12;g:3 9", "k:0 4 8 12;g:5 15", "k:0 4 8 12;g:2 11",
            "k:0 4 8 12;g:6 13", "k:0 4 8 12;g:7 10"), hands="x:4 12")

    family("disco octave pocket", "disco", "4/4", 123, "lifting four-on-the-floor",
           "Set a four-quarter bass drum under two-and-four snare, with a repeating tom answer.",
           (("a crisp open hat on all four offbeats", "o:2 6 10 14"),
            ("an articulated close-open hat line", "h:0 4 8 12;o:2 6 10 14"),
            ("a ride on all the eighths", "r:0 2 4 6 8 10 12 14"),
            ("a ride on the offbeats", "r:2 6 10 14")),
           ("k:0 4 8 12;m:2 10", "k:0 4 8 12;l:5 14", "k:0 4 8 12;t:2 6 10",
            "k:0 4 8 12;m:3 11", "k:0 4 8 12;t:6 11 14"), hands="s:4 12")

    family("steppers reggae", "reggae", "4/4", 86, "steady quarters and one-drop hands",
           "Use bass drum on all four beats, with rim on three and an offbeat top line.",
           (("only offbeat closed hats", "h:2 6 10 14"),
            ("open hats on the later two offbeats", "h:2 6;o:10 14"),
            ("a soft offbeat shaker", "z:2 6 10 14"),
            ("an offbeat ride with a short pickup", "r:2 6 10 14 15")),
           ("k:0 4 8 12;g:3 11", "k:0 4 8 12;m:2 10", "k:0 4 8 12;g:5 13",
            "k:0 4 8 12;l:6 14", "k:0 4 8 12;g:7 14"), hands="x:8")

    family("dubby one-drop reggae", "reggae", "4/4", 74, "sparse rim and breathing room",
           "Leave the first quarter free of kick, with the bass drum and rim together on three.",
           (("offbeat hats with one gap", "h:2 10 14"),
            ("long hat lifts after one and three", "o:2 10;h:14"),
            ("a very light ride offbeat", "r:2 6 10 14"),
            ("a shaker that leaves space near three", "z:2 6 14")),
           ("k:8;g:4 12", "k:8 15;g:6", "k:8 11;g:3 13",
            "k:8;m:3 12", "k:8 14;t:2 13"), hands="x:8")

    family("boom-bap hip-hop", "hip_hop", "4/4", 94, "late two-and-four with staggered hats",
           "Use quieter kicks around the two-and-four backbeat and let the snares sit late.",
           (("gapped offbeat hats", "h:2 6 10 14"),
            ("tight sixteenth pairs", "h:0 1 4 5 8 9 12 13"),
            ("eighth-note hats with a few extra taps", "h:0 2 3 4 6 8 10 11 12 14"),
            ("closed hats and an open seam accent", "h:0 2 4 6 8 10 12;o:14")),
           ("k:0 2 7 10;g:11", "k:0 3 8 14;g:6", "k:0 5 9 15;g:2 11",
            "k:0 6 8 11;g:7", "k:0 1 6 10 15;g:3"), hands="s:4 12", lag=4, soft_kicks=True)

    family("half-time hip-hop", "hip_hop", "4/4", 136, "half-time backbeat and broken sixteenths",
           "Put the main snare on three and leave space around the staggered subdivision.",
           (("sparse closed hats", "h:0 2 4 7 8 12 14"),
            ("closed hat pairs with a final open lift", "h:0 1 4 5 8 9 12 13;o:14"),
            ("a light eighth-note hat pattern", "h:0 2 4 6 8 10 12 14"),
            ("an offbeat hat pulse", "h:2 6 10 14")),
           ("k:0 3 6 12;g:10", "k:0 5 9 15;g:6", "k:0 2 7 14;g:5 11",
            "k:0 4 6 10 13;g:3", "k:0 3 7 11 15;g:13"), hands="s:8", lag=3)

    family("lo-fi hip-hop", "hip_hop", "4/4", 79, "soft timing and open space",
           "Let the kick and a late snare outline the bar with quiet space between them.",
           (("quarter-note hats with one quiet offbeat", "h:0 4 8 12 14"),
            ("spaced eighth-note hats with two taps near the seam", "h:0 2 4 8 12 14 15"),
            ("short closed hats with an open offbeat", "h:0 2 4 8 12 14;o:10"),
            ("close hat strokes just ahead of the final beat", "h:0 4 7 8 11 12 14")),
           ("k:0 6 10;g:3", "k:0 3 8 11;g:14", "k:0 2 7 10;g:5",
            "k:0 5 8 13;g:7", "k:0 3 6 9 14;g:2 11"), hands="s:4 12", lag=5, light_offbeat=48)

    family("five-four fusion", "fusion", "5/4", 128, "two plus three with a repeated bass figure",
           "Keep the bass drum at both group starts and answer with rim and floor tom.",
           (("quarter-note ride", "r:0 4 8 12 16"),
            ("even eighth-note ride", "r:0 2 4 6 8 10 12 14 16 18"),
            ("a bell landing on the group starts", "c:0 3 6 8 11 14 16 18"),
            ("short closed hats with two open accents", "h:0 2 4 8 10 12 16;o:6 18")),
           ("k:0 5 8 14 18", "k:0 3 8 12 16", "k:0 6 8 11 19", "k:0 4 8 10 15", "k:0 2 6 8 13 18"),
           hands="x:4 12 16;l:6 14", grouping=(2,3))

    family("two-three-two fusion", "fusion", "7/8", 119, "alternating short-long-short",
           "Keep the kick on the three group starts and repeat the tom as a mid-bar answer.",
           (("an eighth-note ride", "r:0 2 4 6 8 10 12"),
            ("a ride skipping inside the longer group", "r:0 2 4 7 8 10 12"),
            ("closed hats with an open accent late in the long group", "h:0 2 4 6 10 12;o:8"),
            ("a bell with short replies", "c:0 3 4 8 10 13")),
           ("k:0 4 10", "k:0 2 4 8 10", "k:0 3 4 7 10 12",
            "k:0 4 6 9 10 13", "k:0 4 5 8 10 12"), hands="s:2 8 12;l:6", grouping=(2,3,2))

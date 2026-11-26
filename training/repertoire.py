"""Original kit arrangements informed by the lessons linked in the README."""


def add_repertoire(family):
    family("sixteenth-note ghost funk", "funk", "4/4", 80, "straight sixteenths",
           "Play quiet snare ghost notes between the firm backbeats.",
           (("continuous sixteenths on closed hi-hat", "h:0* 1 2 3 4* 5 6 7 8* 9 10 11 12* 13 14 15"),
            ("sixteenths on hi-hat with two open lifts", "h:0 1 2 3 4 5 6 8 9 10 11 12 13 14;o:7 15")),
           ("k:0 3 8 10", "k:0 5 8 14", "k:0 6 9 11 15", "k:0 2 7 8 11", "k:0 3 5 9 14"),
           hands="s:4 12;g:2 7 11 13", variants=2)

    # RLRR LRLL twice, splitting the stickings across two kit voices.
    family("single paradiddle funk", "funk", "4/4", 68, "slow rudimental groove",
           "Repeat RLRR LRLL and accent the start of each half-note pulse.",
           (("the right hand on closed hats", "h:0* 2 3 5 8* 10 11 13"),
            ("the right hand on the ride", "r:0* 2 3 5 8* 10 11 13")),
           ("k:0 8", "k:0 6 8 11", "k:0 5 8 14", "k:0 3 7 8 13", "k:0 2 6 8 12"),
           hands="g:1 4 6 7 9 12 14 15", variants=2)

    family("Purdie shuffle", "blues", "4/4", 82, "triplet half time",
           "Put the backbeat on beat three, ghost the middle triplets, and let the kick answer across two bars.",
           (("long-short closed hats", "h:0 2 3 5 6 8 9 11"),
            ("long-short hats with a late open note", "h:0 2 3 5 6 8 9|h:0 2 3 5 6 8 9;o:11")),
           ("k:0 5 8|k:0 3 8 11", "k:0 2 5|k:0 5 8 10", "k:0 5 9|k:0 2 8",
            "k:0 5 8 11|k:0 2 7 9", "k:0 3 8|k:0 5 6 11"),
           hands="s:6;g:1 4 7 10", unit=32, phrase_bars=2, variants=2, artists=("Bernard Purdie",))

    family("Mozambique latin kit", "latin", "4/4", 101, "bell and tom conversation",
           "Repeat a bell timeline above syncopated kicks and snare-to-tom replies.",
           (("a two-bar cowbell", "c:0 3 6 8 10 13|c:0 2 5 8 11 14"),
            ("the bell phrase on ride", "r:0 3 6 8 10 13|r:0 2 5 8 11 14")),
           ("k:0 6 10|k:0 7 14", "k:0 3 7 12|k:0 6 10 15", "k:0 5 11|k:0 3 8 14",
            "k:0 7 10 14|k:0 5 11", "k:0 4 9 15|k:0 7 10"),
           hands="p:4 12;s:6 14;l:2 11", phrase_bars=2, variants=2)

    family("cáscara latin kit", "latin", "4/4", 110, "two-bar shell phrasing",
           "Play a two-bar cáscara-style timeline and leave room for a tumbao-style kick pickup.",
           (("shell-inspired bell time", "c:0 3 4 7 8 10 13|c:0 2 5 6 8 11 12 14"),
            ("the shell-inspired phrase on ride", "r:0 3 4 7 8 10 13|r:0 2 5 6 8 11 12 14")),
           ("k:6 14", "k:3 6 14|k:6 14", "k:6 10 14|k:2 6 14",
            "k:6 14|k:6 11 14", "k:2 6 14|k:6 10 14"),
           hands="x:6 12|x:0 6 12", phrase_bars=2, variants=2)

    family("baião latin kit", "latin", "4/4", 108, "zabumba-inspired bass pair",
           "Repeat a deep pickup-and-downbeat bass figure against a light triangle-like pulse.",
           (("light eighths on the ride", "r:0 2 4 6 8 10 12 14"),
            ("a light closed-hat pulse", "h:0 2 4 6 8 10 12 14")),
           ("k:0 7 8 15;x:4 12", "k:0 7 8 15;x:3 12", "k:0 5 7 8 15;x:4 11",
            "k:0 7 8 13 15;x:4 12", "k:0 7 8 15;x:4 10 12"),
           variants=2)

    family("choro latin kit", "latin", "4/4", 107, "light pandeiro-inspired motion",
           "Maintain the light forward pulse and answer the bass with small rim accents.",
           (("soft shaker sixteenths", "z:0 1 2* 3 4 5 6* 7 8 9 10* 11 12 13 14* 15"),
            ("broken shaker sixteenths", "z:0 1 2* 4 5 6* 7 8 9 10* 12 13 14* 15")),
           ("k:0 6 8 14;x:3 11", "k:0 6 8 14;x:5 13", "k:0 3 8 11;x:2 6 14",
            "k:0 6 8 14;x:3 7 11 15", "k:0 5 8 13;x:2 7 10 15"),
           variants=2)

    family("soca carnival latin kit", "latin", "4/4", 123, "Caribbean dance pulse",
           "Keep the quarter-note kick under syncopated snare calls and a light offbeat top.",
           (("offbeat hats with lifts", "h:2 10;o:6 14"), ("a light offbeat bell", "c:2 6 10 14")),
           ("k:0 4 8 12;s:3 6 11 14", "k:0 4 8 12;s:3 7 11 15",
            "k:0 4 8 12;s:2 6 11 15", "k:0 4 8 12;s:3 6 10 15",
            "k:0 4 8 12;s:2 7 10 14"), variants=2)

    family("boogaloo latin kit", "latin", "4/4", 105, "Latin soul backbeat",
           "Tie a dry two-and-four backbeat to a syncopated bell and bass part.",
           (("a broken bell timeline", "c:0 3 6 8 11 14"), ("a syncopated closed hat", "h:0 3 6 8 11 14")),
           ("k:0 6 8 13", "k:0 3 7 10 14", "k:0 6 9 12 15",
            "k:0 2 7 8 14", "k:0 5 8 11 14"),
           hands="s:4 12;x:7 15", variants=2)

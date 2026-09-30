# usergen

Edits usernames you give it into new ones, and makes new ones from scratch,
using two kinds of knowledge:

- **Own knowledge**, which you teach it by training on your own lists of
  usernames.
- **Claude knowledge**, built in: how real email-style usernames are made,
  the kind people end up with once `john.smith` is long taken. It is always
  there, and never mixes with yours.

Every edit looks like a real username that is likely still free: lowercase,
with no decorations like `its`, `lil`, `xX…Xx`, `playz` or `.io`, and never
as plain as `johnkevin` or `emma2008`. That holds whichever knowledge you
edit with (see [Unique email style](#unique-email-style)).

It handles millions of usernames in seconds by using every CPU core.

```sh
go build -o usergen .
./usergen
```

```
  1 - Training (your own knowledge)
  2 - Edit
  3 - Generate
  4 - Knowledge info
  5 - Settings
  6 - Clear own knowledge
  0 - Exit
> 2
Shortest length (3-24) [3]: 8
Longest length (8-24) [24]: 14
Which knowledge?
  1 - Own
  2 - Claude
  3 - Both
Choose 1-3 [3]: 2
Edits per username (1-1000, or max) [10]: 4
File: mine.txt
Editing  [██████████████████████████████] 100%
Saved 20 edited usernames (8-14 characters) to edited_20.txt in 54ms
```

`mine.txt` held `maria.garcia92`, `tobiaslindqvist`, `itsnoah2004`,
`xXShadowXx`, `kevin.hollis` and `dariwex07`; `edited_20.txt` holds
`kayla.garcia92`, `maria.garcia76`, `tobiasgentry`, `thanhlindqvist`,
`philip2004`, `abril2004`, `kevin.goddard`, `ayesha.hollis`, `dariwex96`,
`dariwex1608` and so on. `itsnoah2004` lost its `its` (and `noah2004` alone is
too plain to keep); `xXShadowXx` became `shadow`, too short for 8-14.

1. **Training** learns from a file of usernames (one per line) into your own
   knowledge, saved in `usergen.model`. Each run adds to it rather than
   starting over.
2. **Edit** asks, in order: the shortest and longest length; which
   knowledge (own, Claude's, or both); and how many edits per username, as a
   number or `max`. Then it asks for the file, edits every username in it,
   and saves the edits to `edited_<count>.txt`. Pressing Enter keeps the
   answer in brackets, which is always your last one.
3. **Generate** asks which knowledge, then makes brand-new usernames and
   saves them to `generated_<count>.txt`, showing the first few.
4. **Knowledge info** shows what both knowledges contain.
5. **Settings** changes how the other options work (below).
6. **Clear own knowledge** deletes your own knowledge after you type `yes`.
   Claude's built-in knowledge stays.

`max` makes every edit the knowledge allows: every learned word or number
swapped into every part of the name, plus whatever the model can write. That
is often hundreds per username, so a large file can produce a very large
result.

## Claude knowledge

Rather than a list of anyone's real accounts, it is the patterns that real,
still-free email-style usernames follow, measured on a sample of them and
keeping every shape that made up at least 5% of it:

| Share | Shape | Examples |
|------:|-------|----------|
| 26% | nicknames and blends of names, mostly with a number | `dariwex2006`, `brandy2011`, `tobiasski`, `kristofer91` |
| 18% | a first name and a less common surname, run together, sometimes with a number, a digit or an initial between | `tobiaslindqvist`, `ivan7okafor`, `saragwinslow08` |
| 18% | a first name and a less common surname with a dot, `_` or `-` | `maya.thorne`, `leo_quigley`, `nina.ashby` |
| 10% | a first name and a 3-4 digit number or a date | `rosalind4318`, `tariq_0912` |
| 10% | a first name and initials or a short surname | `yusuf.tk`, `greta.hol`, `omar.a.j` |
| 9% | two first names | `amir.helena` |
| 9% | a first name and a common surname, now and then with a number | `ines.carter`, `leo_hayes27` |

Almost all are lowercase, half end in a number (a birth year, 2-4 digits, a
date, or a favourite like `123` or `007`), and half have no separator.
Prefixes and suffixes (`its`, `lil`, `mr`, `xX…Xx`, `playz`, `boi`, `mom`,
`.io`) were under 5% of the sample and are left out, and so are brand and
game names and crude words.

The names come from about 1,250 first names and nicknames (more often ones
common in English- and Spanish-speaking countries), 150 common and 1,380
less common last names from many countries. The knowledge is learned from
200,000 usernames built this way, in under a second when first needed.

```
> 4
Own knowledge
  Nothing learned yet. Choose 1 - Training to teach it.
Claude knowledge (built in)
  Email-style  198,419 usernames, 8,714 words; most common: hudson, russell, kennedy, grant, james, thomas
```

## Unique email style

Plain usernames are almost all taken, and decorated ones look like gamer
tags. So while **Unique email style** is on (it is by default), everything
Edit and Generate make, from either knowledge:

- is lowercase;
- has no decorations: they are taken off each username before it is edited
  (`its.Mike_99` → `mike_99`, `xXShadowXx` → `shadow`, `noahplayz` → `noah`),
  and edits that would have them are dropped;
- is not plain: not a name on its own (`stefan`), not a very common name
  with a birth year or up to two digits (`emma2008`, `john92`), and not two
  very common names or an initial and one (`johnkevin`, `john.smith`,
  `jsmith`).

Uncommon names pass (`kevin.hollis`, `jason4471`), so your own knowledge
still edits in its own style, just without decorations or plain results.
Turn it off in Settings, or with `-unique=false`, to edit any style.

## Settings

Choose a number to change a setting; on/off settings flip straight away, and
Knowledge cycles through own, claude and both. Settings are saved to
`usergen.settings.json` and remembered next time; Edit's and Generate's
answers update them too.

```
Settings (saved in usergen.settings.json)
  Training
    1  Context length for a new model (1-8)           3
    2  Save rejected usernames to a file              off
    3  Dry run: learn nothing, just report            off
  Edit
    4  Edits per username (1-1000, or max)            10
    5  Shortest length (3-24)                         3
    6  Longest length (3-24)                          24
  Generate
    7  How many usernames                             10
    8  Shortest length (3-24)                         4
    9  Longest length (3-24)                          16
   10  Creativity (1 = as learned, higher = wilder)   1
   11  Context length (0 = the model's)               0
  Edit and Generate
   12  Knowledge (own, claude or both)                both
   13  Unique email style (no plain or decorated)     on
   14  Allow usernames the knowledge learned from     off
   15  Seed (0 = random each time)                    0
  Own knowledge
   16  File                                           usergen.model
    0  Back
```

- **Context length** is how many previous characters the model looks at.
  It is fixed when your own knowledge is created, so a change applies after
  Clear own knowledge. For Generate, a lower value than the model's gives wilder names.
- **Save rejected usernames** writes each garbage username skipped during
  training, with the reason, to `rejected_<count>.txt`.
- **Dry run** shows what training would learn without saving anything. While
  it is on, the menu says so next to Training.
- **Seed** makes Edit and Generate repeatable: the same seed, model and input
  always give the same output.

## Speed

Measured on a 4-core machine; more cores are faster.

| Task | Input | Time |
|------|-------|------|
| Training | 2,000,000 lines | 2.0 s |
| Edit, own knowledge | 1,000,000 usernames → 5.5M edits | 9.7 s |
| Edit, Claude knowledge | 1,000,000 usernames → 5.2M edits | 8.3 s |
| Edit, both | 1,000,000 usernames → 5.7M edits | 10.6 s |

With Unique email style off, editing takes about 30% less time.

Training checks and deduplicates names in file order, then counts patterns on
every core. Editing splits the input across every core and writes results in
input order. For a given seed and knowledge, the output is identical on
every run.

## How it edits

Each username is split into words, numbers and separators
(`xXSniperXx` → `x` `X` `Sniper` `Xx`), then edited four ways:

| Edit | Example |
|------|---------|
| swap a word or number for one learned from other usernames | `maria.garcia92` → `kayla.garcia92` |
| keep the leading words and let the model write the rest | `kevin.hollis` → `kevin.lorenzo` |
| let the model add up to 4 characters to the end | `maria.garcia92` → `maria.garcia928` |
| drop a number | `samforest2004` → `samforest` |

When the model writes characters, it only continues from patterns it has
actually seen, and never guesses from less context. Every new word it writes
must be made of words it learned: a known word, several run together
(`juanbaker`), or one after an initial (`jsmith`). Fragments like `leepy`
are rejected, which Generate follows too. Every edit must pass the garbage
filter below. So a username unlike anything the model learned gets few or no
edits rather than junk; train on usernames in the same style as the ones you
edit.

Words are learned only from usernames that show where words start and end
(`ShadowWolf`, `shadow_wolf`), and letters split by a leetspeak digit
(`Fr0zen`) aren't taken for words. The model also learns where each word
goes: whether it starts names (`dark`, `john`) or comes later (`wolf`,
`smith`). Swaps put words only where they belong, so `john.smith` becomes
`john.washington` or `samir.smith`, never `john.priya` or `WolfFox`.
Wrappers like `xX…Xx`, `The` and `its` stay in place and are never swapped
in (with Unique email style on, they are taken off instead).

Only words the knowledge learned are swapped, each for one like it: a name
for a name, initials for initials (`yusuf.tk` → `yusuf.jb`). Words it doesn't
know, like the made-up `dariwex` in `dariwex07`, are kept, since they are what
make a name its own; the number changes instead (`dariwex96`).

Lowercase names that run words together are split where a learned word
ends (`tobiaslindqvist` → `tobias` `lindqvist`), so each part can be swapped
and the edit stays joined: `tobiasgentry`, `thanhlindqvist`. The first part
must be a word that starts names, so names the knowledge doesn't know, like
`warren`, are never cut into fragments. Numbers are swapped for numbers of
the same shape: a year for a year, two digits for two (`dariwex07` →
`dariwex96`).

The saved file has one username per line with no duplicates. It leaves out
the usernames you gave it and names any knowledge in use learned from. It is never
overwritten: if `edited_30.txt` exists, the next one is `edited_30_2.txt`.

## Garbage filtering

Garbage is never learned and never produced. A name is rejected if it:

| Rule | Rejected examples |
|------|-------------------|
| is outside 3–24 characters | `x`, `ThisIsAReallyLongUsernameThatNobodyUses` |
| has anything but letters, digits, `_`, `-`, `.` | `john@gmail.com`, `[deleted]`, `cool😎guy`, `has space` |
| is a placeholder or default account name | `deleted`, `AutoModerator`, `user_83921`, `Player7` |
| is less than half letters | `12345678`, `Johnny99887766` |
| looks like a hex ID | `8f3a9c2b1e` |
| contains a keyboard row or alphabet run of 5 | `asdfghjkl`, `qwerty123`, `abcdefg` |
| repeats a character 4+ times or a 2–3 character chunk 3+ times | `aaaaaa`, `NooooOob`, `hahaha`, `lolololol` |
| has 5 or more digits in a row | `xX_Sniper_12345` |
| has 7+ consonants in a row, or 6+ Latin letters with no vowel | `xkcdfjgh`, `x_k_c_d_f_g` |

Letters from any script are allowed, and real names such as `FirstStrike`,
`CyberPunk2077`, `L33tH4x0r`, `BananaSplit` and `xX_Reaper_Xx` pass.

## Commands

Everything the menu does is also a command, for scripts. Commands take
flags and ignore the menu's settings file, so scripts behave the same
wherever they run. Progress bars are drawn only when output goes to a
terminal.

```sh
./usergen train names.txt more.txt          # own knowledge; or - for stdin
./usergen train -dry-run -show-rejected names.txt
./usergen edit mine.txt                     # both knowledges, 10 edits each
./usergen edit -knowledge claude -edits max mine.txt
./usergen edit -min 8 -max 12 -edits 5 -seed 42 mine.txt
./usergen edit -unique=false mine.txt       # any style, decorations kept
./usergen generate -n 20 -knowledge claude  # brand-new usernames from scratch
./usergen stats                             # your own knowledge
```

| Flag | Command | Meaning |
|------|---------|---------|
| `-model` | all | your own knowledge's file (default `usergen.model`) |
| `-order` | train | characters of context for a **new** model (default 3) |
| `-show-rejected` | train | print each garbage username skipped and why |
| `-dry-run` | train | report what would be learned without saving |
| `-knowledge` | edit, generate | `own`, `claude` or `both` (default `both`; without own knowledge, both uses Claude's) |
| `-edits` | edit | most edits per username, or `max` (default 10) |
| `-min`, `-max` | edit | shortest and longest edit in characters (default 3-24) |
| `-seed` | edit, generate | fixed seed for repeatable output (0 = random) |
| `-allow-known` | edit, generate | allow names the knowledge learned from |
| `-unique` | edit, generate | only lowercase email-style names, without decorations or plain names (default `true`) |
| `-n`, `-min`, `-max`, `-temp`, `-order` | generate | count, shortest and longest length, creativity, context |

## Own knowledge file

`usergen.model` is binary so millions of usernames save and load quickly. It
holds the character patterns, the learned words (with where they go in names)
and numbers, and a 64-bit fingerprint of every learned name (for skipping
duplicates), not the names themselves. Saves are atomic, so an interrupted
run never corrupts it.

Models from earlier versions still load. A `usergen.json` model is converted,
dropping any names the garbage filter rejects. A model saved before word
positions were recorded works without them until it's retrained: choose
6 - Clear own knowledge and train again to get position-aware swaps.

## Tests

```sh
go test ./...
go test ./edit -bench .
```

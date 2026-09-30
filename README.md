# usergen

Edits usernames you give it into new ones, and makes new ones from scratch,
using two kinds of knowledge:

- **Own knowledge**, which you teach it by training on your own lists of
  usernames.
- **Claude knowledge**, built in: how real email-style and social/gaming
  usernames are made. It is always there, and never mixes with yours.

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
Shortest length (3-24) [3]: 6
Longest length (6-24) [24]: 14
Which knowledge?
  1 - Own
  2 - Claude
  3 - Both
Choose 1-3 [3]: 2
Edits per username (1-1000, or max) [10]: 4
File: mine.txt
Editing  [██████████████████████████████] 100%
Saved 22 edited usernames (6-14 characters) to edited_22.txt in 33ms
```

`mine.txt` held `john.smith`, `maria.garcia92`, `jsmith`, `SilentWolf`,
`itsmike` and `xXShadowXx`; `edited_22.txt` holds `ford.smith`,
`john.smith1990`, `ramos.garcia92`, `jsmith96`, `SilentSteel`, `ArtistWolf`,
`itsmike_art`, `xXShadowIron` and so on.

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

Rather than a list of anyone's real accounts, it is the patterns real
usernames follow, combined with about 200 common first names and 200 last
names from many countries and a few hundred everyday words. It is learned
from 100,000 usernames of each style, built in a fraction of a second when
first needed:

| Style | Patterns | Examples |
|-------|----------|----------|
| Email | first.last, firstlast, flast, f.last, firstl, last.first, first.m.last, and those with birth years or numbers | `john.smith`, `jsmith`, `neha.walker72`, `lnguyen1982`, `gpark007` |
| Normal | AdjectiveNoun, NounNoun, name + number, its/real/the + name, name + hobby, xX…Xx, TheNoun, channel names | `SilentWolf`, `itsmike`, `sarahbakes`, `mike_92`, `xXShadowXx`, `HyperHD` |

You don't choose a style: each username is edited with whichever knowledge
knows its words best, so `john.smith` gets email-style edits and
`SilentWolf` gets gaming ones. With **Both**, your own knowledge joins in on
the same terms.

```
> 4
Own knowledge
  Nothing learned yet. Choose 1 - Training to teach it.
Claude knowledge (built in)
  Email-style  78,301 usernames, 401 words; most common: carter, taylor, morgan, thomas, gordon, becker
  Normal       71,200 usernames, 505 words; most common: xx, the, pixel, pirate, orbit, rain
```

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
   13  Allow usernames the knowledge learned from     off
   14  Seed (0 = random each time)                    0
  Own knowledge
   15  File                                           usergen.model
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
| Training | 2,000,000 lines | 1.5 s |
| Edit, own knowledge | 1,000,000 usernames → 6.0M edits | 6.0 s |
| Edit, Claude knowledge | 1,000,000 usernames → 4.9M edits | 5.5 s |
| Edit, both | 1,000,000 usernames → 6.1M edits | 7.1 s |

Training checks and deduplicates names in file order, then counts patterns on
every core. Editing splits the input across every core and writes results in
input order. For a given seed and knowledge, the output is identical on
every run.

## How it edits

Each username is split into words, numbers and separators
(`xXSniperXx` → `x` `X` `Sniper` `Xx`), then edited four ways:

| Edit | Example |
|------|---------|
| swap a word or number for one learned from other usernames | `ShadowFox` → `SolarFox` |
| keep the leading words and let the model write the rest | `ShadowFox` → `ShadowWalker` |
| let the model add up to 4 characters to the end | `DarkKnight_7` → `DarkKnight_77` |
| drop a number | `MysticPanda99` → `MysticPanda` |

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
(`Fr0zen`) aren't taken for words.

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
| `-n`, `-min`, `-max`, `-temp`, `-order` | generate | count, shortest and longest length, creativity, context |

## Own knowledge file

`usergen.model` is binary so millions of usernames save and load quickly. It
holds the character patterns, the learned words and numbers, and a 64-bit
fingerprint of every learned name (for skipping duplicates), not the names
themselves. Saves are atomic, so an interrupted run never corrupts it. Models
from earlier versions (`usergen.json`) are converted automatically when
loaded, dropping any names the garbage filter rejects.

## Tests

```sh
go test ./...
go test ./edit -bench .
```

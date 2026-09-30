# usergen

Learns how usernames are built from example lists, then edits usernames you
give it into new ones using what it learned. It handles millions of usernames
in seconds by using every CPU core.

```sh
go build -o usergen .
./usergen
```

```
  1 - Training
  2 - Edit
  3 - Generate
  4 - Model info
  5 - Settings
  6 - Clear all knowledge
  0 - Exit
> 1
File: names.txt
Training [██████████████████████████████] 100%
Learned 1,001,559 new usernames in 1.5s (skipped 898,193 already known, 100,248 garbage)

> 2
Shortest length (3-24) [3]: 8
Longest length (8-24) [24]: 12
File: mine.txt
Editing  [██████████████████████████████] 100%
Saved 2,784,132 edited usernames (8-12 characters) to edited_2784132.txt in 5.6s

> 4
  Model file   usergen.model (8.7 MB)
  Usernames    1,001,559 learned, 6-24 characters (average 12.9)
  Patterns     105,964, using 3 characters of context
  Words        228; most common: xx, man, code, owl, star, shadow, starter, sneaky, titan, ghostly
  Numbers      106; most common: 69, 88, 99, 53, 22, 26, 74, 77, 27, 72

> 6
This erases everything learned (1,001,559 usernames) and cannot be undone. Type yes to confirm: yes
All knowledge cleared.
```

1. **Training** learns from a file of usernames (one per line) and saves what
   it learned to `usergen.model`. Each run adds to the model rather than
   starting over.
2. **Edit** asks for the shortest and longest length you want (Enter keeps
   the last ones used), then turns every username in a file into up to 10
   new ones of that length and saves them to `edited_<count>.txt`.
3. **Generate** makes brand-new usernames from scratch and saves them to
   `generated_<count>.txt`, showing the first few.
4. **Model info** shows what the model has learned: how many usernames,
   patterns, words and numbers, and the most common ones.
5. **Settings** changes how the other options work (below).
6. **Clear all knowledge** deletes the model after you type `yes`, so the
   next training starts from nothing.

## Settings

Choose a number to change a setting; on/off settings flip straight away.
Settings are saved to `usergen.settings.json` and remembered next time.

```
Settings (saved in usergen.settings.json)
  Training
    1  Context length for a new model (1-8)           3
    2  Save rejected usernames to a file              off
    3  Dry run: learn nothing, just report            off
  Edit
    4  Edits per username                             10
    5  Shortest length (3-24)                         3
    6  Longest length (3-24)                          24
  Generate
    7  How many usernames                             10
    8  Shortest length (3-24)                         4
    9  Longest length (3-24)                          16
   10  Creativity (1 = as learned, higher = wilder)   1
   11  Context length (0 = the model's)               0
  Edit and Generate
   12  Allow usernames the model learned from         off
   13  Seed (0 = random each time)                    0
  Model
   14  Model file                                     usergen.model
    0  Back
```

- **Context length** is how many previous characters the model looks at.
  It is fixed when a model is created, so a change applies after Clear all
  knowledge. For Generate, a lower value than the model's gives wilder names.
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
| Edit | 1,000,000 usernames → 6.2M edits | 5 s |
| Edit, 8-12 characters | 1,000,000 usernames → 2.8M edits | 5.6 s |

Training checks and deduplicates names in file order, then counts patterns on
every core. Editing splits the input across every core and writes results in
input order. For a given seed and model, the output is identical on every run.

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
actually seen, and never guesses from less context. Every edit must pass the
garbage filter below. So a username unlike anything the model learned gets
few or no edits rather than junk; train on usernames in the same style as
the ones you edit.

The saved file has one username per line with no duplicates. It leaves out
the usernames you gave it and names the model learned from. It is never
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
./usergen train names.txt more.txt          # or - for stdin
./usergen train -dry-run -show-rejected names.txt
./usergen edit mine.txt
./usergen edit -min 8 -max 12 -edits 5 -seed 42 mine.txt
./usergen generate -n 20                    # brand-new usernames from scratch
./usergen stats
```

| Flag | Command | Meaning |
|------|---------|---------|
| `-model` | all | model file (default `usergen.model`) |
| `-order` | train | characters of context for a **new** model (default 3) |
| `-show-rejected` | train | print each garbage username skipped and why |
| `-dry-run` | train | report what would be learned without saving |
| `-edits` | edit | most edits per username (default 10) |
| `-min`, `-max` | edit | shortest and longest edit in characters (default 3-24) |
| `-seed` | edit, generate | fixed seed for repeatable output (0 = random) |
| `-allow-known` | edit, generate | allow names the model learned from |
| `-n`, `-min`, `-max`, `-temp`, `-order` | generate | count, shortest and longest length, creativity, context |

## Model file

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

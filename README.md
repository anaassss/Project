# usergen

Learns how usernames are built from example lists and generates new ones.
What it learns is saved to a model file (`usergen.json` by default), and each
training run adds to that model rather than starting over.

## How it learns

usergen is a character-level Markov model. For every username it counts which
character follows each run of up to *order* previous characters (default 3),
including how names start and end. Generating walks those counts at random, so
the output follows the patterns in the training data: `Shadow` + `Wolf`,
trailing `_7` or `Xx`, CamelCase, and so on.

Names are deduplicated case-insensitively, so training on the same list twice
does not skew the model. Generated names that match a training name are
discarded unless you pass `-allow-known`.

## Usage

```sh
go build -o usergen .

# Learn from a list (one username per line; blank lines and # comments ignored)
./usergen train examples/usernames.txt

# Keep teaching it later; the saved model is loaded and extended
./usergen train more_names.txt
cat scraped.txt | ./usergen train -

# Generate
./usergen generate -n 20
./usergen generate -n 20 -order 2 -temp 1.3   # wilder
./usergen generate -n 20 -temp 0.7            # safer
./usergen generate -seed 42                   # repeatable output

# Inspect what it has learned
./usergen stats
```

| Flag | Command | Meaning |
|------|---------|---------|
| `-model` | all | model file to load/save (default `usergen.json`) |
| `-order` | train | context length for a **new** model; an existing model keeps its order |
| `-n` | generate | how many usernames |
| `-min`, `-max` | generate | length bounds (default 4–16) |
| `-temp` | generate | below 1 favours common patterns, above 1 rarer ones |
| `-order` | generate | use less context than the model has, for more novel names |
| `-seed` | generate | fixed seed for repeatable output (0 = random) |
| `-allow-known` | generate | allow names from the training data |

## Tuning

- **Output is mostly real names glued together:** normal for small training
  sets. Lower `-order` at generate time or raise `-temp`.
- **Output is gibberish:** raise `-order` when training (needs more data) or
  lower `-temp`.
- **"only found N new usernames":** the model can't produce enough novel names
  under your settings. Train on more names or relax the settings above.

## Model file

Plain JSON: the order, every learned name, and the transition counts. Saves are
atomic (written to a temp file, then renamed), so an interrupted run never
corrupts the model. Size grows roughly with the number of distinct character
patterns; the 155-name example produces about 30 KB.

## Tests

```sh
go test ./...
```

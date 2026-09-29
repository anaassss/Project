package markov

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestCheckRejectsGarbage(t *testing.T) {
	for name, reason := range map[string]string{
		"":                           "too short",
		"ab":                         "too short",
		"ThisUsernameIsWayTooLong42": "too long",
		"bad\xff":                    "not valid UTF-8",
		"john@example.com":           "disallowed character",
		"https://x.io":               "disallowed character",
		"[deleted]":                  "disallowed character",
		"has space":                  "disallowed character",
		"cool😎dude":                  "disallowed character",
		"deleted":                    "placeholder or auto-generated",
		"AutoModerator":              "placeholder or auto-generated",
		"user_48213":                 "placeholder or auto-generated",
		"Guest-7":                    "placeholder or auto-generated",
		"Player2":                    "placeholder or auto-generated",
		"12345678":                   "mostly digits or symbols",
		"8f3a9c2b":                   "looks like an ID",
		"a3f9c2e8b1d4":               "looks like an ID",
		"x1234_":                     "mostly digits or symbols",
		"__x__":                      "mostly digits or symbols",
		"Johnny123456":               "long run of digits",
		"aaaaaa":                     "repeated pattern",
		"NooooOob":                   "repeated pattern",
		"lolololol":                  "repeated pattern",
		"lololol":                    "repeated pattern",
		"xXxX":                       "repeated pattern",
		"HaHaHa_Joker":               "repeated pattern",
		"xyzxyzxyz":                  "repeated pattern",
		"asdfghjkl":                  "keyboard or alphabet sequence",
		"QwertyKing":                 "keyboard or alphabet sequence",
		"lkjhgfd":                    "keyboard or alphabet sequence",
		"abcdefg":                    "keyboard or alphabet sequence",
		"xkcdfjgh":                   "unpronounceable",
		"brtkmnz":                    "unpronounceable",
		"x_k_c_d_f_g":                "unpronounceable",
	} {
		err := Check(name)
		var ge *GarbageError
		if !errors.As(err, &ge) || !errors.Is(err, ErrGarbage) {
			t.Errorf("Check(%q) = %v, want %q", name, err, reason)
			continue
		}
		if ge.Reason != reason {
			t.Errorf("Check(%q) reason = %q, want %q", name, ge.Reason, reason)
		}
	}
}

func TestCheckKeepsRealUsernames(t *testing.T) {
	names := []string{
		"Ash", "FirstStrike", "xX_Reaper_Xx", "CyberPunk2077", "L33tH4x0r",
		"john.doe", "dark-knight", "Rhythm", "Zzz_Sleepy", "Ünïcødé_Ωmega",
		"Влад_Крутой", "星のカービィ", "007_bond", "Player_One", "UserFriendly",
		"deadbeef", "cafe42", "Face1t", "haha_king", "BananaSplit",
		"Mississippi", "Zzz", "lolol", "xXx_Slayer_xXx",
	}
	f, err := os.Open("../examples/usernames.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
			names = append(names, line)
		}
	}
	for _, name := range names {
		if err := Check(name); err != nil {
			t.Errorf("Check(%q) = %v, want nil", name, err)
		}
	}
}

func TestGenerateNeverReturnsGarbage(t *testing.T) {
	m := trained(t, 3)
	for _, name := range []string{"Zed4444Zed", "Bob_12_34", "Mxyzptlk"} {
		m.learn(name) // bypass Check to give the chain garbage-prone patterns
	}
	names, err := m.Generate(rng(), 300, GenerateOptions{MinLen: 1, MaxLen: 40, Temperature: 3, Order: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if err := Check(name); err != nil {
			t.Errorf("generated garbage %q: %v", name, err)
		}
	}
}

func TestCleanDropsGarbage(t *testing.T) {
	m := trained(t, 3)
	want := m.Contexts()
	for _, name := range []string{"asdfghjkl", "user_123"} {
		m.learn(name) // as a model trained before the garbage rules would have
	}
	if m.Contexts() == want {
		t.Fatal("test setup did not add any patterns")
	}

	dropped := m.Clean()
	if strings.Join(dropped, ",") != "asdfghjkl,user_123" {
		t.Errorf("Clean dropped %v", dropped)
	}
	if m.Contexts() != want || len(m.Names()) != len(training) || m.Knows("asdfghjkl") {
		t.Error("Clean did not restore the model trained on clean names only")
	}
	if dropped := m.Clean(); len(dropped) != 0 {
		t.Errorf("second Clean dropped %v", dropped)
	}
}

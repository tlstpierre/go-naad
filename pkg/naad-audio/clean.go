package naadaudio

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Alert text arrives in whatever shape the issuer typed it: CR LF line
// endings, trailing spaces, runs of blank lines, double spaces, the odd
// control character. CleanText normalizes it once; the output-specific
// helpers below then adapt it for a display (DisplayText) or for a speech
// synthesizer (SpeechText).

// CleanText normalizes alert text for any output medium:
//   - CR LF, CR, U+2028 and U+2029 become "\n"
//   - tabs become spaces; other control and format characters (including
//     BEL, so alert content can never ring a bell) are removed
//   - runs of spaces collapse to one, and each line is trimmed
//   - blank lines are removed
//
// The result is the non-empty lines of the text joined with "\n".
func CleanText(text string) string {
	text = strings.NewReplacer("\r\n", "\n", "\r", "\n", " ", "\n", " ", "\n", "\t", " ").Replace(text)
	text = strings.Map(func(r rune) rune {
		switch {
		case r == '\n':
			return r
		case r == utf8.RuneError, unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			return -1
		case unicode.IsSpace(r): // NBSP and friends
			return ' '
		}
		return r
	}, text)

	lines := strings.Split(text, "\n")
	out := lines[:0]
	for _, line := range lines {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// StripComments removes the "###"-delimited comment section that some
// issuers append to the description.
func StripComments(text string) string {
	text, _, _ = strings.Cut(text, "###")
	return text
}

// DisplayText prepares text for a display, sign or teleprinter: cleaned,
// one line per line, no blank lines.
func DisplayText(text string) string {
	return CleanText(text)
}

// SpeechText prepares text for a speech synthesizer: cleaned, lines joined
// into sentences (a line without closing punctuation gets a full stop so
// the voice pauses), repeated punctuation collapsed, and pronunciation
// replacements applied on word boundaries.
func SpeechText(text string, replacements []TTSReplacement) string {
	text = CleanText(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if r, _ := utf8.DecodeLastRuneInString(line); !strings.ContainsRune(".!?:;,", r) {
			lines[i] = line + "."
		}
	}
	text = strings.Join(lines, " ")
	for _, r := range replacements {
		text = replaceWords(text, r.From, r.To)
	}
	return collapsePunctuation(text)
}

// replaceWords replaces from with to wherever from is not part of a larger
// word: if from starts (ends) with a letter, the character before (after)
// the match must not be a letter. Digits are not letters, so "km" matches
// in "10km" but not "mm" in "community".
func replaceWords(text, from, to string) string {
	if from == "" {
		return text
	}
	first, _ := utf8.DecodeRuneInString(from)
	last, _ := utf8.DecodeLastRuneInString(from)
	var b strings.Builder
	for {
		i := strings.Index(text, from)
		if i < 0 {
			b.WriteString(text)
			return b.String()
		}
		end := i + len(from)
		before, _ := utf8.DecodeLastRuneInString(text[:i])
		after, _ := utf8.DecodeRuneInString(text[end:])
		ok := !(unicode.IsLetter(first) && i > 0 && unicode.IsLetter(before)) &&
			!(unicode.IsLetter(last) && end < len(text) && unicode.IsLetter(after))
		if ok {
			b.WriteString(text[:i])
			b.WriteString(to)
		} else {
			b.WriteString(text[:end])
		}
		text = text[end:]
	}
}

// collapsePunctuation tidies the joins produced by templates and
// replacements: repeated full stops (".." or ". .") become one (followed
// by a space if a word comes straight after), spaces
// before punctuation are removed, and runs of spaces collapse to one.
func collapsePunctuation(text string) string {
	out := make([]rune, 0, len(text))
	droppedStop := false
	for _, r := range text {
		if droppedStop && !unicode.IsSpace(r) && !strings.ContainsRune(".,;:!?", r) {
			out = append(out, ' ') // "required..This" -> "required. This"
		}
		droppedStop = false
		n := len(out)
		switch {
		case r == ' ':
			if n == 0 || out[n-1] == ' ' {
				continue
			}
		case strings.ContainsRune(".,;:!?", r):
			if n > 0 && out[n-1] == ' ' {
				out, n = out[:n-1], n-1
			}
			if r == '.' && n > 0 && out[n-1] == '.' {
				droppedStop = true
				continue
			}
		}
		out = append(out, r)
	}
	return strings.TrimSpace(string(out))
}

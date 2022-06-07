package internal

import (
	"bufio"
	"strings"

	"github.com/johnfercher/maroto/internal/fpdf"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/props"
)

// Text is the abstraction which deals of how to add text inside PDF.
type Text interface {
	Add(text string, cell Cell, textProp props.Text) float64
	GetLinesQuantity(text string, fontFamily props.Text, colWidth float64) int
}

type text struct {
	pdf  fpdf.Fpdf
	math Math
	font Font
}

// NewText create a Text.
func NewText(pdf fpdf.Fpdf, math Math, font Font) *text {
	return &text{
		pdf,
		math,
		font,
	}
}

// Add a text inside a cell.
func (s *text) Add(text string, cell Cell, textProp props.Text) float64 {
	s.font.SetFont(textProp.Family, textProp.Style, textProp.Size)

	originalColor := s.font.GetColor()
	s.font.SetColor(textProp.Color)

	// duplicated
	_, _, fontSize := s.font.GetFont()
	fontHeight := fontSize / s.font.GetScaleFactor()

	cell.Y += fontHeight

	// Apply Unicode before calc spaces
	unicodeText := s.textToUnicode(text, textProp)
	accumulateOffsetY := 0.0
	lastOffsetY := 0.0

	lines := s.getLines(unicodeText, cell.Width)

	for index, line := range lines {
		lineWidth := s.pdf.GetStringWidth(line)
		_, _, fontSize := s.font.GetFont()
		textHeight := fontSize / s.font.GetScaleFactor()

		lastOffsetY = cell.Y + float64(index)*textHeight + accumulateOffsetY
		s.addLine(textProp, cell.X, cell.Width, cell.Y+float64(index)*textHeight+accumulateOffsetY, lineWidth, line)
		accumulateOffsetY += textProp.VerticalPadding
	}

	s.font.SetColor(originalColor)
	return lastOffsetY
}

// GetLinesQuantity retrieve the quantity of lines which a text will occupy to avoid that text to extrapolate a cell.
func (s *text) GetLinesQuantity(text string, textProp props.Text, colWidth float64) int {
	translator := s.pdf.UnicodeTranslatorFromDescriptor("")
	s.font.SetFont(textProp.Family, textProp.Style, textProp.Size)

	// Apply Unicode.
	textTranslated := translator(text)

	stringWidth := s.pdf.GetStringWidth(textTranslated)
	words := strings.Split(textTranslated, " ")

	// If should add one line.
	if stringWidth < colWidth || textProp.Extrapolate || len(words) == 1 {
		return 1
	}

	lines := s.getLines(textTranslated, colWidth)
	return len(lines)
}

func (s *text) getLines(text string, colWidth float64) []string {
	currentlySize := 0.0
	actualLine := 0
	spaceWidth := s.pdf.GetStringWidth(" ")

	var lines []string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := sc.Text()

		if s.pdf.GetStringWidth(line) < colWidth {
			// Forced line break is valid
			lines = append(lines, line)
			actualLine++
		} else {
			// Need to split again
			var newLine string
			words := strings.Split(line, " ")

			// Split chars
			for i := 0; i < len(words); i++ {
				word := words[i]
				wordWidth := s.pdf.GetStringWidth(word)
				if wordWidth > colWidth {
					// Single word is too long
					runes := []rune(word)
					for k := 0; k < len(runes); k++ {
						ch := string(runes[k])
						charWidth := s.pdf.GetStringWidth(ch)
						if charWidth+currentlySize < colWidth {
							newLine += ch
							currentlySize = currentlySize + charWidth
						} else {
							lines = append(lines, newLine)
							newLine = ""
							actualLine++
							currentlySize = charWidth
						}
					}
				} else {
					if spaceWidth+wordWidth+currentlySize < colWidth {
						newLine += " " + word
						currentlySize = currentlySize + spaceWidth + wordWidth
					} else {
						// Start new line
						lines = append(lines, newLine)
						newLine = word
						actualLine++
						currentlySize = wordWidth
					}
				}
			}
			if newLine != "" {
				// Add line orphans
				lines = append(lines, newLine)
				actualLine++
				newLine = ""
			}
		}
	}

	return lines
}

func (s *text) addLine(textProp props.Text, xColOffset, colWidth, yColOffset, textWidth float64, text string) {
	left, top, _, _ := s.pdf.GetMargins()

	if textProp.Align == consts.Left {
		s.pdf.Text(xColOffset+left, yColOffset+top, text)
		return
	}

	var modifier float64 = 2

	if textProp.Align == consts.Right {
		modifier = 1
	}

	dx := (colWidth - textWidth) / modifier

	s.pdf.Text(dx+xColOffset+left, yColOffset+top, text)
}

func (s *text) textToUnicode(txt string, props props.Text) string {
	if props.Family == consts.Arial ||
		props.Family == consts.Helvetica ||
		props.Family == consts.Symbol ||
		props.Family == consts.ZapBats ||
		props.Family == consts.Courier {
		translator := s.pdf.UnicodeTranslatorFromDescriptor("")
		return translator(txt)
	}

	return txt
}

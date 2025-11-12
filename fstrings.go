package gnome

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// matchInfo stores details about a found f-string match.
type matchInfo struct {
	fullMatchStart int
	fullMatchEnd   int
	contentStart   int
	contentEnd     int
	quoteChar      string
}

// desugarFString takes the content of an f-string (e.g., "hello {name}")
// and converts it into a Python-like format string and a slice of expressions.
// It handles simple {expression} and escaped {{, }}.
// This function performs a character-by-character scan of the f-string's inner content.
func desugarFString(fstringValue string) (formatString string, expressions []string, err error) {
	var parts []string
	var currentPart strings.Builder

	inExpression := false
	braceBalance := 0 // To handle nested braces within an expression (e.g., {func(a, {b: c})})

	// Iterate over the runes of the f-string content
	for i := 0; i < len(fstringValue); i++ {
		r := rune(fstringValue[i])

		if inExpression {
			// Inside an expression, we need to track brace balance to find its end
			if r == '{' {
				braceBalance++
				currentPart.WriteRune(r)
			} else if r == '}' {
				if braceBalance > 0 { // Still inside nested braces within the expression
					braceBalance--
					currentPart.WriteRune(r)
				} else { // End of the top-level expression
					expressions = append(expressions, currentPart.String())
					currentPart.Reset()
					inExpression = false
					parts = append(parts, "{}") // Add placeholder for the expression in the format string
				}
			} else {
				currentPart.WriteRune(r)
			}
		} else { // Not in an expression, processing literal text or escaped braces
			if r == '{' {
				// Check for escaped {{
				if i+1 < len(fstringValue) && fstringValue[i+1] == '{' {
					currentPart.WriteRune('{')
					i++ // Consume the second '{'
				} else { // Start of a new expression
					parts = append(parts, currentPart.String()) // Add accumulated literal part
					currentPart.Reset()
					inExpression = true
					braceBalance = 0 // Reset brace balance for the new expression
				}
			} else if r == '}' {
				// Check for escaped }}
				if i+1 < len(fstringValue) && fstringValue[i+1] == '}' {
					currentPart.WriteRune('}')
					i++ // Consume the second '}'
				} else {
					return "", nil, fmt.Errorf("unmatched '}' in f-string content: %s (at index %d)", fstringValue, i)
				}
			} else {
				currentPart.WriteRune(r)
			}
		}
	}

	// After the loop, check for unclosed expressions or unbalanced braces
	if inExpression {
		return "", nil, fmt.Errorf("unclosed expression in f-string: %s", fstringValue)
	}
	if braceBalance != 0 {
		// This case should ideally be caught by the `inExpression` check unless the logic is flawed
		return "", nil, fmt.Errorf("internal error: unbalanced braces detected after processing f-string")
	}

	parts = append(parts, currentPart.String()) // Add the last literal part

	// Join all parts to form the final format string.
	var finalFormatString strings.Builder
	for _, p := range parts {
		finalFormatString.WriteString(p)
	}

	return finalFormatString.String(), expressions, nil
}

// ConvertFStrings processes Starlark code, replacing f-string literals
// with equivalent string.format() calls.
// This is a source-to-source transformation and might not preserve
// all original formatting (e.g., comments, exact whitespace).
func ConvertFStrings(starlarkSource string) (string, error) {
	// Define separate regexes for each quote type, as Go's regexp doesn't support \1 backreferences in patterns.
	// The `(?:[^"\\]|\\.)*` matches any character that is not a double quote or backslash,
	// OR an escaped character (e.g., \", \\). This ensures correct handling of escaped quotes within the string.
	fStringDoubleQuoteRe := regexp.MustCompile(`f"((?:[^"\\]|\\.)*)"`)
	fStringSingleQuoteRe := regexp.MustCompile(`f'((?:[^'\\]|\\.)*)'`)
	fStringBacktickRe := regexp.MustCompile("f`((?:[^`\\\\]|\\\\.)*)`") // Backtick needs special handling in string literal

	var allMatches []matchInfo

	// Find matches for double-quoted f-strings
	for _, match := range fStringDoubleQuoteRe.FindAllStringSubmatchIndex(starlarkSource, -1) {
		allMatches = append(allMatches, matchInfo{
			fullMatchStart: match[0],
			fullMatchEnd:   match[1],
			contentStart:   match[2],
			contentEnd:     match[3],
			quoteChar:      `"`,
		})
	}

	// Find matches for single-quoted f-strings
	for _, match := range fStringSingleQuoteRe.FindAllStringSubmatchIndex(starlarkSource, -1) {
		allMatches = append(allMatches, matchInfo{
			fullMatchStart: match[0],
			fullMatchEnd:   match[1],
			contentStart:   match[2],
			contentEnd:     match[3],
			quoteChar:      `'`,
		})
	}

	// Find matches for backtick-quoted f-strings
	for _, match := range fStringBacktickRe.FindAllStringSubmatchIndex(starlarkSource, -1) {
		allMatches = append(allMatches, matchInfo{
			fullMatchStart: match[0],
			fullMatchEnd:   match[1],
			contentStart:   match[2],
			contentEnd:     match[3],
			quoteChar:      "`",
		})
	}

	// Sort matches by their starting index to process them in order
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].fullMatchStart < allMatches[j].fullMatchStart
	})

	// Build the new source string by replacing f-strings incrementally
	var sb strings.Builder
	lastIndex := 0

	for _, match := range allMatches {
		// Append the part of the source string before the current f-string
		sb.WriteString(starlarkSource[lastIndex:match.fullMatchStart])

		// Extract the f-string's content
		fstringContent := starlarkSource[match.contentStart:match.contentEnd]

		// Desugar the f-string content into a format string and a list of expressions
		formatString, expressions, err := desugarFString(fstringContent)
		if err != nil {
			// Return an error if desugaring fails for any f-string
			return "", fmt.Errorf("error processing f-string at character %d: %w", match.fullMatchStart, err)
		}

		// Construct the new string.format() call
		sb.WriteString(match.quoteChar) // Re-use the original quote type for the new format string
		sb.WriteString(formatString)
		sb.WriteString(match.quoteChar)

		// If there are expressions, append the .format() call
		if len(expressions) > 0 {
			sb.WriteString(".format(")
			for i, expr := range expressions {
				sb.WriteString(expr) // The extracted expression, e.g., "name" or "1 + 2"
				if i < len(expressions)-1 {
					sb.WriteString(", ") // Add comma separator between expressions
				}
			}
			sb.WriteString(")")
		}

		// Update the last processed index for the next iteration
		lastIndex = match.fullMatchEnd
	}

	// Append any remaining text after the last f-string
	sb.WriteString(starlarkSource[lastIndex:])

	return sb.String(), nil
}

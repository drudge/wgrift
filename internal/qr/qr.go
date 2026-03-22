package qr

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// GeneratePNG generates a QR code as PNG bytes at the given size.
func GeneratePNG(content string, size int) ([]byte, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("generating QR PNG: %w", err)
	}
	return png, nil
}

// GenerateTerminal generates a QR code as unicode block characters for terminal display.
// Uses upper-half and lower-half block characters to render 2 rows per line.
func GenerateTerminal(content string) (string, error) {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("generating QR code: %w", err)
	}

	bitmap := q.Bitmap()
	rows := len(bitmap)
	cols := 0
	if rows > 0 {
		cols = len(bitmap[0])
	}

	var b strings.Builder

	// Process two rows at a time using half-block characters
	for y := 0; y < rows; y += 2 {
		for x := 0; x < cols; x++ {
			top := bitmap[y][x]
			bottom := false
			if y+1 < rows {
				bottom = bitmap[y+1][x]
			}

			switch {
			case top && bottom:
				b.WriteRune('\u2588') // Full block █
			case top && !bottom:
				b.WriteRune('\u2580') // Upper half block ▀
			case !top && bottom:
				b.WriteRune('\u2584') // Lower half block ▄
			default:
				b.WriteRune(' ')
			}
		}
		b.WriteRune('\n')
	}

	return b.String(), nil
}

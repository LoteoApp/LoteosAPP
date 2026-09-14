package domain_test

import (
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestFileContentMatchesMimeType(t *testing.T) {
	cases := []struct {
		name     string
		mimeType string
		header   []byte
		want     bool
	}{
		{"jpeg signature", "image/jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{"png signature", "image/png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n', 0x00}, true},
		{"webp signature", "image/webp", []byte("RIFF\x00\x00\x00\x00WEBP"), true},
		{"pdf signature", "application/pdf", []byte("%PDF-1.7"), true},
		{"jpeg header too short", "image/jpeg", []byte{0xFF, 0xD8}, false},
		{"webp header too short", "image/webp", []byte("RIFF"), false},
		{"plain text claiming jpeg", "image/jpeg", []byte("not a jpeg at all"), false},
		{"png bytes claiming pdf", "application/pdf", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}, false},
		{"unsupported mime type", "text/plain", []byte("hello"), false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := domain.FileContentMatchesMimeType(testCase.mimeType, testCase.header)
			if got != testCase.want {
				t.Errorf("FileContentMatchesMimeType(%q, %q) = %v, want %v",
					testCase.mimeType, testCase.header, got, testCase.want)
			}
		})
	}
}

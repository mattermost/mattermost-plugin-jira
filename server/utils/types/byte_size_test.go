// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package types

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByteSize_String(t *testing.T) {
	tests := []struct {
		name     string
		size     ByteSize
		expected string
	}{
		{
			name:     "zero",
			size:     ByteSize(0),
			expected: "0",
		},
		{
			name:     "bytes",
			size:     ByteSize(42),
			expected: "42b",
		},
		{
			name:     "bytes with commas",
			size:     ByteSize(1234),
			expected: "1.2Kb",
		},
		{
			name:     "kilobytes exact",
			size:     ByteSize(2048),
			expected: "2Kb",
		},
		{
			name:     "kilobytes with decimal",
			size:     ByteSize(1536), // 1.5 KB
			expected: "1.5Kb",
		},
		{
			name:     "megabytes exact",
			size:     ByteSize(2 * 1024 * 1024),
			expected: "2Mb",
		},
		{
			name:     "megabytes with decimal",
			size:     ByteSize(2.5 * 1024 * 1024),
			expected: "2.5Mb",
		},
		{
			name:     "gigabytes",
			size:     ByteSize(3 * 1024 * 1024 * 1024),
			expected: "3Gb",
		},
		{
			name:     "terabytes",
			size:     ByteSize(4 * 1024 * 1024 * 1024 * 1024),
			expected: "4Tb",
		},
		{
			name:     "very large number",
			size:     ByteSize(math.MaxInt64),
			expected: NotAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.size.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedSize   ByteSize
		expectError    bool
		errorSubstring string
	}{
		{
			name:         "bytes",
			input:        "42b",
			expectedSize: ByteSize(42),
		},
		{
			name:         "bytes uppercase",
			input:        "42B",
			expectedSize: ByteSize(42),
		},
		{
			name:         "kilobytes",
			input:        "2kb",
			expectedSize: ByteSize(2 * 1024),
		},
		{
			name:         "kilobytes uppercase",
			input:        "2KB",
			expectedSize: ByteSize(2 * 1024),
		},
		{
			name:         "megabytes",
			input:        "3mb",
			expectedSize: ByteSize(3 * 1024 * 1024),
		},
		{
			name:         "gigabytes",
			input:        "1gb",
			expectedSize: ByteSize(1 * 1024 * 1024 * 1024),
		},
		{
			name:         "terabytes",
			input:        "2tb",
			expectedSize: ByteSize(2 * 1024 * 1024 * 1024 * 1024),
		},
		{
			name:         "with commas",
			input:        "1,234b",
			expectedSize: ByteSize(1234),
		},
		{
			name:         "decimal value",
			input:        "1.5kb",
			expectedSize: ByteSize(1.5 * 1024),
		},
		{
			name:         "plain number",
			input:        "1000",
			expectedSize: ByteSize(1000),
		},
		// The current implementation doesn't support whitespace in the input
		{
			name:        "with whitespaces",
			input:       "42 kb",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expectError: true,
		},
		{
			name:         "negative value",
			input:        "-5kb",
			expectedSize: ByteSize(-5 * 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := ParseByteSize(tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorSubstring != "" {
					assert.Contains(t, err.Error(), tt.errorSubstring)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSize, size)
			}
		})
	}
}

func TestByteSizeRoundTrip(t *testing.T) {
	// Test that ByteSize -> String -> ParseByteSize preserves the value
	testSizes := []ByteSize{
		ByteSize(0),
		ByteSize(1),
		ByteSize(1023),
		ByteSize(1024),
		ByteSize(1025),
		ByteSize(1024 * 1024),
		ByteSize(1024 * 1024 * 1024),
		ByteSize(2.5 * 1024 * 1024),
	}

	for _, size := range testSizes {
		t.Run(size.String(), func(t *testing.T) {
			str := size.String()
			parsed, err := ParseByteSize(str)
			require.NoError(t, err)

			switch {
			case size == 0:
				assert.Equal(t, size, parsed)
			case size < 1024:
				// Bytes should be exact
				assert.Equal(t, size, parsed)
			default:
				// For higher units, there might be rounding differences
				// due to the decimal representation, so we check if they're close
				ratio := float64(parsed) / float64(size)
				assert.InDelta(t, 1.0, ratio, 0.05, "Expected sizes to be within 5% of each other")
			}
		})
	}
}

package pixelforge_cart

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeFooter_Shape(t *testing.T) {
	footer := EncodeFooter([]byte("hello"))
	require.Len(t, footer, FooterSize, "footer must be exactly FooterSize bytes")

	// CRC32 occupies bytes [0:4], LE.
	gotCRC := binary.LittleEndian.Uint32(footer[0:4])
	wantCRC := crc32.ChecksumIEEE([]byte("hello"))
	assert.Equal(t, wantCRC, gotCRC, "CRC32 must match IEEE polynomial over payload")

	// Length occupies bytes [4:12], LE.
	gotLen := binary.LittleEndian.Uint64(footer[4:12])
	assert.Equal(t, uint64(5), gotLen, "length must equal len(payload)")

	// Magic occupies bytes [12:20], ASCII.
	assert.Equal(t, MagicV2, string(footer[12:20]), "footer must end with MagicV2")
}

func TestEncodeDecodeFooter_RoundTrip(t *testing.T) {
	// Various payload sizes — 0 byte boundary, single byte, mid-range,
	// 1 MB. The 256 MB upper bound is enforced by the reader path
	// (selfread_native), not by EncodeFooter itself, so we don't need
	// to exercise it here.
	sizes := []int{0, 1, 100, 1 << 20}
	for _, size := range sizes {
		payload := make([]byte, size)
		for i := range payload {
			payload[i] = byte(i)
		}
		footer := EncodeFooter(payload)
		length, crc, err := DecodeFooter(footer)
		require.NoErrorf(t, err, "DecodeFooter must succeed for size=%d", size)
		assert.Equal(t, uint64(size), length, "decoded length must equal payload size")
		assert.Equal(t, crc32.ChecksumIEEE(payload), crc, "decoded CRC must equal ChecksumIEEE")
	}
}

func TestDecodeFooter_WrongLength(t *testing.T) {
	_, _, err := DecodeFooter(make([]byte, FooterSize-1))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCorruptCart, "short footer must surface as ErrCorruptCart")
}

func TestDecodeFooter_MagicAbsent(t *testing.T) {
	// All-zero footer — no magic anywhere; reader treats as
	// "unshipped player".
	_, _, err := DecodeFooter(make([]byte, FooterSize))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCart, "absent magic must return ErrNoCart cleanly")
}

func TestDecodeFooter_VersionMismatch(t *testing.T) {
	// Synthesize a footer whose magic is "PFORGEv1" — old version,
	// reader recognizes the prefix and returns ErrVersionMismatch
	// rather than treating it as garbage.
	footer := make([]byte, FooterSize)
	copy(footer[12:20], "PFORGEv1")
	_, _, err := DecodeFooter(footer)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVersionMismatch, "PFORGEv1 magic must return ErrVersionMismatch")
	assert.Contains(t, err.Error(), "PFORGEv1", "error must include observed version")
}

func TestDecodeFooter_RandomMagicIsNoCart(t *testing.T) {
	// Magic doesn't start with "PFORGEv" at all — could be any
	// trailing bytes of a player binary that was never shipped.
	footer := make([]byte, FooterSize)
	copy(footer[12:20], "RANDOMBY")
	_, _, err := DecodeFooter(footer)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCart, "non-PFORGE magic must return ErrNoCart")
}

func TestEncodeFooter_CRCDetectsBitFlip(t *testing.T) {
	// Encode, flip a payload byte, verify the recomputed CRC no
	// longer matches the footer's claimed CRC — this is the
	// integrity guarantee EncodeFooter promises.
	payload := []byte("the quick brown fox")
	footer := EncodeFooter(payload)
	_, footerCRC, err := DecodeFooter(footer)
	require.NoError(t, err)

	tampered := bytes.Clone(payload)
	tampered[5] ^= 0xFF
	assert.NotEqual(t, footerCRC, crc32.ChecksumIEEE(tampered), "CRC must change when payload bit-flips")
}

func TestSentinelsAreDistinct(t *testing.T) {
	// Sanity guard: errors.Is must distinguish each sentinel so
	// callers can switch on them. Catches accidental aliasing if
	// someone refactors the var block.
	all := []error{
		ErrNoCart,
		ErrCorruptCart,
		ErrVersionMismatch,
		ErrPayloadTooLarge,
		ErrCodesignUnavailable,
		ErrCodesignFailed,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			assert.False(t, errors.Is(a, b), "sentinel %v must not be %v", a, b)
		}
	}
}

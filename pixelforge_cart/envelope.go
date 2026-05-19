package pixelforge_cart

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
)

// MagicV2 is the ASCII tag the v2 cart footer ends with. The
// version digit is intentionally encoded into the magic itself so
// a reader that finds a v1 footer can return ErrVersionMismatch
// rather than mis-decoding a different layout.
const MagicV2 = "PFORGEv2"

// magicLen is the byte width of every footer magic — both v1 and
// v2 are 8 bytes. Kept separate from FooterSize so future versions
// that grow the magic don't have to refactor the offset math here.
const magicLen = 8

// FooterSize is the total byte count of the appended cart footer:
//
//	CRC32 (4) + length uint64 LE (8) + magic (8) = 20.
//
// Stored at end-of-file. Readers seek to fileSize - FooterSize.
const FooterSize = 4 + 8 + magicLen

// MaxPayloadSize is the hard upper bound the reader enforces on a
// declared cart length BEFORE allocating. Shipped games today are
// well under 50 MB; the 256 MB ceiling exists so a corrupted or
// hostile footer claiming length=18 EB can't OOM the player.
const MaxPayloadSize = 256 << 20 // 256 MiB

// ErrNoCart is returned by ReadSelf when the running executable
// has no v2 footer — i.e. it's a fresh pixelforge-player that
// hasn't been shipped yet. Callers (the player main) distinguish
// this from ErrCorruptCart: ErrNoCart prints a clean "this is an
// unshipped player; nothing to run" message and exits 0.
var ErrNoCart = errors.New("pixelforge_cart: no cart footer present")

// ErrCorruptCart is returned when the footer is present but the
// payload fails verification: CRC mismatch, declared length
// exceeds the remaining file bytes, or any decode error short of
// "magic absent". Surfaces as a fatal startup error in the
// shipped player.
var ErrCorruptCart = errors.New("pixelforge_cart: cart payload failed integrity check")

// ErrVersionMismatch is returned when the magic starts with the
// "PFORGEv" prefix but the version digit doesn't match MagicV2.
// The error message includes the observed magic so build-host
// diagnostics can tell users which version their binary carries.
var ErrVersionMismatch = errors.New("pixelforge_cart: cart version not supported")

// ErrPayloadTooLarge is returned when the footer's declared length
// exceeds MaxPayloadSize. The reader returns this BEFORE allocating
// the payload buffer — the whole point of the cap is to refuse
// hostile or corrupted footers without first eating their declared
// gigabytes of RAM.
var ErrPayloadTooLarge = errors.New("pixelforge_cart: cart payload exceeds maximum size")

// ErrCodesignUnavailable is returned by Append on darwin hosts
// when /usr/bin/codesign is missing from PATH. Common on
// developer machines without Xcode Command Line Tools. The append
// itself has already completed; the caller can decide whether to
// ship the unsigned binary (works on darwin/amd64, fails with
// killed:9 on darwin/arm64).
var ErrCodesignUnavailable = errors.New("pixelforge_cart: codesign not found in PATH")

// ErrCodesignFailed is returned when `codesign --force --sign -`
// exits non-zero. The captured stderr is appended to the error
// message so build-host operators can see exactly what codesign
// complained about (e.g. corrupt binary, missing entitlements).
var ErrCodesignFailed = errors.New("pixelforge_cart: codesign failed")

// EncodeFooter computes the v2 footer for the given payload. The
// returned byte slice is exactly FooterSize bytes; callers append
// it to the cart payload, which itself is appended to the player
// binary.
//
// CRC32 uses the IEEE polynomial (Go's standard crc32.ChecksumIEEE)
// — the same polynomial as the gzip/zip ecosystem, so external
// inspection tools can verify the footer without bespoke code.
func EncodeFooter(payload []byte) []byte {
	footer := make([]byte, FooterSize)
	crc := crc32.ChecksumIEEE(payload)
	binary.LittleEndian.PutUint32(footer[0:4], crc)
	binary.LittleEndian.PutUint64(footer[4:12], uint64(len(payload)))
	copy(footer[12:20], MagicV2)
	return footer
}

// DecodeFooter parses a FooterSize-byte slice and returns the
// declared payload length + CRC. The byte count itself is not
// validated against any real file length here — the caller knows
// the file size and is responsible for cross-checking; this
// function only validates the magic + structural shape.
//
// Errors:
//   - len(footer) != FooterSize    → ErrCorruptCart
//   - magic prefix "PFORGEv" but wrong digit → ErrVersionMismatch
//     wrapped with the observed magic string
//   - any other magic mismatch     → ErrNoCart
func DecodeFooter(footer []byte) (length uint64, crc uint32, err error) {
	if len(footer) != FooterSize {
		return 0, 0, fmt.Errorf("%w: footer length %d, want %d", ErrCorruptCart, len(footer), FooterSize)
	}
	magic := string(footer[12:20])
	if magic != MagicV2 {
		if strings.HasPrefix(magic, "PFORGEv") {
			return 0, 0, fmt.Errorf("%w: observed magic %q, want %q", ErrVersionMismatch, magic, MagicV2)
		}
		return 0, 0, ErrNoCart
	}
	crc = binary.LittleEndian.Uint32(footer[0:4])
	length = binary.LittleEndian.Uint64(footer[4:12])
	return length, crc, nil
}

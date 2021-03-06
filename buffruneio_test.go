package buffruneio

import (
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"unicode/utf8"
	"io"
	"fmt"
)

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Log("unexpected error", err)
		debug.PrintStack()
		t.FailNow()
	}
}

func assumeRunesArray(t *testing.T, expected []rune, got []rune) {
	if len(expected) != len(got) {
		t.Fatal("expected", len(expected), "runes, but got", len(got))
	}
	for i := 0; i < len(got); i++ {
		if expected[i] != got[i] {
			t.Fatal("expected rune", expected[i], "at index", i, "but got", got[i])
		}
	}
}

func assumeRune(t *testing.T, rd *Reader, r rune) {
	gotRune, size, err := rd.ReadRune()
	wantSize := utf8.RuneLen(r)
	if wantSize < 0 {
		wantSize = 0
	}
	if gotRune != r || size != wantSize || err != nil {
		t.Fatalf("ReadRune() = %q, %d, %v, wanted %q, %d, nil", gotRune, size, err, r, wantSize)
	}
}

func assumeBadRune(t *testing.T, rd *Reader) {
	gotRune, size, err := rd.ReadRune()
	if gotRune != utf8.RuneError || size != 1 || err != nil {
		t.Fatalf("ReadRune() = %q, %d, %v, wanted %q, 1, nil", gotRune, size, err, utf8.RuneError)
	}
}

func TestReadString(t *testing.T) {
	s := "hello"
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, 'h')
	assumeRune(t, rd, 'e')
	assumeRune(t, rd, 'l')
	assumeRune(t, rd, 'l')
	assumeRune(t, rd, 'o')
	assumeRune(t, rd, EOF)
}

func TestMultipleEOF(t *testing.T) {
	s := ""
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, EOF)
	assumeRune(t, rd, EOF)
}

func TestBadRunes(t *testing.T) {
	s := "ab\xff\ufffd\xffcd"
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, 'a')
	assumeRune(t, rd, 'b')
	assumeBadRune(t, rd)
	assumeRune(t, rd, utf8.RuneError)
	assumeBadRune(t, rd)
	assumeRune(t, rd, 'c')
	assumeRune(t, rd, 'd')

	for i := 0; i < 6; i++ {
		assertNoError(t, rd.UnreadRune())
	}
	assumeRune(t, rd, 'b')
	assumeBadRune(t, rd)
	assumeRune(t, rd, utf8.RuneError)
	assumeBadRune(t, rd)
	assumeRune(t, rd, 'c')
	assumeRune(t, rd, 'd')
}

func TestUnread(t *testing.T) {
	s := "ab"
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, 'a')
	assumeRune(t, rd, 'b')
	assertNoError(t, rd.UnreadRune())
	assumeRune(t, rd, 'b')
	assumeRune(t, rd, EOF)
}

func TestUnreadEOF(t *testing.T) {
	s := "x"
	rd := NewReader(strings.NewReader(s))

	_ = rd.UnreadRune()
	assumeRune(t, rd, 'x')
	assumeRune(t, rd, EOF)
	assumeRune(t, rd, EOF)
	assertNoError(t, rd.UnreadRune())
	assumeRune(t, rd, EOF)
	assertNoError(t, rd.UnreadRune())
	assertNoError(t, rd.UnreadRune())
	assumeRune(t, rd, EOF)
	assumeRune(t, rd, EOF)
	assertNoError(t, rd.UnreadRune())
	assertNoError(t, rd.UnreadRune())
	assertNoError(t, rd.UnreadRune())
	assumeRune(t, rd, 'x')
	assumeRune(t, rd, EOF)
	assumeRune(t, rd, EOF)
}

func TestForget(t *testing.T) {
	s := "helio"
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, 'h')
	assumeRune(t, rd, 'e')
	assumeRune(t, rd, 'l')
	assumeRune(t, rd, 'i')
	rd.Forget()
	if rd.UnreadRune() != ErrNoRuneToUnread {
		t.Fatal("no rune should be available")
	}
	assumeRune(t, rd, 'o')
}

func TestForgetAfterUnread(t *testing.T) {
	s := "helio"
	rd := NewReader(strings.NewReader(s))

	assumeRune(t, rd, 'h')
	assumeRune(t, rd, 'e')
	assumeRune(t, rd, 'l')
	assumeRune(t, rd, 'i')
	assertNoError(t, rd.UnreadRune())
	rd.Forget()
	if rd.UnreadRune() != ErrNoRuneToUnread {
		t.Fatal("no rune should be available")
	}
	assumeRune(t, rd, 'i')
	assumeRune(t, rd, 'o')
}

func TestForgetEmpty(t *testing.T) {
	s := ""
	rd := NewReader(strings.NewReader(s))

	rd.Forget()
	assumeRune(t, rd, EOF)
	rd.Forget()
}

func TestPeekEmpty(t *testing.T) {
	s := ""
	rd := NewReader(strings.NewReader(s))

	runes := rd.PeekRunes(1)
	if len(runes) != 1 {
		t.Fatal("incorrect number of runes", len(runes))
	}
	if runes[0] != EOF {
		t.Fatal("incorrect rune", runes[0])
	}
}

func TestPeek(t *testing.T) {
	s := "a"
	rd := NewReader(strings.NewReader(s))

	runes := rd.PeekRunes(1)
	assumeRunesArray(t, []rune{'a'}, runes)

	runes = rd.PeekRunes(1)
	assumeRunesArray(t, []rune{'a'}, runes)

	assumeRune(t, rd, 'a')
	runes = rd.PeekRunes(1)
	assumeRunesArray(t, []rune{EOF}, runes)

	assumeRune(t, rd, EOF)
}

func TestPeekLarge(t *testing.T) {
	s := "abcdefg☺\xff☹"
	rd := NewReader(strings.NewReader(s))

	runes := rd.PeekRunes(100)
	want := []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', '☺', utf8.RuneError, '☹', EOF}
	if !reflect.DeepEqual(runes, want) {
		t.Fatalf("PeekRunes(100) = %q, want %q", runes, want)
	}
}

var bigString = strings.Repeat("abcdefghi☺\xff☹", 1024) // 16 kB

const bigStringRunes = 12 * 1024 // 12k runes

func BenchmarkRead16K(b *testing.B) {
	// Read 16K with no unread, no forget.
	benchmarkRead(b, 1, false)
}

func BenchmarkReadForget16K(b *testing.B) {
	// Read 16K, forgetting every 128 runes.
	benchmarkRead(b, 1, true)
}

func BenchmarkReadRewind16K(b *testing.B) {
	// Read 16K, unread all, read that 16K again.
	benchmarkRead(b, 2, false)
}

func benchmarkRead(b *testing.B, count int, forget bool) {
	if len(bigString) != 16*1024 {
		b.Fatal("wrong length for bigString")
	}
	sr0 := strings.NewReader(bigString)
	sr := new(strings.Reader)
	b.SetBytes(int64(len(bigString)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		*sr = *sr0
		rd := NewReader(sr)
		for repeat := 0; repeat < count; repeat++ {
			for j := 0; j < bigStringRunes; j++ {
				r, _, err := rd.ReadRune()
				if err != nil {
					b.Fatal(err)
				}
				if r == EOF {
					b.Fatal("unexpected EOF")
				}
				if forget && j%128 == 127 {
					rd.Forget()
				}
			}
			r, _, err := rd.ReadRune()
			if err != nil {
				b.Fatal(err)
			}
			if r != EOF {
				b.Fatalf("missing EOF - %q", r)
			}
			if repeat == count-1 {
				break
			}
			for rd.UnreadRune() == nil {
				// keep unreading
			}
		}
	}
}

// test reader that will fail reading after a given number of reads
type failingReader struct {
	r io.Reader // underlying reader
	failAfter int // start failing after that number of reads
	readCount int // number of reads already done
}

func newFailingReaderFromString(s string, failAfter int) *failingReader {
	return &failingReader{
		r: strings.NewReader(s),
		failAfter: failAfter,
		readCount: 0,
	}
}

func (r *failingReader) Read(b []byte) (n int, err error) {
	if r.readCount < r.failAfter {
		n, err = r.r.Read(b)
		r.readCount++
		return
	}
	return 0, fmt.Errorf("expected read failure")
}

func TestReadFails(t *testing.T) {
	size := 4097 // needs to be more than bufio.defaultBufSize, which is 4096
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = 'a'
	}

	rd := NewReader(newFailingReaderFromString(string(s), 1))

	runes := rd.PeekRunes(256) // first read, ok

	runes = rd.PeekRunes(1) // rune already loaded, ok

	runes = rd.PeekRunes(4097) // forces a new read, fails
	if len(runes) != 4096 {
		t.Fatalf("expected %d runes. got %d", 4096, len(runes))
	}
	if runes[4095] != 'a' {
		t.Fatalf("expected last rune to be 'a'. got '%c'", runes[4095])
	}


	rd = NewReader(newFailingReaderFromString(string(s), 1))
	for i := 0; i < size - 1; i++ {
		r, size, err := rd.ReadRune() // read all the runes but last
		if err != nil {
			t.Fatalf("no error expeceted at that point, got %s", err)
		}
		if size != 1 {
			t.Fatalf("reading runes that should have size 1, got size %d", size)
		}
		if r != 'a' {
			t.Fatalf("reading a string of 'a', got %c", r)
		}
	}
	//  EOF, 0, err
	r, n, err := rd.ReadRune() // should error
	if r != EOF {
		t.Fatalf("expected EOF, got %c", r)
	}
	if n != 0 {
		t.Fatalf("expected size 0, got %d", n)
	}
	if err.Error() != "expected read failure" {
		t.Fatalf("incorrect error: %s", err.Error())
	}
}

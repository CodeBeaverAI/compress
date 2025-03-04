package gzkp

import (
    "strconv"
    "bytes"
    "github.com/klauspost/compress/gzip"
    "github.com/klauspost/compress/gzhttp/writer"
    "testing"
    "io"
    "time"
)

func TestGzipDoubleClose(t *testing.T) {
    // reset the pool for the default compression so we can make sure duplicates
    // aren't added back by double close
    addLevelPool(gzip.DefaultCompression)

    w := bytes.NewBufferString("")
    writer := NewWriter(w, gzip.DefaultCompression)
    writer.Close()

    // the second close shouldn't have added the same writer
    // so we pull out 2 writers from the pool and make sure they're different
    w1 := gzipWriterPools[poolIndex(gzip.DefaultCompression)].Get()
    w2 := gzipWriterPools[poolIndex(gzip.DefaultCompression)].Get()

    if w1 == w2 {
    t.Fatal("got same writer")
    }
}

// TestNewWriterWriting checks that data written via NewWriter
// can be decompressed to yield the original message.
func TestNewWriterWriting(t *testing.T) {
    var buf bytes.Buffer
    w := NewWriter(&buf, gzip.DefaultCompression)
    message := "Hello, gzip"
    _, err := w.Write([]byte(message))
    if err != nil {
        t.Fatalf("failed to write: %v", err)
    }
    if err = w.Close(); err != nil {
        t.Fatalf("failed to close writer: %v", err)
    }
    r, err := gzip.NewReader(&buf)
    if err != nil {
        t.Fatalf("failed to create gzip reader: %v", err)
    }
    decompressed, err := io.ReadAll(r)
    if err != nil {
        t.Fatalf("failed to read from gzip reader: %v", err)
    }
    r.Close()
    if string(decompressed) != message {
        t.Fatalf("unexpected decompressed message: got %q, want %q", string(decompressed), message)
    }
}

// TestSetHeader verifies that SetHeader correctly sets gzip header fields
// by compressing data and then checking the header values from a gzip.Reader.
func TestSetHeader(t *testing.T) {
    var buf bytes.Buffer
    w := NewWriter(&buf, gzip.DefaultCompression)
    // Create a header with test values.
    header := writer.Header{
        Name:    "test.txt",
        Comment: "sample comment",
        Extra:   []byte("1234"),
        ModTime: time.Unix(1633036800, 0), // example unix time
        OS:      3, // arbitrary OS code
    }
    // Type assertion to gain access to the SetHeader method on *pooledWriter
    if pw, ok := w.(*pooledWriter); ok {
        pw.SetHeader(header)
    } else {
        t.Fatalf("expected concrete type *pooledWriter, got %T", w)
    }
    if _, err := w.Write([]byte("header test")); err != nil {
        t.Fatalf("failed to write: %v", err)
    }
    if err := w.Close(); err != nil {
        t.Fatalf("failed to close writer: %v", err)
    }
    r, err := gzip.NewReader(&buf)
    if err != nil {
        t.Fatalf("failed to create gzip reader: %v", err)
    }
    if r.Name != header.Name {
        t.Fatalf("Name mismatch: got %q, want %q", r.Name, header.Name)
    }
    if r.Comment != header.Comment {
        t.Fatalf("Comment mismatch: got %q, want %q", r.Comment, header.Comment)
    }
    if r.ModTime.Unix() != header.ModTime.Unix() {
        t.Fatalf("ModTime mismatch: got %v, want %v", r.ModTime, header.ModTime)
    }
    // r.Extra holds the entire extra data block. Test for exact match.
    if len(r.Extra) != len(header.Extra) || string(r.Extra) != string(header.Extra) {
        t.Fatalf("Extra mismatch: got %q, want %q", r.Extra, header.Extra)
    }
    r.Close()
}

// TestLevels verifies that the Levels function returns the expected compression levels.
func TestLevels(t *testing.T) {
    min, max := Levels()
    if min != gzip.StatelessCompression {
        t.Fatalf("min level mismatch: got %v, want %v", min, gzip.StatelessCompression)
    }
    if max != gzip.BestCompression {
        t.Fatalf("max level mismatch: got %v, want %v", max, gzip.BestCompression)
    }
}

// TestImplementationInfo verifies that ImplementationInfo returns the correct string.
func TestImplementationInfo(t *testing.T) {
    info := ImplementationInfo()
    expected := "klauspost/compress/gzip"
    if info != expected {
        t.Fatalf("ImplementationInfo mismatch: got %q, want %q", info, expected)
    }
}
// TestWriteAfterClose tests that writing to a closed writer panics.
func TestWriteAfterClose(t *testing.T) {
    var buf bytes.Buffer
    w := NewWriter(&buf, gzip.DefaultCompression)
    w.Close()
    defer func() {
        if r := recover(); r == nil {
            t.Fatal("expected panic when writing after close, but got none")
        }
    }()
    // This should panic.
    w.Write([]byte("this should panic"))
}

// TestConcurrentNewWriter tests that multiple goroutines can safely use NewWriter concurrently.
func TestConcurrentNewWriter(t *testing.T) {
    const numGoroutines = 50
    done := make(chan bool, numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        go func(i int) {
            // Alternate compression levels: use BestCompression for even and DefaultCompression for odd.
            level := gzip.DefaultCompression
            if i%2 == 0 {
                level = gzip.BestCompression
            }
            var localBuf bytes.Buffer
            w := NewWriter(&localBuf, level)
            // Set a header with a unique name for each goroutine.
            header := writer.Header{
                Name:    "file_" + strconv.Itoa(i),
                Comment: "Concurrent test",
                Extra:   []byte{byte(i)},
                ModTime: time.Now(),
                OS:      0,
            }
            if pw, ok := w.(*pooledWriter); ok {
                pw.SetHeader(header)
            }
            if _, err := w.Write([]byte("concurrent test")); err != nil {
                t.Errorf("write failed: %v", err)
            }
            if err := w.Close(); err != nil {
                t.Errorf("close failed: %v", err)
            }
            done <- true
        }(i)
    }

    // Wait for all goroutines to complete.
    for i := 0; i < numGoroutines; i++ {
        <-done
    }
}
// TestPoolIndexMapping verifies that the internal poolIndex function maps each compression level
// to the correct index in the gzipWriterPools array.
func TestPoolIndexMapping(t *testing.T) {
    for level := gzip.StatelessCompression; level <= gzip.BestCompression; level++ {
    expected := level - gzip.StatelessCompression
    got := poolIndex(level)
    if got != expected {
    t.Fatalf("poolIndex(%d) = %d, want %d", level, got, expected)
    }
    }
}

// TestLargeDataCompression verifies that NewWriter can compress a large data block and that the data
// can be decompressed back to the original content.
func TestLargeDataCompression(t *testing.T) {
    var buf bytes.Buffer

    // Generate 100KB of deterministic data.
    data := make([]byte, 100*1024)
    for i := 0; i < len(data); i++ {
    data[i] = byte(i % 256)
    }

    // Create a new gzip writer using DefaultCompression.
    w := NewWriter(&buf, gzip.DefaultCompression)
    n, err := w.Write(data)
    if err != nil {
    t.Fatalf("failed to write data: %v", err)
    }
    if n != len(data) {
    t.Fatalf("written bytes mismatch: got %d, want %d", n, len(data))
    }
    if err = w.Close(); err != nil {
    t.Fatalf("failed to close writer: %v", err)
    }

    // Decompress the data and verify it matches the original.
    r, err := gzip.NewReader(&buf)
    if err != nil {
    t.Fatalf("failed to create gzip reader: %v", err)
    }
    decompressed, err := io.ReadAll(r)
    if err != nil {
    t.Fatalf("failed to read from gzip reader: %v", err)
    }
    r.Close()
    if !bytes.Equal(decompressed, data) {
    t.Fatalf("decompressed data does not match original")
    }
}
// TestInvalidCompressionLevelLow tests that NewWriter panics for an invalid low compression level.
func TestInvalidCompressionLevelLow(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Fatal("expected panic for compression level below valid range, but got none")
        }
    }()
    // This should trigger a panic because the compression level is below gzip.StatelessCompression.
    NewWriter(&bytes.Buffer{}, gzip.StatelessCompression-1)
}

// TestInvalidCompressionLevelHigh tests that NewWriter panics for an invalid high compression level.
func TestInvalidCompressionLevelHigh(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Fatal("expected panic for compression level above valid range, but got none")
        }
    }()
    // This should trigger a panic because the compression level is above gzip.BestCompression.
    NewWriter(&bytes.Buffer{}, gzip.BestCompression+1)
}

// TestWriteEmpty tests that writing an empty byte slice produces a valid gzip stream.
func TestWriteEmpty(t *testing.T) {
    var buf bytes.Buffer
    w := NewWriter(&buf, gzip.DefaultCompression)
    n, err := w.Write([]byte(""))
    if err != nil {
        t.Fatalf("failed to write empty data: %v", err)
    }
    if n != 0 {
        t.Fatalf("expected 0 bytes written for empty data, got %d", n)
    }
    if err = w.Close(); err != nil {
        t.Fatalf("failed to close writer: %v", err)
    }
    r, err := gzip.NewReader(&buf)
    if err != nil {
        t.Fatalf("failed to create gzip reader: %v", err)
    }
    decompressed, err := io.ReadAll(r)
    if err != nil {
        t.Fatalf("failed to read from gzip reader: %v", err)
    }
    r.Close()
    if len(decompressed) != 0 {
        t.Fatalf("expected empty decompressed data, got %d bytes", len(decompressed))
    }
}

// TestResetFunctionality tests that resetting a pooled gzip writer to a new destination works as expected.
func TestResetFunctionality(t *testing.T) {
    var buf1, buf2 bytes.Buffer
    w := NewWriter(&buf1, gzip.DefaultCompression)
    message1 := "first message"
    if _, err := w.Write([]byte(message1)); err != nil {
        t.Fatalf("failed to write first message: %v", err)
    }
    // Now, reset the writer to write to a new underlying writer.
    pw, ok := w.(*pooledWriter)
    if !ok {
        t.Fatalf("expected w to be of type *pooledWriter, got %T", w)
    }
    pw.Reset(&buf2)
    message2 := "second message"
    if _, err := pw.Write([]byte(message2)); err != nil {
        t.Fatalf("failed to write second message: %v", err)
    }
    if err := pw.Close(); err != nil {
        t.Fatalf("failed to close writer: %v", err)
    }
    // Decompress buf2 and verify it only contains the second message.
    r, err := gzip.NewReader(&buf2)
    if err != nil {
        t.Fatalf("failed to create gzip reader: %v", err)
    }
    decompressed, err := io.ReadAll(r)
    if err != nil {
        t.Fatalf("failed to read decompressed data: %v", err)
    }
    r.Close()
    if string(decompressed) != message2 {
        t.Fatalf("reset test failed: got %q, want %q", string(decompressed), message2)
    }
}
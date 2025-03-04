// File: fse/decompress_extra_test.go
package fse

import (
    "errors"
    "strings"
    "testing"
)

///////////////////////////////
func TestDecompress_CorruptedTotal(t *testing.T) {
    // Provide an input that is exactly 4 bytes long. With a minTablelog assumed to be nonzero,
    // the total frequency computed will not match the expected 1<<actualTableLog.
    input := []byte{0, 0, 0, 0}
    s := &Scratch{}
    _, err := Decompress(input, s)
    if err == nil || !strings.Contains(err.Error(), "corruption detected (total") {
    t.Errorf("expected error containing 'corruption detected (total', got: %v", err)
    }
}

///////////////////////////////
func TestBuildDtable_NewStateError(t *testing.T) {
    // Manually create a Scratch and set its fields so we can call buildDtable directly.
    s := &Scratch{}
    // Force an actualTableLog of 5 (so table size = 32)
    s.actualTableLog = 5
    // We need at least two symbols (symbolLen must be > 1)
    s.symbolLen = 2
    // Initialize the norm array (it has 256 elements)
    if s.norm == nil || len(s.norm) < 256 {
    s.norm = make([]int16, 256)
    }
    // Set up a low probability for symbol 0 and an exaggerated count for symbol 1.
    s.norm[0] = -1  // low-probability marker
    s.norm[1] = 100 // an artificially huge count to force an error later

    // Also initialize the counter fields in the inner counter struct.
    if s.ct.stateTable == nil || len(s.ct.stateTable) < 256 {
    s.ct.stateTable = make([]uint16, 256)
    }
    if s.ct.tableSymbol == nil || len(s.ct.tableSymbol) < 256 {
    s.ct.tableSymbol = make([]byte, 256)
    }

    // Allocate the decoding table
    s.allocDtable()

    // As a simple trick, force one decTable slot to be assigned symbol=1 and set the starting count for symbol 1.
    tableSize := uint16(1 << s.actualTableLog)
    s.decTable[0].symbol = 1
    s.ct.stateTable[1] = uint16(s.norm[1])

    // Calling buildDtable should now trigger an error in the newState calculation.
    err := s.buildDtable()
    if err == nil || !strings.Contains(err.Error(), "newState (") {
    t.Errorf("expected buildDtable newState error, got: %v", err)
    }
}

///////////////////////////////
type fakeBitReader struct {
    val          uint32
    finishedFlag bool
}

func (f *fakeBitReader) init(data []byte) error {
    // Fake implementation does nothing.
    return nil
}

func (f *fakeBitReader) getBits(n uint8) uint32 {
    // Return the lowest n bits from f.val.
    return f.val & ((1 << n) - 1)
}

func (f *fakeBitReader) getBitsFast(n uint8) uint32 {
    return f.getBits(n)
}

func (f *fakeBitReader) finished() bool {
    return f.finishedFlag
}

func (f *fakeBitReader) fillFast() {}
func (f *fakeBitReader) fill() {}
func (f *fakeBitReader) close() error { return nil }
func (f *fakeBitReader) unread() []byte { return nil }

// fakeBits is a fake implementation of bitReader used for testing the error path and controlled behavior.
type fakeBits struct {
    off           int
    finishedFlag  bool
}

func (f *fakeBits) init(data []byte) error {
    return errors.New("fake bits init error")
}

func (f *fakeBits) getBits(n uint8) uint32 {
    // Always return 0 bits regardless of n.
    return 0
}

func (f *fakeBits) getBitsFast(n uint8) uint32 {
    return 0
}

func (f *fakeBits) finished() bool {
    return f.finishedFlag
}

func (f *fakeBits) fillFast() {
    f.off = 0
}

func (f *fakeBits) fill() {
    f.off = 0
}

func (f *fakeBits) close() error { return nil }

func (f *fakeBits) unread() []byte { return nil }
///////////////////////////////
func TestDecoderMethods(t *testing.T) {
    // Create a simple decSymbol table with a single element.
    ds := []decSymbol{
    {newState: 0, symbol: 'X', nbBits: 0},
    }

    // Create a fake bitReader which always returns 0 and is flagged as finished.
    fbr := &fakeBitReader{val: 0, finishedFlag: true}

    var d decoder
    d.dt = ds
    d.br = fbr
    d.state = 0

    // Test next decoding method.
    sym := d.next()
    if sym != 'X' {
    t.Errorf("expected symbol 'X', got %c", sym)
    }

    // Test nextFast decoding method.
    d.state = 0
    symFast := d.nextFast()
    if symFast != 'X' {
    t.Errorf("expected symbol 'X' from nextFast, got %c", symFast)
    }

    // Test finished and final methods.
    if !d.finished() {
    t.Errorf("expected decoder to be finished")
    }
    symFinal := d.final()
    if symFinal != 'X' {
    t.Errorf("expected final symbol 'X', got %c", symFinal)
    }
}

// End of file: fse/decompress_extra_test.go
    t.Run(tt.Name, func(t *testing.T) {
    s, rem, err := ReadTable(data, nil)
    if err != nil {
    t.Fatal(err)
    }
    _, err = s.Decompress1X(rem)
    if err == nil {
    t.Fatal("expected error to be returned")
    }

    t.Logf("returned error: %s", err)
    })
    }
}

func TestDecompress4X(t *testing.T) {
    for _, test := range testfiles {
    t.Run(test.name, func(t *testing.T) {
    for _, tl := range []uint8{0, 5, 6, 7, 8, 9, 10, 11} {
    t.Run(fmt.Sprintf("tablelog-%d", tl), func(t *testing.T) {
        var s = &Scratch{}
        s.TableLog = tl
        buf0, err := test.fn()
        if err != nil {
        t.Fatal(err)
        }
        if len(buf0) > BlockSizeMax {
        buf0 = buf0[:BlockSizeMax]
        }
        b, re, err := Compress4X(buf0, s)
        if err != test.err4X {
        t.Errorf("want error %v (%T), got %v (%T)", test.err1X, test.err1X, err, err)
        }
        if err != nil {
        t.Log(test.name, err.Error())
        return
        }
        if b == nil {
        t.Error("got no output")
        return
        }
        if len(s.OutTable) == 0 {
        t.Error("got no table definition")
        }
        if re {
        t.Error("claimed to have re-used.")
        }
        if len(s.OutData) == 0 {
        t.Error("got no data output")
        }

        wantRemain := len(s.OutData)
        t.Logf("%s: %d -> %d bytes (%.2f:1) %t (table: %d bytes)", test.name, len(buf0), len(b), float64(len(buf0))/float64(len(b)), re, len(s.OutTable))

        s.Out = nil
        var remain []byte
        s, remain, err = ReadTable(b, s)
        if err != nil {
        t.Error(err)
        return
        }
        var buf bytes.Buffer
        if s.matches(s.prevTable, &buf); buf.Len() > 0 {
        t.Error(buf.String())
        }
        if len(remain) != wantRemain {
        t.Fatalf("remain mismatch, want %d, got %d bytes", wantRemain, len(remain))
        }
        t.Logf("remain: %d bytes, ok", len(remain))
        dc, err := s.Decompress4X(remain, len(buf0))
        if err != nil {
        t.Error(err)
        return
        }
        if len(buf0) != len(dc) {
        t.Errorf(test.name+"decompressed, want size: %d, got %d", len(buf0), len(dc))
        if len(buf0) > len(dc) {
        buf0 = buf0[:len(dc)]
        } else {
        dc = dc[:len(buf0)]
        }
        if !bytes.Equal(buf0, dc) {
        if len(dc) > 1024 {
        t.Log(string(dc[:1024]))
        t.Errorf(test.name+"decompressed, got delta: \n(in)\t%02x !=\n(out)\t%02x\n", buf0[:1024], dc[:1024])
        } else {
        t.Log(string(dc))
        t.Errorf(test.name+"decompressed, got delta: (in) %v != (out) %v\n", buf0, dc)
        }
        }
        return
        }
        if !bytes.Equal(buf0, dc) {
        if len(buf0) > 1024 {
        t.Log(string(dc[:1024]))
        } else {
        t.Log(string(dc))
        }
        //t.Errorf(test.name+": decompressed, got delta: \n%s")
        t.Errorf(test.name + ": decompressed, got delta")
        }
        if !t.Failed() {
        t.Log("... roundtrip ok!")
        }

    })
    }
    })
    }
}

func TestRoundtrip1XFuzz(t *testing.T) {
    for _, test := range testfilesExtended {
    t.Run(test.name, func(t *testing.T) {
    var s = &Scratch{}
    buf0, err := test.fn()
    if err != nil {
    t.Fatal(err)
    }
    if len(buf0) > BlockSizeMax {
    buf0 = buf0[:BlockSizeMax]
    }
    b, re, err := Compress1X(buf0, s)
    if err != nil {
    if err == ErrIncompressible || err == ErrUseRLE || err == ErrTooBig {
        t.Log(test.name, err.Error())
        return
    }
    t.Error(test.name, err.Error())
    return
    }
    if b == nil {
    t.Error("got no output")
    return
    }
    if len(s.OutTable) == 0 {
    t.Error("got no table definition")
    }
    if re {
    t.Error("claimed to have re-used.")
    }
    if len(s.OutData) == 0 {
    t.Error("got no data output")
    }

    wantRemain := len(s.OutData)
    t.Logf("%s: %d -> %d bytes (%.2f:1) %t (table: %d bytes)", test.name, len(buf0), len(b), float64(len(buf0))/float64(len(b)), re, len(s.OutTable))

    s.Out = nil
    var remain []byte
    s, remain, err = ReadTable(b, s)
    if err != nil {
    t.Error(err)
    return
    }
    var buf bytes.Buffer
    if s.matches(s.prevTable, &buf); buf.Len() > 0 {
    t.Error(buf.String())
    }
    if len(remain) != wantRemain {
    t.Fatalf("remain mismatch, want %d, got %d bytes", wantRemain, len(remain))
    }
    t.Logf("remain: %d bytes, ok", len(remain))
    dc, err := s.Decompress1X(remain)
    if err != nil {
    t.Error(err)
    return
    }
    if len(buf0) != len(dc) {
    t.Errorf(test.name+"decompressed, want size: %d, got %d", len(buf0), len(dc))
    if len(buf0) > len(dc) {
        buf0 = buf0[:len(dc)]
    } else {
        dc = dc[:len(buf0)]
    }
    if !bytes.Equal(buf0, dc) {
        if len(dc) > 1024 {
        t.Log(string(dc[:1024]))
        t.Errorf(test.name+"decompressed, got delta: \n(in)\t%02x !=\n(out)\t%02x\n", buf0[:1024], dc[:1024])
        } else {
        t.Log(string(dc))
        t.Errorf(test.name+"decompressed, got delta: (in) %v != (out) %v\n", buf0, dc)
        }
    }
    return
    }
    if !bytes.Equal(buf0, dc) {
    if len(buf0) > 1024 {
        t.Log(string(dc[:1024]))
    } else {
        t.Log(string(dc))
    }
    //t.Errorf(test.name+": decompressed, got delta: \n%s")
    t.Errorf(test.name + ": decompressed, got delta")
    }
    if !t.Failed() {
    t.Log("... roundtrip ok!")
    }
    })
    }
}

func TestRoundtrip4XFuzz(t *testing.T) {
    for _, test := range testfilesExtended {
    t.Run(test.name, func(t *testing.T) {
    var s = &Scratch{}
    buf0, err := test.fn()
    if err != nil {
    t.Fatal(err)
    }
    if len(buf0) > BlockSizeMax {
    buf0 = buf0[:BlockSizeMax]
    }
    b, re, err := Compress4X(buf0, s)
    if err != nil {
    if err == ErrIncompressible || err == ErrUseRLE || err == ErrTooBig {
        t.Log(test.name, err.Error())
        return
    }
    t.Error(test.name, err.Error())
    return
    }
    if b == nil {
    t.Error("got no output")
    return
    }
    if len(s.OutTable) == 0 {
    t.Error("got no table definition")
    }
    if re {
    t.Error("claimed to have re-used.")
    }
    if len(s.OutData) == 0 {
    t.Error("got no data output")
    }

    wantRemain := len(s.OutData)
    t.Logf("%s: %d -> %d bytes (%.2f:1) %t (table: %d bytes)", test.name, len(buf0), len(b), float64(len(buf0))/float64(len(b)), re, len(s.OutTable))

    s.Out = nil
    var remain []byte
    s, remain, err = ReadTable(b, s)
    if err != nil {
    t.Error(err)
    return
    }
    var buf bytes.Buffer
    if s.matches(s.prevTable, &buf); buf.Len() > 0 {
    t.Error(buf.String())
    }
    if len(remain) != wantRemain {
    t.Fatalf("remain mismatch, want %d, got %d bytes", wantRemain, len(remain))
    }
    t.Logf("remain: %d bytes, ok", len(remain))
    dc, err := s.Decompress4X(remain, len(buf0))
    if err != nil {
    t.Error(err)
    return
    }
    if len(buf0) != len(dc) {
    t.Errorf(test.name+"decompressed, want size: %d, got %d", len(buf0), len(dc))
    if len(buf0) > len(dc) {
        buf0 = buf0[:len(dc)]
    } else {
        dc = dc[:len(buf0)]
    }
    if !bytes.Equal(buf0, dc) {
        if len(dc) > 1024 {
        t.Log(string(dc[:1024]))
        t.Errorf(test.name+"decompressed, got delta: \n(in)\t%02x !=\n(out)\t%02x\n", buf0[:1024], dc[:1024])
        } else {
        t.Log(string(dc))
        t.Errorf(test.name+"decompressed, got delta: (in) %v != (out) %v\n", buf0, dc)
        }
    }
    return
    }
    if !bytes.Equal(buf0, dc) {
    if len(buf0) > 1024 {
        t.Log(string(dc[:1024]))
    } else {
        t.Log(string(dc))
    }
    //t.Errorf(test.name+": decompressed, got delta: \n%s")
    t.Errorf(test.name + ": decompressed, got delta")
    }
    if !t.Failed() {
    t.Log("... roundtrip ok!")
    }
    })
    }
}

func BenchmarkDecompress1XTable(b *testing.B) {
    for _, tt := range testfiles {
    test := tt
    if test.err1X != nil {
    continue
    }
    b.Run(test.name, func(b *testing.B) {
    var s = &Scratch{}
    s.Reuse = ReusePolicyNone
    buf0, err := test.fn()
    if err != nil {
    b.Fatal(err)
    }
    if len(buf0) > BlockSizeMax {
    buf0 = buf0[:BlockSizeMax]
    }
    compressed, _, err := Compress1X(buf0, s)
    if err != test.err1X {
    b.Fatal("unexpected error:", err)
    }
    s.Out = nil
    s, remain, _ := ReadTable(compressed, s)
    s.Decompress1X(remain)
    b.ResetTimer()
    b.ReportAllocs()
    b.SetBytes(int64(len(buf0)))
    for i := 0; i < b.N; i++ {
    s, remain, err := ReadTable(compressed, s)
    if err != nil {
        b.Fatal(err)
    }
    _, err = s.Decompress1X(remain)
    if err != nil {
        b.Fatal(err)
    }
    }
    })
    }
}

func BenchmarkDecompress1XNoTable(b *testing.B) {
    for _, tt := range testfiles {
    test := tt
    if test.err1X != nil {
    continue
    }
    b.Run(test.name, func(b *testing.B) {
    for _, sz := range []int{1e2, 1e4, BlockSizeMax} {
    b.Run(fmt.Sprintf("%d", sz), func(b *testing.B) {
        var s = &Scratch{}
        s.Reuse = ReusePolicyNone
        buf0, err := test.fn()
        if err != nil {
        b.Fatal(err)
        }
        for len(buf0) < sz {
        buf0 = append(buf0, buf0...)
        }
        if len(buf0) > sz {
        buf0 = buf0[:sz]
        }
        compressed, _, err := Compress1X(buf0, s)
        if err != test.err1X {
        if err == ErrUseRLE {
        b.Skip("RLE")
        return
        }
        b.Fatal("unexpected error:", err)
        }
        s.Out = nil
        s, remain, _ := ReadTable(compressed, s)
        s.Decompress1X(remain)
        b.ResetTimer()
        b.ReportAllocs()
        b.SetBytes(int64(len(buf0)))
        for i := 0; i < b.N; i++ {
        _, err = s.Decompress1X(remain)
        if err != nil {
        b.Fatal(err)
        }
        }
        b.ReportMetric(float64(s.actualTableLog), "log")
        b.ReportMetric(100*float64(len(compressed))/float64(len(buf0)), "pct")
    })
    }
    })
    }
}

func BenchmarkDecompress4XNoTable(b *testing.B) {
    for _, tt := range testfiles {
    test := tt
    if test.err4X != nil {
    continue
    }
    b.Run(test.name, func(b *testing.B) {
    for _, sz := range []int{1e2, 1e4, BlockSizeMax} {
    b.Run(fmt.Sprintf("%d", sz), func(b *testing.B) {
        var s = &Scratch{}
        s.Reuse = ReusePolicyNone
        buf0, err := test.fn()
        if err != nil {
        b.Fatal(err)
        }
        for len(buf0) < sz {
        buf0 = append(buf0, buf0...)
        }
        if len(buf0) > sz {
        buf0 = buf0[:sz]
        }
        compressed, _, err := Compress4X(buf0, s)
        if err != test.err4X {
        if err == ErrUseRLE {
        b.Skip("RLE")
        return
        }
        b.Fatal("unexpected error:", err)
        }
        s.Out = nil
        s, remain, _ := ReadTable(compressed, s)
        s.Decompress4X(remain, len(buf0))
        b.ResetTimer()
        b.ReportAllocs()
        b.SetBytes(int64(len(buf0)))
        for i := 0; i < b.N; i++ {
        _, err = s.Decompress4X(remain, len(buf0))
        if err != nil {
        b.Fatal(err)
        }
        }
        b.ReportMetric(float64(s.actualTableLog), "log")
        b.ReportMetric(100*float64(len(compressed))/float64(len(buf0)), "pct")

    })
    }
    })
    }
}

func BenchmarkDecompress4XNoTableTableLog8(b *testing.B) {
    for _, tt := range testfiles[:1] {
    test := tt
    if test.err4X != nil {
    continue
    }
    b.Run(test.name, func(b *testing.B) {
    var s = &Scratch{}
    s.Reuse = ReusePolicyNone
    buf0, err := test.fn()
    if err != nil {
    b.Fatal(err)
    }
    if len(buf0) > BlockSizeMax {
    buf0 = buf0[:BlockSizeMax]
    }
    s.TableLog = 8
    compressed, _, err := Compress4X(buf0, s)
    if err != test.err1X {
    b.Fatal("unexpected error:", err)
    }
    s.Out = nil
    s, remain, _ := ReadTable(compressed, s)
    s.Decompress4X(remain, len(buf0))
    b.ResetTimer()
    b.ReportAllocs()
    b.SetBytes(int64(len(buf0)))
    for i := 0; i < b.N; i++ {
    _, err = s.Decompress4X(remain, len(buf0))
    if err != nil {
        b.Fatal(err)
    }
    }
    })
    }
}

func BenchmarkDecompress4XTable(b *testing.B) {
    for _, tt := range testfiles {
    test := tt
    if test.err4X != nil {
    continue
    }
    b.Run(test.name, func(b *testing.B) {
    var s = &Scratch{}
    s.Reuse = ReusePolicyNone
    buf0, err := test.fn()
    if err != nil {
    b.Fatal(err)
    }
    if len(buf0) > BlockSizeMax {
    buf0 = buf0[:BlockSizeMax]
    }
    compressed, _, err := Compress4X(buf0, s)
    if err != test.err1X {
    b.Fatal("unexpected error:", err)
    }
    s.Out = nil
    b.ResetTimer()
    b.ReportAllocs()
    b.SetBytes(int64(len(buf0)))
    for i := 0; i < b.N; i++ {
    s, remain, err := ReadTable(compressed, s)
    if err != nil {
        b.Fatal(err)
    }
    _, err = s.Decompress4X(remain, len(buf0))
    if err != nil {
        b.Fatal(err)
    }
    }
    })
    }
}

// TestFSEDecompress_EmptyInput verifies that an input with fewer than 4 bytes returns an "input too small" error.
func TestFSEDecompress_EmptyInput(t *testing.T) {
    s := &fse.Scratch{}
    _, err := fse.Decompress([]byte{}, s)
    if err == nil || !strings.Contains(err.Error(), "input too small") {
    t.Errorf("expected error 'input too small', got: %v", err)
    }
}

// TestFSEDecompress_TableLogTooLarge verifies that an input whose tableLog is too large returns the expected error.
func TestFSEDecompress_TableLogTooLarge(t *testing.T) {
    // Assuming a minTablelog of 5, then nibble 11 gives nbBits = 16 which is > tablelogAbsoluteMax (15).
    data := []byte{0xB, 0, 0, 0, 0, 0, 0, 0}
    s := &fse.Scratch{}
    _, err := fse.Decompress(data, s)
    if err == nil || !strings.Contains(err.Error(), "tableLog too large") {
    t.Errorf("expected error 'tableLog too large', got: %v", err)
    }
}

// TestFSEDecompress_NilScratch verifies that passing a nil Scratch pointer causes a panic.
func TestFSEDecompress_NilScratch(t *testing.T) {
    defer func() {
    if r := recover(); r == nil {
    t.Errorf("expected panic when scratch is nil")
    }
    }()
    data := []byte{0, 0, 0, 0, 0, 0, 0, 0}
    fse.Decompress(data, nil)
}
// TestDecompress_ErrorFromBitsInit verifies that an error in fakeBits.init is properly returned.
func TestDecompress_ErrorFromBitsInit(t *testing.T) {
    // Create a Scratch object and assign its internal bits to our fakeBits that returns an error.
    s := &Scratch{}
    // Assume br is already set to a valid fake (we use the existing fakeBitReader for s.br).
    s.br = &fakeBitReader{val: 0, finishedFlag: true}
    // Override the bits field with our fakeBits instance.
    s.bits = fakeBits{off: 8, finishedFlag: false}

    // Since fakeBits.init always errors, Decompress should catch that error.
    _, err := Decompress([]byte{0, 0, 0, 0, 0, 0, 0, 0}, s)
    if err == nil || !strings.Contains(err.Error(), "fake bits init error") {
        t.Errorf("expected error from fake bits init, got: %v", err)
    }
}

// TestDecompress_FakeDecoder simulates a minimal valid decompression flow by manually
// setting up the Scratch fields so that the decoder always produces a known symbol.
func TestDecompress_FakeDecoder(t *testing.T) {
    s := &Scratch{}

    // Manually set up minimal fields: use a table log of 1 (so table size = 2) and symbolLen = 2.
    s.actualTableLog = 1
    s.symbolLen = 2
    s.DecompressLimit = 1000

    // Allocate a decTable of size 2. Both entries will always output the symbol 'A'
    s.decTable = make([]decSymbol, 2)
    for i := 0; i < 2; i++ {
        s.decTable[i] = decSymbol{
            newState: 0,
            nbBits:   0,
            symbol:   'A',
        }
    }

    // Set s.zeroBits to false so that the fast decoding loop in decompress is used.
    s.zeroBits = false

    // Set up s.bits as a fakeBits that simulates having enough bits to enter the main loop,
    // then after a call to fillFast, the off counter goes to 0 and finished() returns true.
    s.bits = fakeBits{
        off:          8, // ensure the main loop is entered once
        finishedFlag: true,
    }

    // Also set s.br to an instance of fakeBitReader so that any calls in readNCount/buildDtable do not error.
    s.br = &fakeBitReader{val: 0, finishedFlag: true}

    // We override the following steps by not calling readNCount and buildDtable.
    // Instead, we call the decompress method directly.
    // The decoder is initialized in decompress() and will use our s.decTable.
    err := s.decompress()
    if err != nil {
        t.Fatalf("decompress() returned error: %v", err)
    }

    // In our simulation:
    // - The fast decoding loop (when s.zeroBits is false) will append 4 symbols ('A') from the tmp buffer.
    // - Then, the final loop will detect that the decoder is finished and append two additional symbols.
    // So we expect a total output of 6 symbols: "AAAAAA".
    expected := []byte("AAAAAA")
    if string(s.Out) != string(expected) {
        t.Errorf("expected output %q, got %q", expected, s.Out)
    }
}
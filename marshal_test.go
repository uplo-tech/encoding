package encoding

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	// EncodingTestingDir is the directory that contains all of the files and
	// folders created during testing.
	EncodingTestingDir = filepath.Join(os.TempDir(), "EncodingTesting")
)

// TempDir joins the provided directories and prefixes them with the testing
// directory.
func TempDir(dirs ...string) string {
	path := filepath.Join(EncodingTestingDir, filepath.Join(dirs...))
	err := os.RemoveAll(path) // remove old test data
	if err != nil {
		panic(err)
	}
	return path
}

// dummy types to test encoding
type (
	// basic
	test0 struct {
		B bool
		I int32
		U uint16
		S string
	}
	// slice/array
	test1 struct {
		Is []int32
		Bs []byte
		Sa [3]string
		Ba [3]byte
	}
	// nested
	test2 struct {
		T test0
	}
	// embedded
	test3 struct {
		test2
	}
	// pointer
	test4 struct {
		P *test1
	}
	// private field -- need to implement MarshalUplo/UnmarshalUplo
	test5 struct {
		s string
	}
	// private field with pointer receiver
	test6 struct {
		s string
	}
	// nil pointer
	test7 struct{}
)

func (t test5) MarshalUplo(w io.Writer) error {
	return NewEncoder(w).WritePrefixedBytes([]byte(t.s))
}

func (t *test5) UnmarshalUplo(r io.Reader) error {
	d := NewDecoder(r, 1e6)
	t.s = string(d.ReadPrefixedBytes())
	return d.Err()
}

// same as above methods, but with a pointer receiver
func (t *test6) MarshalUplo(w io.Writer) error {
	return NewEncoder(w).WritePrefixedBytes([]byte(t.s))
}

func (t *test6) UnmarshalUplo(r io.Reader) error {
	d := NewDecoder(r, 1e6)
	t.s = string(d.ReadPrefixedBytes())
	return d.Err()
}

var testStructs = []interface{}{
	test0{false, 65537, 256, "foo"},
	test1{[]int32{1, 2, 3}, []byte("foo"), [3]string{"foo", "bar", "baz"}, [3]byte{'f', 'o', 'o'}},
	test2{test0{false, 65537, 256, "foo"}},
	test3{test2{test0{false, 65537, 256, "foo"}}},
	test4{&test1{[]int32{1, 2, 3}, []byte("foo"), [3]string{"foo", "bar", "baz"}, [3]byte{'f', 'o', 'o'}}},
	test5{"foo"},
	&test6{"foo"},
	(*test7)(nil),
}

var testEncodings = [][]byte{
	{0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o'},
	{3, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0,
		0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 3,
		0, 0, 0, 0, 0, 0, 0, 'b', 'a', 'r', 3, 0, 0, 0, 0, 0, 0, 0, 'b', 'a', 'z', 'f', 'o', 'o'},
	{0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o'},
	{0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o'},
	{1, 3, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0,
		0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 3,
		0, 0, 0, 0, 0, 0, 0, 'b', 'a', 'r', 3, 0, 0, 0, 0, 0, 0, 0, 'b', 'a', 'z', 'f', 'o', 'o'},
	{3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o'},
	{3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o'},
	{0},
}

// TestEncode tests the Encode function.
func TestEncode(t *testing.T) {
	// use Marshal for convenience
	for i := range testStructs {
		b := Marshal(testStructs[i])
		if !bytes.Equal(b, testEncodings[i]) {
			t.Errorf("bad encoding of testStructs[%d]: \nexp:\t%v\ngot:\t%v", i, testEncodings[i], b)
		}
	}

	// bad type
	defer func() {
		if recover() == nil {
			t.Error("expected panic, got nil")
		}
	}()
	err := NewEncoder(ioutil.Discard).Encode(map[int]int{})
	if err != nil {
		t.Fatal(err)
	}
}

// TestDecode tests the Decode function.
func TestDecode(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	// use Unmarshal for convenience
	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	for i := range testEncodings {
		err := Unmarshal(testEncodings[i], emptyStructs[i])
		if err != nil {
			t.Error(err)
		}
	}

	// bad boolean
	err := Unmarshal([]byte{3}, new(bool))
	if err == nil || err.Error() != "could not decode type bool: boolean value was not 0 or 1" {
		t.Error("expected bool error, got", err)
	}

	// non-pointer
	err = Unmarshal([]byte{1, 2, 3}, "foo")
	if err != ErrBadPointer {
		t.Error("expected ErrBadPointer, got", err)
	}

	// unknown type
	err = Unmarshal([]byte{1, 2, 3}, new(map[int]int))
	if err == nil || err.Error() != "could not decode type map[int]int: unknown type" {
		t.Error("expected unknown type error, got", err)
	}

	// badReader should fail on every decode
	dec := NewDecoder(new(badReader), 1e6)
	for i := range testEncodings {
		err := dec.Decode(emptyStructs[i])
		if err == nil {
			t.Error("expected error, got nil")
		}
	}
	// special case, not covered by testStructs
	err = dec.Decode(new([3]byte))
	if err == nil || err.Error() != "could not decode type [3]uint8: EOF" {
		t.Error("expected EOF error, got", err)
	}
}

// TestMarshalUnmarshal tests the Marshal and Unmarshal functions, which are
// inverses of each other.
func TestMarshalUnmarshal(t *testing.T) {
	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	for i := range testStructs {
		b := Marshal(testStructs[i])
		err := Unmarshal(b, emptyStructs[i])
		if err != nil {
			t.Error(err)
		}
	}
}

// TestEncodeDecode tests the Encode and Decode functions, which are inverses
// of each other.
func TestEncodeDecode(t *testing.T) {
	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	b := new(bytes.Buffer)
	enc := NewEncoder(b)
	dec := NewDecoder(b, 1e6)
	for i := range testStructs {
		err := enc.Encode(testStructs[i])
		if err != nil {
			t.Error(err)
		}
		err = dec.Decode(emptyStructs[i])
		if err != nil {
			t.Error(err)
		}
	}
}

// TestEncodeAll tests the EncodeAll function.
func TestEncodeAll(t *testing.T) {
	// EncodeAll should produce the same result as individually encoding each
	// object
	exp := new(bytes.Buffer)
	enc := NewEncoder(exp)
	for i := range testStructs {
		err := enc.Encode(testStructs[i])
		if err != nil {
			t.Error(err)
		}
	}

	b := new(bytes.Buffer)
	err := NewEncoder(b).EncodeAll(testStructs...)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(b.Bytes(), exp.Bytes()) {
		t.Errorf("expected %v, got %v", exp.Bytes(), b.Bytes())
	}

	// hardcoded check
	exp.Reset()
	_, err = exp.Write([]byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 1})
	if err != nil {
		t.Error(err)
	}

	b.Reset()
	err = NewEncoder(b).EncodeAll(1, 2, "foo", true)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(b.Bytes(), exp.Bytes()) {
		t.Errorf("expected %v, got %v", exp.Bytes(), b.Bytes())
	}
}

// TestDecodeAll tests the DecodeAll function.
func TestDecodeAll(t *testing.T) {
	b := new(bytes.Buffer)
	err := NewEncoder(b).EncodeAll(testStructs...)
	if err != nil {
		t.Error(err)
	}

	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	err = NewDecoder(b, 1e6).DecodeAll(emptyStructs...)
	if err != nil {
		t.Error(err)
	}
	empty0 := *emptyStructs[0].(*test0)
	if !reflect.DeepEqual(empty0, testStructs[0]) {
		t.Error("deep equal:", empty0, testStructs[0])
	}
	empty6 := emptyStructs[6].(*test6)
	if !reflect.DeepEqual(empty6, testStructs[6]) {
		t.Error("deep equal:", empty6, testStructs[6])
	}

	// hardcoded check
	b.Reset()
	_, err = b.Write([]byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 1})
	if err != nil {
		t.Error(err)
	}

	var (
		one, two uint64
		foo      string
		tru      bool
	)
	err = NewDecoder(b, 1e6).DecodeAll(&one, &two, &foo, &tru)
	if err != nil {
		t.Fatal(err)
	} else if one != 1 || two != 2 || foo != "foo" || tru != true {
		t.Error("values were not decoded correctly:", one, two, foo, tru)
	}
}

// TestMarshalAll tests the MarshalAll function.
func TestMarshalAll(t *testing.T) {
	// MarshalAll should produce the same result as individually marshalling
	// each object
	var expected []byte
	for i := range testStructs {
		expected = append(expected, Marshal(testStructs[i])...)
	}

	b := MarshalAll(testStructs...)
	if !bytes.Equal(b, expected) {
		t.Errorf("expected %v, got %v", expected, b)
	}

	// hardcoded check
	exp := []byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 1}
	b = MarshalAll(1, 2, "foo", true)
	if !bytes.Equal(b, exp) {
		t.Errorf("expected %v, got %v", exp, b)
	}
}

// TestUnmarshalAll tests the UnmarshalAll function.
func TestUnmarshalAll(t *testing.T) {
	b := MarshalAll(testStructs...)

	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	err := UnmarshalAll(b, emptyStructs...)
	if err != nil {
		t.Error(err)
	}
	empty1 := *emptyStructs[1].(*test1)
	if !reflect.DeepEqual(empty1, testStructs[1]) {
		t.Error("deep equal:", empty1, testStructs[1])
	}
	empty5 := *emptyStructs[5].(*test5)
	if !reflect.DeepEqual(empty5, testStructs[5]) {
		t.Error("deep equal:", empty5, testStructs[5])
	}

	// hardcoded check
	b = []byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 'f', 'o', 'o', 1}
	var (
		one, two uint64
		foo      string
		tru      bool
	)
	err = UnmarshalAll(b, &one, &two, &foo, &tru)
	if err != nil {
		t.Fatal(err)
	} else if one != 1 || two != 2 || foo != "foo" || tru != true {
		t.Error("values were not decoded correctly:", one, two, foo, tru)
	}
}

// TestReadWriteFile tests the ReadFiles and WriteFile functions, which are
// inverses of each other.
func TestReadWriteFile(t *testing.T) {
	// standard
	err := os.MkdirAll(TempDir("encoding"), 0777)
	if err != nil {
		t.Error(err)
	}
	path := TempDir("encoding", t.Name())
	err = WriteFile(path, testStructs[3])
	if err != nil {
		t.Fatal(err)
	}

	var obj test4
	err = ReadFile(path, &obj)
	if err != nil {
		t.Error(err)
	}

	// bad paths
	err = WriteFile("/foo/bar", "baz")
	if err == nil {
		t.Error("expected error, got nil")
	}
	err = ReadFile("/foo/bar", nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// i5-4670K, 9a90f86: 33 MB/s
func BenchmarkEncode(b *testing.B) {
	b.ReportAllocs()
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	for i := 0; i < b.N; i++ {
		buf.Reset()
		for i := range testStructs {
			err := enc.Encode(testStructs[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	b.SetBytes(int64(buf.Len()))
}

// i5-4670K, 9a90f86: 26 MB/s
func BenchmarkDecode(b *testing.B) {
	b.ReportAllocs()
	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	var numBytes int64
	for i := 0; i < b.N; i++ {
		numBytes = 0
		for i := range testEncodings {
			err := Unmarshal(testEncodings[i], emptyStructs[i])
			if err != nil {
				b.Fatal(err)
			}
			numBytes += int64(len(testEncodings[i]))
		}
	}
	b.SetBytes(numBytes)
}

// i5-4670K, 2059112: 44 MB/s
func BenchmarkMarshalAll(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MarshalAll(testStructs...)
	}
	b.SetBytes(int64(len(bytes.Join(testEncodings, nil))))
}

// i5-4670K, 2059112: 36 MB/s
func BenchmarkUnmarshalAll(b *testing.B) {
	b.ReportAllocs()
	var emptyStructs = []interface{}{&test0{}, &test1{}, &test2{}, &test3{}, &test4{}, &test5{}, &test6{}, &test7{}}
	structBytes := bytes.Join(testEncodings, nil)
	for i := 0; i < b.N; i++ {
		err := UnmarshalAll(structBytes, emptyStructs...)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(len(structBytes)))
}

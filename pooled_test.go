package pooled

import "testing"

func skipPoolReuseUnderRace(t *testing.T) {
	t.Helper()
	if testRaceEnabled {
		t.Skip("sync.Pool may drop values under the -race")
	}
}

func TestSlicesGetCapacity(t *testing.T) {
	t.Run("zero value max cap", func(t *testing.T) {
		var p Slices[int]

		tests := []struct {
			capHint int
			wantCap int
		}{
			{capHint: -1, wantCap: sliceMinCap},
			{capHint: 0, wantCap: sliceMinCap},
			{capHint: 1, wantCap: sliceMinCap},
			{capHint: sliceMinCap - 1, wantCap: sliceMinCap},
			{capHint: sliceMinCap, wantCap: sliceMinCap},
			{capHint: sliceMinCap + 1, wantCap: sliceMinCap + 1},
		}

		for _, tt := range tests {
			got := p.Get(tt.capHint)
			if len(got) != 0 {
				t.Fatalf("Get(%d) len = %d, want 0", tt.capHint, len(got))
			}
			if cap(got) != tt.wantCap {
				t.Fatalf("Get(%d) cap = %d, want %d", tt.capHint, cap(got), tt.wantCap)
			}
		}
	})

	t.Run("configured max cap rounds up", func(t *testing.T) {
		p := Slices[int]{MaxCap: 100}

		tests := []struct {
			capHint int
			wantCap int
		}{
			{capHint: 33, wantCap: 64},
			{capHint: 64, wantCap: 64},
			{capHint: 65, wantCap: 128},
			{capHint: 100, wantCap: 128},
			{capHint: 128, wantCap: 128},
			{capHint: 129, wantCap: 129},
		}

		for _, tt := range tests {
			got := p.Get(tt.capHint)
			if len(got) != 0 {
				t.Fatalf("Get(%d) len = %d, want 0", tt.capHint, len(got))
			}
			if cap(got) != tt.wantCap {
				t.Fatalf("Get(%d) cap = %d, want %d", tt.capHint, cap(got), tt.wantCap)
			}
		}
	})
}

func TestSlicesCleanupRunsBeforeClear(t *testing.T) {
	var seen []int
	p := Slices[int]{
		MaxCap: sliceMinCap,
		Clear:  ClearCap,
		Cleanup: func(v []int) {
			seen = append(seen, v...)
		},
	}

	v := make([]int, 3, sliceMinCap)
	copy(v, []int{1, 2, 3})
	p.Put(v)

	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 3 {
		t.Fatalf("Cleanup saw %v, want [1 2 3]", seen)
	}
	for i, x := range v {
		if x != 0 {
			t.Fatalf("v[%d] after Put = %d, want 0", i, x)
		}
	}
}

func TestSlicesClearPoliciesOnReuse(t *testing.T) {
	skipPoolReuseUnderRace(t)

	t.Run("NoClear", func(t *testing.T) {
		p := Slices[int]{MaxCap: sliceMinCap, Clear: NoClear}
		v := make([]int, 1, sliceMinCap)
		v[0] = 42

		p.Put(v)
		got := p.Get(1)
		full := got[:cap(got)]
		if full[0] != 42 {
			t.Fatalf("reused value = %d, want 42", full[0])
		}
	})

	t.Run("ClearLen", func(t *testing.T) {
		p := Slices[int]{MaxCap: sliceMinCap, Clear: ClearLen}
		v := make([]int, 2, sliceMinCap)
		full := v[:cap(v)]
		for i := range full {
			full[i] = 9
		}

		p.Put(v)
		got := p.Get(1)
		full = got[:cap(got)]
		if full[0] != 0 || full[1] != 0 {
			t.Fatalf("first len elements = [%d %d], want [0 0]", full[0], full[1])
		}
		if full[2] != 9 {
			t.Fatalf("element beyond len = %d, want 9", full[2])
		}
	})

	t.Run("ClearCap", func(t *testing.T) {
		p := Slices[int]{MaxCap: sliceMinCap, Clear: ClearCap}
		v := make([]int, 2, sliceMinCap)
		full := v[:cap(v)]
		for i := range full {
			full[i] = 9
		}

		p.Put(v)
		got := p.Get(1)
		full = got[:cap(got)]
		for i, x := range full {
			if x != 0 {
				t.Fatalf("reused element %d = %d, want 0", i, x)
			}
		}
	})
}

func TestSlicesSearchesNearbyLargerBucket(t *testing.T) {
	skipPoolReuseUnderRace(t)

	p := Slices[int]{MaxCap: 1 << 10}
	v := make([]int, 1, 1<<7)
	v[0] = 42

	p.Put(v)
	got := p.Get(sliceMinCap)
	if cap(got) != 1<<7 {
		t.Fatalf("cap = %d, want %d", cap(got), 1<<7)
	}
	if got[:cap(got)][0] != 42 {
		t.Fatalf("reused value = %d, want 42", got[:cap(got)][0])
	}
}

func TestSlicesDoesNotSearchTooFar(t *testing.T) {
	skipPoolReuseUnderRace(t)

	p := Slices[int]{MaxCap: 1 << 10}
	v := make([]int, 1, 1<<(sliceMinShift+sliceMaxDistance+1))
	v[0] = 42

	p.Put(v)
	got := p.Get(sliceMinCap)
	if cap(got) != sliceMinCap {
		t.Fatalf("cap = %d, want %d", cap(got), sliceMinCap)
	}
	if got[:cap(got)][0] == 42 {
		t.Fatalf("Get reused a bucket beyond sliceMaxDistance")
	}
}

func TestSlicesRejectsOutOfRangeCapacities(t *testing.T) {
	skipPoolReuseUnderRace(t)

	t.Run("too small", func(t *testing.T) {
		p := Slices[int]{MaxCap: sliceMinCap}
		v := make([]int, 1, sliceMinCap-1)
		v[0] = 42

		p.Put(v)
		got := p.Get(1)
		if got[:cap(got)][0] == 42 {
			t.Fatalf("Get reused a too-small slice")
		}
	})

	t.Run("too large", func(t *testing.T) {
		p := Slices[int]{MaxCap: sliceMinCap}
		v := make([]int, 1, sliceMinCap*2)
		v[0] = 42

		p.Put(v)
		got := p.Get(sliceMinCap)
		if got[:cap(got)][0] == 42 {
			t.Fatalf("Get reused a slice above MaxCap")
		}
	})

	t.Run("too much slack", func(t *testing.T) {
		const shift = 10

		p := Slices[int]{MaxCap: 1 << (shift + 1)}
		v := make([]int, 1, sliceRetainCaps[shift]+1)
		v[0] = 42

		p.Put(v)
		got := p.Get(1 << shift)
		if got[:cap(got)][0] == 42 {
			t.Fatalf("Get reused a slice above its retain cap")
		}
	})
}

func TestBuffersGetPut(t *testing.T) {
	t.Run("MinCap", func(t *testing.T) {
		p := Buffers{MinCap: 64}
		b := p.Get()
		if b == nil {
			t.Fatalf("Get returned nil")
		}
		if b.Len() != 0 {
			t.Fatalf("Len = %d, want 0", b.Len())
		}
		if b.Cap() < 64 {
			t.Fatalf("Cap = %d, want >= 64", b.Cap())
		}
	})

	t.Run("Put resets", func(t *testing.T) {
		var p Buffers
		b := p.Get()
		b.WriteString("payload")

		p.Put(b)
		if b.Len() != 0 {
			t.Fatalf("Len after Put = %d, want 0", b.Len())
		}
	})

	t.Run("reuse", func(t *testing.T) {
		skipPoolReuseUnderRace(t)

		var p Buffers
		b := p.Get()
		p.Put(b)

		got := p.Get()
		if got != b {
			t.Fatalf("Get did not reuse buffer")
		}
	})

	t.Run("MaxCap rejects", func(t *testing.T) {
		skipPoolReuseUnderRace(t)

		p := Buffers{MaxCap: 32}
		b := p.Get()
		b.Grow(64)
		b.WriteString("payload")

		p.Put(b)
		if b.Len() == 0 {
			t.Fatalf("oversized buffer was reset")
		}

		got := p.Get()
		if got == b {
			t.Fatalf("Get reused an oversized buffer")
		}
	})

	t.Run("nil Put", func(t *testing.T) {
		var p Buffers
		p.Put(nil)
	})
}

func TestMapsGetPut(t *testing.T) {
	t.Run("Get returns usable map", func(t *testing.T) {
		var p Maps[string, int]
		m := p.Get()
		if m == nil {
			t.Fatalf("Get returned nil")
		}
		m["answer"] = 42
		if m["answer"] != 42 {
			t.Fatalf("map returned by Get is not usable")
		}
	})

	t.Run("Cleanup before clear", func(t *testing.T) {
		var seenLen int
		p := Maps[string, int]{
			Cleanup: func(m map[string]int) {
				seenLen = len(m)
				m["cleanup"] = 1
			},
		}
		m := map[string]int{"a": 1}

		p.Put(m)
		if seenLen != 1 {
			t.Fatalf("Cleanup saw len %d, want 1", seenLen)
		}
		if len(m) != 0 {
			t.Fatalf("map len after Put = %d, want 0", len(m))
		}
	})

	t.Run("reuse", func(t *testing.T) {
		skipPoolReuseUnderRace(t)

		var p Maps[string, int]
		m := map[string]int{"a": 1}

		p.Put(m)
		got := p.Get()
		got["probe"] = 1
		if m["probe"] != 1 {
			t.Fatalf("Get did not reuse map")
		}
	})

	t.Run("MaxLen rejects", func(t *testing.T) {
		skipPoolReuseUnderRace(t)

		p := Maps[string, int]{MaxLen: 1}
		m := map[string]int{"a": 1, "b": 2}

		p.Put(m)
		if len(m) != 2 {
			t.Fatalf("rejected map len after Put = %d, want 2", len(m))
		}

		got := p.Get()
		got["probe"] = 1
		if m["probe"] == 1 {
			t.Fatalf("Get reused a map above MaxLen")
		}
	})

	t.Run("Cleanup runs for rejected map", func(t *testing.T) {
		calls := 0
		p := Maps[string, int]{
			MaxLen: 1,
			Cleanup: func(m map[string]int) {
				calls++
			},
		}

		p.Put(map[string]int{"a": 1, "b": 2})
		if calls != 1 {
			t.Fatalf("Cleanup calls = %d, want 1", calls)
		}
	})

	t.Run("nil Put", func(t *testing.T) {
		calls := 0
		p := Maps[string, int]{
			Cleanup: func(m map[string]int) {
				calls++
			},
		}

		p.Put(nil)
		if calls != 0 {
			t.Fatalf("Cleanup calls = %d, want 0", calls)
		}
	})
}

func TestPointersGetPut(t *testing.T) {
	t.Run("New and Init", func(t *testing.T) {
		newCalls := 0
		initCalls := 0
		p := Pointers[int]{
			New: func() *int {
				newCalls++
				v := new(int)
				*v = 40
				return v
			},
			Init: func(v *int) {
				initCalls++
				*v += 2
			},
		}

		v := p.Get()
		if v == nil {
			t.Fatalf("Get returned nil")
		}
		if *v != 42 {
			t.Fatalf("*Get() = %d, want 42", *v)
		}
		if newCalls != 1 {
			t.Fatalf("New calls = %d, want 1", newCalls)
		}
		if initCalls != 1 {
			t.Fatalf("Init calls = %d, want 1", initCalls)
		}
	})

	t.Run("Cleanup before clear", func(t *testing.T) {
		type item struct {
			N int
			S string
		}

		var seen item
		p := Pointers[item]{
			Clear: true,
			Cleanup: func(v *item) {
				seen = *v
			},
		}
		v := &item{N: 7, S: "payload"}

		p.Put(v)
		if seen != (item{N: 7, S: "payload"}) {
			t.Fatalf("Cleanup saw %+v, want original value", seen)
		}
		if *v != (item{}) {
			t.Fatalf("value after Put = %+v, want zero", *v)
		}
	})

	t.Run("reuse and Init", func(t *testing.T) {
		skipPoolReuseUnderRace(t)

		initCalls := 0
		p := Pointers[int]{
			Init: func(v *int) {
				initCalls++
				*v++
			},
		}

		v := p.Get()
		*v = 10
		p.Put(v)

		got := p.Get()
		if got != v {
			t.Fatalf("Get did not reuse pointer")
		}
		if *got != 11 {
			t.Fatalf("reused value after Init = %d, want 11", *got)
		}
		if initCalls != 2 {
			t.Fatalf("Init calls = %d, want 2", initCalls)
		}
	})

	t.Run("nil Put", func(t *testing.T) {
		calls := 0
		p := Pointers[int]{
			Cleanup: func(v *int) {
				calls++
			},
		}

		p.Put(nil)
		if calls != 0 {
			t.Fatalf("Cleanup calls = %d, want 0", calls)
		}
	})
}

func BenchmarkSlicesGetPut(b *testing.B) {
	benchmarks := []struct {
		name  string
		clear SliceClearPolicy
	}{
		{name: "NoClear", clear: NoClear},
		{name: "ClearLen", clear: ClearLen},
		{name: "ClearCap", clear: ClearCap},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			p := Slices[int]{MaxCap: 1 << 20, Clear: bm.clear}
			v := p.Get(1024)
			p.Put(v)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v := p.Get(1024)[:64]
				p.Put(v)
			}
		})
	}
}

func BenchmarkBuffersGetPut(b *testing.B) {
	p := Buffers{MinCap: 1024, MaxCap: 1 << 20}
	v := p.Get()
	p.Put(v)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := p.Get()
		p.Put(v)
	}
}

func BenchmarkMapsGetPut(b *testing.B) {
	p := Maps[string, int]{NewCap: 1024, MaxLen: 1 << 20}
	v := p.Get()
	p.Put(v)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := p.Get()
		p.Put(v)
	}
}

func BenchmarkPointersGetPut(b *testing.B) {
	var p Pointers[int]
	v := p.Get()
	p.Put(v)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := p.Get()
		p.Put(v)
	}
}

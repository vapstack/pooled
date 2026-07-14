# pooled

Typed customizable helpers for pooling entities.

- `Pointers[T]` for `*T`
- `Slices[T]` for `[]T` (using capacity buckets)
- `FlatSlices[T]` for `[]T` (using a single pool)
- `Maps[K, V]` for `map[K]V`
- `Buffers` for `*bytes.Buffer`

Pools are backed by `sync.Pool`.
Reuse is opportunistic: values may be dropped by the Go runtime at any time.

## Pointers

```go
var myPool = pooled.Pointers[MyType]{
    New: func() *MyType {
        // called by Get to allocate a new value (optional)
    },
    Init: func(v *MyType) {
        // called by Get before returning a value (optional)
    },
    Cleanup: func(v *MyType) {
        // called by Put before Clear (optional)
    },
    Clear: true, // zeroes values before returning to the pool (optional)
}

v := myPool.Get()
defer myPool.Put(v)
```

## Slices

`Slices[T]` pools slice backing arrays in power-of-two capacity buckets.

`Get` returns a slice with `len == 0` and `cap >= capHint`.
If `capHint` is larger than `MaxCap` (rounded to power-of-two),
the returned slice is allocated without consulting the pool.
If the target bucket is empty, `Get` tries the next three buckets before
allocating a new slice.

`Put` classifies a slice by its capacity at call time, so a slice may grow while
in use and still be reused from the matching capacity bucket. A retained slice
is returned by `Get` with the actual capacity it had at `Put` time.

`Put` uses the bucket whose base is the largest power of two not exceeding the
slice capacity. It retains every capacity belonging to that bucket, including                                                                                  
capacities above the bucket's base size. A slice is discarded when its
capacity is below the minimum bucket or exceeds `MaxCap`.

```go
var mySlicePool = pooled.Slices[*MyType]{
    MaxCap:  64 << 10,       // default is 32
    Clear:   pooled.NoClear, // default is NoClear
    Cleanup: func(v []*MyType) {
        // optional cleanup
    },
}

s := mySlicePool.Get(10000)
// ...
mySlicePool.Put(s)
```

`MaxCap` is rounded up to the next power of two. `MaxCap <= 32` is treated as 32.

Clearing policy:
- `NoClear` leaves contents unchanged.
- `ClearLen` clears the current length.
- `ClearCap` clears the full capacity.

`NoClear` leaves the backing array unchanged.
This is fastest, but for pointer-containing element types it may keep
references alive while the slice is retained by the pool.

`Cleanup`, when set, is called by `Put` before clearing and before retention
checks. It runs for every `Put` call, including nil slices and slices that will
be discarded.

## Flat slices

`FlatSlices[T]` is a simpler single-pool variant of `Slices[T]`.
`Get` has no capacity hint and returns any available slice with `len == 0`,
regardless of its capacity. When the pool is empty, it allocates a slice
with `NewCap` capacity. `MaxCap <= 32` is treated as 32, larger values are
used as is. `Clear` and `Cleanup` behave as they do for `Slices`.

```go
var flatSlicePool = pooled.FlatSlices[byte]{
    NewCap: 4 << 10,
    MaxCap: 64 << 10,
    Clear:  pooled.NoClear,
}

s := flatSlicePool.Get()
// ...
flatSlicePool.Put(s)
```

## Shared slice pools

The package includes shared pools for common scalar slice types:

```go
b := pooled.GetUint64Slice(4<<10)
// ...
pooled.ReleaseUint64Slice(b)
```

Helpers are available for `bool`, `byte`, `int`, `int32`, `int64`, `uint`,
`uint32`, `uint64`, `float32`, `float64`, and `string`. Only string slices are
cleared before retention. These pools use the bucketed `Slices` implementation.

## Buffers

`Buffers` pools `*bytes.Buffer` values. `Put` resets the buffer before retaining
it and discards buffers whose capacity exceeds `MaxCap`.

```go
var buffers = pooled.Buffers{
    MinCap: 1024,
    MaxCap: 1 << 20,
}

buf := buffers.Get()
defer buffers.Put(buf)
// ...
```

## Maps

`Maps[K, V]` pools maps and clears them before retention. `MaxLen` limits the
maximum map length kept in the pool.

```go
var labels = pooled.Maps[string, string]{
    NewCap: 16,
    MaxLen: 256,
    Cleanup: func(map[string]string) {
        // ...
    }
}
```

## Benchmarks

```
BenchmarkSlicesGetPut/NoClear-16         52960077        22.47 ns/op       0 B/op       0 allocs/op
BenchmarkSlicesGetPut/ClearLen-16        41668288        27.76 ns/op       0 B/op       0 allocs/op
BenchmarkSlicesGetPut/ClearCap-16         9544119       125.9 ns/op        0 B/op       0 allocs/op
BenchmarkSlicesGetPutGrown-16            33940534        34.74 ns/op       0 B/op       0 allocs/op
BenchmarkFlatSlicesGetPut-16             36189553        33.15 ns/op       0 B/op       0 allocs/op
BenchmarkBuffersGetPut-16                67194806        17.42 ns/op       0 B/op       0 allocs/op
BenchmarkMapsGetPut-16                   56957324        21.38 ns/op       0 B/op       0 allocs/op
BenchmarkPointersGetPut-16               65206532        18.29 ns/op       0 B/op       0 allocs/op
```

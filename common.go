package pooled

var (
	bytePool    = Slices[byte]{MaxCap: 1 << 20, Clear: NoClear}
	boolPool    = Slices[bool]{MaxCap: 1 << 20, Clear: NoClear}
	intPool     = Slices[int]{MaxCap: 1 << 20, Clear: NoClear}
	int32Pool   = Slices[int32]{MaxCap: 1 << 20, Clear: NoClear}
	int64Pool   = Slices[int64]{MaxCap: 1 << 20, Clear: NoClear}
	uintPool    = Slices[uint]{MaxCap: 1 << 20, Clear: NoClear}
	uint32Pool  = Slices[uint32]{MaxCap: 1 << 20, Clear: NoClear}
	uint64Pool  = Slices[uint64]{MaxCap: 1 << 20, Clear: NoClear}
	float32Pool = Slices[float32]{MaxCap: 1 << 20, Clear: NoClear}
	float64Pool = Slices[float64]{MaxCap: 1 << 20, Clear: NoClear}
	stringPool  = Slices[string]{MaxCap: 64 << 10, Clear: ClearCap}
)

// GetBoolSlice returns a bool slice with len 0 and cap at least capHint.
func GetBoolSlice(capHint int) []bool { return boolPool.Get(capHint) }

// ReleaseBoolSlice returns s to the shared bool slice pool.
func ReleaseBoolSlice(s []bool) { boolPool.Put(s) }

// GetIntSlice returns an int slice with len 0 and cap at least capHint.
func GetIntSlice(capHint int) []int { return intPool.Get(capHint) }

// ReleaseIntSlice returns s to the shared int slice pool.
func ReleaseIntSlice(s []int) { intPool.Put(s) }

// GetInt32Slice returns an int32 slice with len 0 and cap at least capHint.
func GetInt32Slice(capHint int) []int32 { return int32Pool.Get(capHint) }

// ReleaseInt32Slice returns s to the shared int32 slice pool.
func ReleaseInt32Slice(s []int32) { int32Pool.Put(s) }

// GetInt64Slice returns an int64 slice with len 0 and cap at least capHint.
func GetInt64Slice(capHint int) []int64 { return int64Pool.Get(capHint) }

// ReleaseInt64Slice returns s to the shared int64 slice pool.
func ReleaseInt64Slice(s []int64) { int64Pool.Put(s) }

// GetUintSlice returns a uint slice with len 0 and cap at least capHint.
func GetUintSlice(capHint int) []uint { return uintPool.Get(capHint) }

// ReleaseUintSlice returns s to the shared uint slice pool.
func ReleaseUintSlice(s []uint) { uintPool.Put(s) }

// GetUint32Slice returns a uint32 slice with len 0 and cap at least capHint.
func GetUint32Slice(capHint int) []uint32 { return uint32Pool.Get(capHint) }

// ReleaseUint32Slice returns s to the shared uint32 slice pool.
func ReleaseUint32Slice(s []uint32) { uint32Pool.Put(s) }

// GetUint64Slice returns a uint64 slice with len 0 and cap at least capHint.
func GetUint64Slice(capHint int) []uint64 { return uint64Pool.Get(capHint) }

// ReleaseUint64Slice returns s to the shared uint64 slice pool.
func ReleaseUint64Slice(s []uint64) { uint64Pool.Put(s) }

// GetStringSlice returns a string slice with len 0 and cap at least capHint.
func GetStringSlice(capHint int) []string { return stringPool.Get(capHint) }

// ReleaseStringSlice returns s to the shared string slice pool.
func ReleaseStringSlice(s []string) { stringPool.Put(s) }

// GetByteSlice returns a byte slice with len 0 and cap at least capHint.
func GetByteSlice(capHint int) []byte { return bytePool.Get(capHint) }

// ReleaseByteSlice returns s to the shared byte slice pool.
func ReleaseByteSlice(s []byte) { bytePool.Put(s) }

// GetFloat32Slice returns a float32 slice with len 0 and cap at least capHint.
func GetFloat32Slice(capHint int) []float32 { return float32Pool.Get(capHint) }

// ReleaseFloat32Slice returns s to the shared float32 slice pool.
func ReleaseFloat32Slice(s []float32) { float32Pool.Put(s) }

// GetFloat64Slice returns a float64 slice with len 0 and cap at least capHint.
func GetFloat64Slice(capHint int) []float64 { return float64Pool.Get(capHint) }

// ReleaseFloat64Slice returns s to the shared float64 slice pool.
func ReleaseFloat64Slice(s []float64) { float64Pool.Put(s) }

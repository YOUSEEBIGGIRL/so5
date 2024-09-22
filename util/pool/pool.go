package pool

import "sync"

var ByteSlice2 = sync.Pool{New: func() any { return make([]byte, 2) }}
var ByteSlice255 = sync.Pool{New: func() any { return make([]byte, 255) }}

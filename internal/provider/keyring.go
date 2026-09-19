package provider

import "sync/atomic"

func nextKey(keys []string, counter *uint64) string {
	if len(keys) == 0 {
		return ""
	}

	idx := atomic.AddUint64(counter, 1) - 1

	return keys[idx%uint64(len(keys))]
}

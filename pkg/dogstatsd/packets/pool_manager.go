// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package packets

import (
	"sync"
	"sync/atomic"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type genericPool interface {
	Get() interface{}
	Put(x interface{})
}

// PoolManager helps manage sync pools so multiple references to the same pool objects may be held.
type PoolManager struct {
	pool genericPool
	refs sync.Map

	passthru int32

	sync.RWMutex
}

// NewPoolManager creates a PoolManager to manage the underlying genericPool.
func NewPoolManager(gp genericPool) *PoolManager {
	return &PoolManager{
		pool:     gp,
		passthru: int32(1),
	}
}

// Get gets an object from the pool.
func (p *PoolManager) Get() interface{} {
	return p.pool.Get()
}

// Put declares intent to return an object to the pool. In passthru mode the object is immediately
// returned to the pool, otherwise we wait until the object is put by all (only 2 currently supported)
// reference holders before actually returning it to the object pool.
func (p *PoolManager) Put(x interface{}) {

	if p.IsPassthru() {
		p.pool.Put(x)
		return
	}

	// This lock is not to guard the map, it's here to
	// avoid adding items to the map while flushing.
	p.RLock()

	var ref interface{}

	switch v := x.(type) {
	case []uint8:
		ref = &v
	default:
		ref = v
	}

	// TODO: use LoadAndDelete when go 1.15 is introduced
	_, loaded := p.refs.Load(ref)
	if loaded {
		// reference exists, put back.
		p.refs.Delete(ref)
		p.pool.Put(x)
		log.Debugf("Returning type: %T to packet pool.", x)
	} else {
		// reference does not exist, account.
		p.refs.Store(ref, struct{}{})
	}

	// relatively hot path so not deferred
	p.RUnlock()
}

// IsPassthru returns a boolean telling us if the PoolManager is in passthru mode or not.
func (p *PoolManager) IsPassthru() bool {
	return atomic.LoadInt32(&(p.passthru)) != 0
}

// SetPassthru sets the passthru mode to the specified value. It will flush the sccounting before
// enabling passthru mode.
func (p *PoolManager) SetPassthru(b bool) {
	if b {
		atomic.StoreInt32(&(p.passthru), 1)
		p.Flush()
	} else {
		atomic.StoreInt32(&(p.passthru), 0)
	}
}

// Count returns the number of elements accounted by the PoolManager.
func (p *PoolManager) Count() int {
	p.RLock()
	defer p.RUnlock()

	size := 0
	p.refs.Range(func(k, v interface{}) bool {
		size++
		return true
	})

	return size
}

// Flush flushes all objects back to the object pool, and stops tracking any pending objects.
func (p *PoolManager) Flush() {
	p.Lock()
	defer p.Unlock()

	p.refs.Range(func(k, v interface{}) bool {
		var ref interface{}

		switch v := k.(type) {
		case *[]uint8:
			ref = *v
		default:
			ref = v
		}
		p.pool.Put(ref)
		p.refs.Delete(k)
		return true
	})
}

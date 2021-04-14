// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"sync"

	"github.com/DataDog/datadog-agent/pkg/trace/export/sampler"
)

const defaultServiceRateKey = "service:,env:"

// serviceKeyCatalog reverse-maps service signatures to their generated hashes for
// easy look up.
type serviceKeyCatalog struct {
	mu     sync.Mutex
	lookup map[sampler.ServiceSignature]sampler.Signature
}

// newServiceLookup returns a new serviceKeyCatalog.
func newServiceLookup() *serviceKeyCatalog {
	return &serviceKeyCatalog{
		lookup: make(map[sampler.ServiceSignature]sampler.Signature),
	}
}

func (cat *serviceKeyCatalog) register(svcSig sampler.ServiceSignature) sampler.Signature {
	hash := svcSig.Hash()
	cat.mu.Lock()
	cat.lookup[svcSig] = hash
	cat.mu.Unlock()
	return hash
}

// ratesByService returns a map of service signatures mapping to the rates identified using
// the signatures.
func (cat *serviceKeyCatalog) ratesByService(rates map[sampler.Signature]float64, totalScore float64) map[sampler.ServiceSignature]float64 {
	rbs := make(map[sampler.ServiceSignature]float64, len(rates)+1)
	cat.mu.Lock()
	defer cat.mu.Unlock()
	for key, sig := range cat.lookup {
		if rate, ok := rates[sig]; ok {
			rbs[key] = rate
		} else {
			delete(cat.lookup, key)
		}
	}
	rbs[sampler.ServiceSignature{}] = totalScore
	return rbs
}

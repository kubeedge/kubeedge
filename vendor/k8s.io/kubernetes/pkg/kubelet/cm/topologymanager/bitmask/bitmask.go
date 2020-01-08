/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bitmask

import (
	"fmt"
	"math/bits"
)

// BitMask interface allows hint providers to create BitMasks for TopologyHints
type BitMask interface {
	Add(bits ...int) error
	Remove(bits ...int) error
	And(masks ...BitMask)
	Or(masks ...BitMask)
	Clear()
	Fill()
	IsEqual(mask BitMask) bool
	IsEmpty() bool
	IsSet(bit int) bool
	IsNarrowerThan(mask BitMask) bool
	String() string
	Count() int
	GetBits() []int
}

type bitMask uint64

// NewEmptyBitMask creates a new, empty BitMask
func NewEmptyBitMask() BitMask {
	s := bitMask(0)
	return &s
}

// NewBitMask creates a new BitMask
func NewBitMask(bits ...int) (BitMask, error) {
	s := bitMask(0)
	err := (&s).Add(bits...)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Add adds the bits with topology affinity to the BitMask
func (s *bitMask) Add(bits ...int) error {
	mask := *s
	for _, i := range bits {
		if i < 0 || i >= 64 {
			return fmt.Errorf("bit number must be in range 0-63")
		}
		mask |= 1 << uint64(i)
	}
	*s = mask
	return nil
}

// Remove removes specified bits from BitMask
func (s *bitMask) Remove(bits ...int) error {
	mask := *s
	for _, i := range bits {
		if i < 0 || i >= 64 {
			return fmt.Errorf("bit number must be in range 0-63")
		}
		mask &^= 1 << uint64(i)
	}
	*s = mask
	return nil
}

// And performs and operation on all bits in masks
func (s *bitMask) And(masks ...BitMask) {
	for _, m := range masks {
		*s &= *m.(*bitMask)
	}
}

// Or performs or operation on all bits in masks
func (s *bitMask) Or(masks ...BitMask) {
	for _, m := range masks {
		*s |= *m.(*bitMask)
	}
}

// Clear resets all bits in mask to zero
func (s *bitMask) Clear() {
	*s = 0
}

// Fill sets all bits in mask to one
func (s *bitMask) Fill() {
	*s = bitMask(^uint64(0))
}

// IsEmpty checks mask to see if all bits are zero
func (s *bitMask) IsEmpty() bool {
	return *s == 0
}

// IsSet checks bit in mask to see if bit is set to one
func (s *bitMask) IsSet(bit int) bool {
	if bit < 0 || bit >= 64 {
		return false
	}
	return (*s & (1 << uint64(bit))) > 0
}

// IsEqual checks if masks are equal
func (s *bitMask) IsEqual(mask BitMask) bool {
	return *s == *mask.(*bitMask)
}

// IsNarrowerThan checks if one mask is narrower than another.
//
// A mask is said to be "narrower" than another if it has lets bits set. If the
// same number of bits are set in both masks, then the mask with more
// lower-numbered bits set wins out.
func (s *bitMask) IsNarrowerThan(mask BitMask) bool {
	if s.Count() == mask.Count() {
		if *s < *mask.(*bitMask) {
			return true
		}
	}
	return s.Count() < mask.Count()
}

// String converts mask to string
func (s *bitMask) String() string {
	return fmt.Sprintf("%064b", *s)
}

// Count counts number of bits in mask set to one
func (s *bitMask) Count() int {
	return bits.OnesCount64(uint64(*s))
}

// Getbits returns each bit number with bits set to one
func (s *bitMask) GetBits() []int {
	var bits []int
	for i := uint64(0); i < 64; i++ {
		if (*s & (1 << i)) > 0 {
			bits = append(bits, int(i))
		}
	}
	return bits
}

// And is a package level implementation of 'and' between first and masks
func And(first BitMask, masks ...BitMask) BitMask {
	s := *first.(*bitMask)
	s.And(masks...)
	return &s
}

// Or is a package level implementation of 'or' between first and masks
func Or(first BitMask, masks ...BitMask) BitMask {
	s := *first.(*bitMask)
	s.Or(masks...)
	return &s
}

// IterateBitMasks iterates all possible masks from a list of bits,
// issuing a callback on each mask.
func IterateBitMasks(bits []int, callback func(BitMask)) {
	var iterate func(bits, accum []int, size int)
	iterate = func(bits, accum []int, size int) {
		if len(accum) == size {
			mask, _ := NewBitMask(accum...)
			callback(mask)
			return
		}
		for i := range bits {
			iterate(bits[i+1:], append(accum, bits[i]), size)
		}
	}

	for i := 1; i <= len(bits); i++ {
		iterate(bits, []int{}, i)
	}
}

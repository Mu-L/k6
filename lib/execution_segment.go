/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2019 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package lib

import (
	"encoding"
	"fmt"
	"math/big"
	"strings"
)

// ExecutionSegment represents a (start, end] partition of the total execution
// work for a specific test. For example, if we want the split the execution of a
// test in 2 different parts, we can split it in two segments (0, 0.5] and (0,5, 1].
//
// We use rational numbers so it's easier to verify the correctness and easier to
// reason about portions of indivisible things, like VUs. This way, we can easily
// split a test in thirds (i.e. (0, 1/3], (1/3, 2/3], (2/3, 1]), without fearing
// that we'll lose a VU along the way...
//
// The most important part is that if work is split between multiple k6 instances,
// each k6 instance can precisely and reproducably calculate its share of the work,
// just by knowing its own segment. There won't be a need to schedule the
// execution from a master node, or to even know how many other k6 instances are
// running!
type ExecutionSegment struct {
	// 0 <= from < to <= 1
	from *big.Rat
	to   *big.Rat

	// derived, equals to-from, but pre-calculated here for speed
	length *big.Rat
}

// Ensure we implement those interfaces
var _ encoding.TextUnmarshaler = &ExecutionSegment{}
var _ fmt.Stringer = &ExecutionSegment{}

// Helpful "constants" so we don't initialize them in every function call
var zeroRat, oneRat = big.NewRat(0, 1), big.NewRat(1, 1) //nolint:gochecknoglobals
var oneBigInt, twoBigInt = big.NewInt(1), big.NewInt(2)  //nolint:gochecknoglobals

// NewExecutionSegment validates the supplied arguments (basically, that 0 <=
// from < to <= 1) and either returns an error, or it returns a
// fully-initialized and usable execution segment.
func NewExecutionSegment(from, to *big.Rat) (*ExecutionSegment, error) {
	if from.Cmp(zeroRat) < 0 {
		return nil, fmt.Errorf("segment start value should be at least 0 but was %s", from.FloatString(2))
	}
	if from.Cmp(to) >= 0 {
		return nil, fmt.Errorf("segment start(%s) should be less than its end(%s)", from.FloatString(2), to.FloatString(2))
	}
	if to.Cmp(oneRat) > 0 {
		return nil, fmt.Errorf("segment end value shouldn't be more than 1 but was %s", to.FloatString(2))
	}
	return &ExecutionSegment{
		from:   from,
		to:     to,
		length: new(big.Rat).Sub(to, from),
	}, nil
}

// stringToRat is a helper function that tries to convert a string to a rational
// number while allowing percentage, decimal, and fraction values.
func stringToRat(s string) (*big.Rat, error) {
	if strings.HasSuffix(s, "%") {
		num, ok := new(big.Int).SetString(strings.TrimSuffix(s, "%"), 10)
		if !ok {
			return nil, fmt.Errorf("'%s' is not a valid percentage", s)
		}
		return new(big.Rat).SetFrac(num, big.NewInt(100)), nil
	}
	rat, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a valid percentage, decimal, fraction or interval value", s)
	}
	return rat, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface, so that
// execution segments can be specified as CLI flags, environment variables, and
// JSON strings.
//
// We are able to parse both single percentage/float/fraction values, and actual
// (from; to] segments. For the single values, we just treat them as the
// beginning segment - thus the execution segment can be used as a shortcut for
// quickly running an arbitrarily scaled-down version of a test.
//
// The parsing logic is that values with a colon, i.e. ':', are full segments:
//  `1/2:3/4`, `0.5:0.75`, `50%:75%`, and even `2/4:75%` should be (1/2, 3/4]
// And values without a hyphen are the end of a first segment:
//  `20%`, `0.2`,  and `1/5` should be converted to (0, 1/5]
// empty values should probably be treated as "1", i.e. the whole execution
func (es *ExecutionSegment) UnmarshalText(text []byte) (err error) {
	from := zeroRat
	toStr := string(text)
	if strings.ContainsRune(toStr, ':') {
		fromToStr := strings.SplitN(toStr, ":", 2)
		toStr = fromToStr[1]
		if from, err = stringToRat(fromToStr[0]); err != nil {
			return err
		}
	}

	to, err := stringToRat(toStr)
	if err != nil {
		return err
	}

	segment, err := NewExecutionSegment(from, to)
	if err != nil {
		return err
	}
	*es = *segment
	return nil
}

func (es *ExecutionSegment) String() string {
	if es == nil {
		return "0:1"
	}
	return es.from.RatString() + ":" + es.to.RatString()
}

// MarshalText implements the encoding.TextMarshaler interface, so is used for
// text and JSON encoding of the execution segment.
func (es *ExecutionSegment) MarshalText() ([]byte, error) {
	if es == nil {
		return nil, nil
	}
	return []byte(es.String()), nil
}

// FloatLength is a helper method for getting some more human-readable
// information about the execution segment.
func (es *ExecutionSegment) FloatLength() float64 {
	if es == nil {
		return 1.0
	}
	res, _ := es.length.Float64()
	return res
}

// Split evenly dividies the execution segment into the specified number of
// equal consecutive execution sub-segments.
func (es *ExecutionSegment) Split(numParts int64) ([]*ExecutionSegment, error) {
	if numParts < 1 {
		return nil, fmt.Errorf("the number of parts should be at least 1, %d received", numParts)
	}

	from, to := zeroRat, oneRat
	if es != nil {
		from, to = es.from, es.to
	}

	increment := new(big.Rat).Sub(to, from)
	increment.Denom().Mul(increment.Denom(), big.NewInt(numParts))

	results := make([]*ExecutionSegment, numParts)
	for i := int64(0); i < numParts; i++ {
		segmentTo := new(big.Rat).Add(from, increment)
		segment, err := NewExecutionSegment(from, segmentTo)
		if err != nil {
			return nil, err
		}
		results[i] = segment
		from = segmentTo
	}

	if from.Cmp(to) != 0 {
		return nil, fmt.Errorf("expected %s and %s to be equal", from, to)
	}

	return results, nil
}

//TODO: add a NewFromString() method

// Equal returns true only if the two execution segments have the same from and
// to values.
func (es *ExecutionSegment) Equal(other *ExecutionSegment) bool {
	if es == other {
		return true
	}
	thisFrom, otherFrom, thisTo, otherTo := zeroRat, zeroRat, oneRat, oneRat
	if es != nil {
		thisFrom, thisTo = es.from, es.to
	}
	if other != nil {
		otherFrom, otherTo = other.from, other.to
	}
	return thisFrom.Cmp(otherFrom) == 0 && thisTo.Cmp(otherTo) == 0
}

// SubSegment returns a new execution sub-segment - if a is (1/2:1] and b is
// (0:1/2], then a.SubSegment(b) will return a new segment (1/2, 3/4].
//
// The basic formula for c = a.SubSegment(b) is:
//    c.from = a.from + b.from * (a.to - a.from)
//    c.to = c.from + (b.to - b.from) * (a.to - a.from)
func (es *ExecutionSegment) SubSegment(child *ExecutionSegment) *ExecutionSegment {
	if child == nil {
		return es // 100% sub-segment is the original segment
	}

	parentFrom, parentLength := zeroRat, oneRat
	if es != nil {
		parentFrom, parentLength = es.from, es.length
	}

	resultFrom := new(big.Rat).Mul(parentLength, child.from)
	resultFrom.Add(resultFrom, parentFrom)

	resultLength := new(big.Rat).Mul(parentLength, child.length)
	return &ExecutionSegment{
		from:   resultFrom,
		length: resultLength,
		to:     new(big.Rat).Add(resultFrom, resultLength),
	}
}

// helper function for rounding (up) of rational numbers to big.Int values
func roundUp(rat *big.Rat) *big.Int {
	quo, rem := new(big.Int).QuoRem(rat.Num(), rat.Denom(), new(big.Int))

	if rem.Mul(rem, twoBigInt).Cmp(rat.Denom()) >= 0 {
		return quo.Add(quo, oneBigInt)
	}
	return quo
}

// Scale proportionally scales the supplied value, according to the execution
// segment's position and size of the work.
func (es *ExecutionSegment) Scale(value int64) int64 {
	if es == nil { // no execution segment, i.e. 100%
		return value
	}
	// Instead of the first proposal that used remainders and floor:
	//    floor( (value * from) % 1 + value * length )
	// We're using an alternative approach with rounding that (hopefully) has
	// the same properties, but it's simpler and has better precision:
	//    round( (value * from) - round(value * from) + (value * (to - from)) )?
	// which reduces to:
	//    round( (value * to) - round(value * from) )?

	toValue := big.NewRat(value, 1)
	toValue.Mul(toValue, es.to)

	fromValue := big.NewRat(value, 1)
	fromValue.Mul(fromValue, es.from)

	toValue.Sub(toValue, new(big.Rat).SetFrac(roundUp(fromValue), oneBigInt))

	return roundUp(toValue).Int64()
}

// InPlaceScaleRat scales rational numbers in-place - it changes the passed
// argument (and also returns it, to allow for chaining, like many other big.Rat
// methods).
func (es *ExecutionSegment) InPlaceScaleRat(value *big.Rat) *big.Rat {
	if es == nil { // no execution segment, i.e. 100%
		return value
	}
	return value.Mul(value, es.length)
}

// CopyScaleRat scales rational numbers without changing them - creates a new
// bit.Rat object and uses it for the calculation.
func (es *ExecutionSegment) CopyScaleRat(value *big.Rat) *big.Rat {
	if es == nil { // no execution segment, i.e. 100%
		return value
	}
	return new(big.Rat).Mul(value, es.length)
}
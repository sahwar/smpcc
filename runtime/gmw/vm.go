package gmw

import "fmt"
import "math/big"
import "crypto/rand"

type Io interface {
	Id() int
	GetInput() uint32

	Open1(bool) bool
	Open8(uint8) uint8
	Open32(uint32) uint32

	Broadcast1(bool)
	Broadcast8(uint8)
	Broadcast32(uint32)

	Receive1(party int) bool
	Receive8(party int) uint8
	Receive32(party int) uint32

	Triple1() (a, b, c bool)
	Triple8() (a, b, c uint8)
	Triple32() (a, b, c uint32)
}

func xor(x, y bool) bool {
	return (x && !y) || (y && !x)
}

func Xor1(io Io, x, y bool) bool {
	return xor(x, y)
}

func Xor8(io Io, x, y uint8) uint8 {
	return x ^ y
}

func Xor32(io Io, x, y uint32) uint32 {
	return x ^ y
}

func And1(io Io, x, y bool) bool {
	a, b, c := io.Triple1()
	d := io.Open1(xor(x, a))
	e := io.Open1(xor(y, b))
	if io.Id() == 0 {
		return xor(c, xor(d && b, xor(e && a, d && e)))
	} else {
		return xor(c, xor(d && b, e && a))
	}
}

func And8(io Io, x, y uint8) uint8 {
	a, b, c := io.Triple8()
	d := io.Open8(x ^ a)
	e := io.Open8(y ^ b)
	if io.Id() == 0 {
		return c ^ d&b ^ e&a ^ d&e
	} else {
		return c ^ d&b ^ e&a
	}
}

func And32(io Io, x, y uint32) uint32 {
	a, b, c := io.Triple32()
	d := io.Open32(x ^ a)
	e := io.Open32(y ^ b)
	if io.Id() == 0 {
		return c ^ d&b ^ e&a ^ d&e
	} else {
		return c ^ d&b ^ e&a
	}
}

func Not1(io Io, a bool) bool {
	if io.Id() == 0 {
		return !a
	}
	return a
}

func Not8(io Io, a uint8) uint8 {
	if io.Id() == 0 {
		return 0xff ^ a
	}
	return a
}

func Not32(io Io, a uint32) uint32 {
	if io.Id() == 0 {
		return 0xffffffff ^ a
	}
	return a
}

func Add8(io Io, a, b uint8) uint8 {
	var result uint8 = 0
	var a0, b0 bool = (a & 1) > 0, (b & 1) > 0
	if xor(a0, b0) {
		result |= 1
	}
	c := a0 && b0 /* carry bit */
	for i := uint8(2); i > 0; i = i * 2 {
		ai := (a & i) > 0
		bi := (b & i) > 0
		/* compute the result bit */
		bi_xor_c := xor(bi, c)
		if xor(ai, bi_xor_c) {
			result |= i
		}
		/* compute the carry bit. */
		c = xor(c, And1(io, xor(ai, c), bi_xor_c))
	}
	return result
}

func Add32(io Io, a, b uint32) uint32 {
	var result uint32 = 0
	var a0, b0 bool = (a & 1) > 0, (b & 1) > 0
	if xor(a0, b0) {
		result |= 1
	}
	c := a0 && b0 /* carry bit */
	for i := uint32(2); i > 0; i = i * 2 {
		ai := (a & i) > 0
		bi := (b & i) > 0
		/* compute the result bit */
		bi_xor_c := xor(bi, c)
		if xor(ai, bi_xor_c) {
			result |= i
		}
		/* compute the carry bit. */
		c = xor(c, And1(io, xor(ai, c), bi_xor_c))
	}
	return result
}

func Sub8(io Io, a, b uint8) uint8 {
	var result uint8 = 0
	var a0, b0 bool = (a & 1) > 0, (b & 1) > 0
	if xor(a0, b0) {
		result |= 1
	}
	c := xor(a0, And1(io, Not1(io, a0), Not1(io, b0))) /* carry bit */
	for i := uint8(2); i > 0; i = i * 2 {
		ai := (a & i) > 0
		bi := (b & i) > 0
		/* compute the result bit */
		bi_xor_c := xor(bi, c)
		if Not1(io, xor(ai, bi_xor_c)) {
			result |= i
		}
		/* compute the carry bit. */
		c = xor(ai, And1(io, xor(ai, c), bi_xor_c))
	}
	return result
}

func Sub32(io Io, a, b uint32) uint32 {
	var result uint32 = 0
	var a0, b0 bool = (a & 1) > 0, (b & 1) > 0
	if xor(a0, b0) {
		result |= 1
	}
	c := xor(a0, And1(io, Not1(io, a0), Not1(io, b0))) /* carry bit */
	for i := uint32(2); i > 0; i = i * 2 {
		ai := (a & i) > 0
		bi := (b & i) > 0
		/* compute the result bit */
		bi_xor_c := xor(bi, c)
		if Not1(io, xor(ai, bi_xor_c)) {
			result |= i
		}
		/* compute the carry bit. */
		c = xor(ai, And1(io, xor(ai, c), bi_xor_c))
	}
	return result
}

/* mocked-up implementation of Io */
type X struct {
	id        int /* id of party, range is 0..n-1 */
	triples32 []struct{ a, b, c uint32 }
	triples8  []struct{ a, b, c uint8 }
	triples1  []struct{ a, b, c bool }
	rchannels []chan uint32 /* channels for reading from other parties */
	wchannels []chan uint32 /* channels for writing to other parties */
	inputs    []uint32      /* inputs of this party */
}

func (x X) Id() int {
	return x.id
}

func (x X) Triple32() (a, b, c uint32) {
	result := x.triples32[0]
	x.triples32 = x.triples32[1:]
	return result.a, result.b, result.c
}

func (x X) Open32(s uint32) uint32 {
	x.Broadcast32(s)
	result := s
	id := x.Id()
	for i := range x.rchannels {
		if i == id {
			continue
		}
		result ^= x.Receive32(i)
	}
	return result
}

func (x X) Broadcast32(n uint32) {
	id := x.Id()
	fmt.Printf("%d: BROADCAST 0x%08x\n", id, n)
	for i, ch := range x.wchannels {
		if i == id {
			continue
		}
		go func(ch chan<- uint32, i int) {
			ch <- n
			fmt.Printf("%d -- 0x%08x -> %d\n", id, n, i)
		}(ch, i)
	}
}

func (x X) Receive32(party int) uint32 {
	id := x.Id()
	if party == id {
		return 0
	}
	ch := x.rchannels[party]
	result := <-ch
	fmt.Printf("%d <- 0x%08x -- %d\n", id, result, party)
	return result
}

func (x X) Triple8() (a, b, c uint8) {
	if len(x.triples8) == 0 {
		a32, b32, c32 := x.Triple32()
		x.triples8 = []struct{ a, b, c uint8 }{{uint8(a32 >> 0), uint8(b32 >> 0), uint8(c32 >> 0)},
			{uint8(a32 >> 8), uint8(b32 >> 8), uint8(c32 >> 8)},
			{uint8(a32 >> 16), uint8(b32 >> 16), uint8(c32 >> 16)},
			{uint8(a32 >> 24), uint8(b32 >> 24), uint8(c32 >> 24)}}
	}
	result := x.triples8[0]
	x.triples8 = x.triples8[1:]
	return result.a, result.b, result.c
}

func (x X) Open8(s uint8) uint8 {
	x.Broadcast8(s)
	result := s
	id := x.Id()
	for i := range x.rchannels {
		if i == id {
			continue
		}
		result ^= x.Receive8(i)
	}
	return result
}

func (x X) Broadcast8(n uint8) {
	id := x.Id()
	fmt.Printf("%d: BROADCAST 0x%02x\n", id, n)
	for i, ch := range x.wchannels {
		if i == id {
			continue
		}
		go func(ch chan<- uint32, i int) {
			ch <- uint32(n)
			fmt.Printf("%d -- 0x%02x -> %d\n", id, n, i)
		}(ch, i)
	}
}

func (x X) Receive8(party int) uint8 {
	id := x.Id()
	if party == id {
		return 0
	}
	ch := x.rchannels[party]
	result := <-ch
	fmt.Printf("%d <- 0x%02x -- %d\n", id, result, party)
	return uint8(result)
}

func (x X) Triple1() (a, b, c bool) {
	if len(x.triples1) == 0 {
		a32, b32, c32 := x.Triple32()
		x.triples1 = make([]struct{ a, b, c bool }, 32)
		for i := range x.triples1 {
			ui := uint(i)
			x.triples1[i] = struct{ a, b, c bool }{0 < (1 & (a32 >> ui)), 0 < (1 & (b32 >> ui)), 0 < (1 & (c32 >> ui))}
		}
	}
	result := x.triples1[0]
	x.triples1 = x.triples1[1:]
	return result.a, result.b, result.c
}

//func (x X) Triple1b() (a, b, c bool) {
//	a8, b8, c8 := x.Triple1()
//	a, b, c = (a8 > 0), (b8 > 0), (c8 > 0)
//	return
//}

func (x X) Open1(s bool) bool {
	x.Broadcast1(s)
	result := s
	id := x.Id()
	for i := range x.rchannels {
		if i == id {
			continue
		}
		result = xor(result, x.Receive1(i))
	}
	return result
}

func (x X) Broadcast1(n bool) {
	var n32 uint32 = 0
	if n {
		n32 = 1
	}
	id := x.Id()
	fmt.Printf("%d: BROADCAST 0x%1x\n", id, n32)
	for i, ch := range x.wchannels {
		if i == id {
			continue
		}
		go func(ch chan<- uint32, i int) {
			ch <- n32
			fmt.Printf("%d -- 0x%1x -> %d\n", id, n32, i)
		}(ch, i)
	}
}

func (x X) Receive1(party int) bool {
	id := x.Id()
	if party == id {
		return false
	}
	ch := x.rchannels[party]
	result := <-ch
	fmt.Printf("%d <- 0x%1x -- %d\n", id, result, party)
	return result > 0
}

func (x X) GetInput() uint32 {
	result := x.inputs[0]
	x.inputs = x.inputs[1:]
	return result
}

func Input32(io Io, party int) uint32 {
	id := io.Id()
	if id == party {
		X := io.GetInput()
		return X
	} else {
		return 0
	}
}

func Output32(io Io, x uint32) uint32 {
	result := io.Open32(x)
	fmt.Printf("%d: RESULT 0x%08x\n", io.Id(), result)
	return result
}

func rand32() uint32 {
	max := big.NewInt(1 << 32)
	x, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic("Error: random number generation")
	}
	return uint32(x.Int64())
}

/* return a slice of n random uint32 values that ^ to x */
func split_uint32(x uint32, n int) []uint32 {
	x0 := x
	if n <= 0 {
		panic("Error: split")
	}
	result := make([]uint32, n)
	for i := 1; i < n; i++ {
		xi := rand32()
		x ^= xi
		result[i] = xi
	}
	result[0] = x
	var y uint32 = 0
	for _, s := range result {
		y ^= s
	}
	if y != x0 {
		panic("NOT EQUAL")
	}
	return result
}

/* create n shares of a multiplication triple */
func triple32(n int) []struct{ a, b, c uint32 } {
	a, b := rand32(), rand32()
	c := a & b
	a_shares := split_uint32(a, n)
	b_shares := split_uint32(b, n)
	c_shares := split_uint32(c, n)
	result := make([]struct{ a, b, c uint32 }, n)
	for i, _ := range result {
		result[i] = struct{ a, b, c uint32 }{a_shares[i], b_shares[i], c_shares[i]}
	}
	return result
}

func Example(n int) []X {
	/* triples */
	triples32 := make([][]struct{ a, b, c uint32 }, n)
	num_triples := 200
	for i, _ := range triples32 {
		triples32[i] = make([]struct{ a, b, c uint32 }, num_triples)
	}
	for j := 0; j < num_triples; j++ {
		shares_of_a_triple := triple32(n)
		for i, t := range shares_of_a_triple {
			triples32[i][j] = t
		}
	}

	/* channels */
	rchannels := make([] /*n*/ [] /*n*/ chan uint32, n)
	wchannels := make([] /*n*/ [] /*n*/ chan uint32, n)
	for i := range rchannels {
		rchannels[i] = make([] /*n*/ chan uint32, n)
		wchannels[i] = make([] /*n*/ chan uint32, n)
	}
	for i := range rchannels {
		for j := range rchannels {
			if i == j {
				continue
			}
			ch := make(chan uint32)
			rchannels[i][j] = ch
			wchannels[j][i] = ch
		}
	}

	/* final result */
	result := make([]X, n)
	for id := range result {
		inputs := make([]uint32, 1)
		inputs[0] = uint32(id)
		result[id] =
			X{id,
				triples32[id],
				make([]struct{ a, b, c uint8 }, 0),
			        make([]struct{ a, b, c bool }, 0),
				rchannels[id],
				wchannels[id],
				inputs}
	}
	return result
}

func RunExample() uint32 {
	xs := Example(7)
	done := make(chan uint32, len(xs))
	results := make([]uint32, len(xs))
	for _, x := range xs {
		go func(x X) {
			y := Input32(x, 3)
			z := Input32(x, 6)
			Output32(x, Sub32(x, y, z))
			done <- Output32(x, Add32(x, y, z))
		}(x)
	}
	for i := range xs {
		results[i] = <-done
	}
	result := results[0]
	for i := range results {
		if results[i] != result {
			panic(fmt.Sprintf("Error: results don't match (0x%08x != 0x%08x)", results[i], result))
		}
	}
	fmt.Printf("Result: 0x%08x\n", result)
	return result
}

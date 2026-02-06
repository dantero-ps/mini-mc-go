[Home](/)

[Go (Fundamentals) 101](/article/101.html)

[Go Generics 101](/generics/101.html)

[Go Details & Tips 101](/details-and-tips/101.html)

[Go Optimizations 101](/optimizations/101.html)

[Go Quizzes 101](/quizzes/101.html)

[Go Q&A 101](/q-and-a/101.html)

Go Practices 101

[Go Bugs 101](/bugs/101.html)

[Go 101 Blog](/blog/101.html)

[Go 101 Apps & Libs](/apps-and-libs/101.html)

Theme: dark/light

The [Go Optimizations 101](/optimizations/101.html), [Go Details & Tips 101](/details-and-tips/101.html) and [Go Generics 101](/generics/101.html) books are all updated to Go 1.25. The most cost-effective way to get them is through [this book bundle](https://leanpub.com/b/go-optimizations-details-generics) in the Leanpub book store.

If you would like to learn some Go details and facts every serveral days, please follow [@zigo\_101](https://x.com/zigo_101).

# BCE (Bound Check Elimination)

Go is a memory safe language. In array/slice/string element indexing and subslice operations, Go runtime will check whether or not the involved indexes are out of range. If an index is out of range, a panic will be produced to prevent the invalid index from doing harm. This is called bounds checking.

Bounds checks make our code run safely, on the other hand, they also make our code run a little slower. This is a trade-off a safe language must made.

Since Go toolchain v1.7, the standard Go compiler has started to support BCE (bounds check elimination). BCE can avoid some unnecessary bounds checks, so that the standard Go compiler could generate more efficient programs.

The following will list some examples to show in which cases BCE works and in which cases BCE doesn't work.

We could use the `-d=ssa/check_bce` compiler option to show which code lines need bound checks.

## Example 1

A simple example:

```go
// example1.go
package main

func f1a(s []struct{}, index int) {
	_ = s[index] // line 5: Found IsInBounds
	_ = s[index]
	_ = s[index:]
	_ = s[:index+1]
}

func f1b(s []byte, index int) {
	s[index-1] = 'a' // line 12: Found IsInBounds
	_ = s[:index]
}

func f1c(a [5]int) {
	_ = a[0]
	_ = a[4]
}

func f1d(s []int) {
	if len(s) > 2 {
	    _, _, _ = s[0], s[1], s[2]
	}
}

func f1g(s []int) {
	middle := len(s) / 2
	_ = s[:middle]
	_ = s[middle:]
}

func main() {}
```

Let's run it with the `-d=ssa/check_bce` compiler option:

```
$ go run -gcflags="-d=ssa/check_bce" example1.go
./example1.go:5:7: Found IsInBounds
./example1.go:12:3: Found IsInBounds
```

The outputs show that only two code lines needs bound checks in the above example code.

Note that: Go toolchains with version smaller than 1.21 failed to remove the bound checks in the `f1g` function.

And note that, up to now (Go toolchain v1.24.n), the official standard compiler doesn't check BCE for an operation in a generic function if the operation involves type parameters and the generic function is never instantiated. For example, the command `go run -gcflags=-d=ssa/check_bce bar.go` will report nothing.

```go
// bar.go
package bar

func foo[E any](s []E) {
	_ = s[0] // line 5
	_ = s[1] // line 6
	_ = s[2] // line 7
}

// var _ = foo[bool]
```

However, if the variable declaration line is enabled, then the compiler will report:

```
./bar.go:5:7: Found IsInBounds
./bar.go:6:7: Found IsInBounds
./bar.go:7:7: Found IsInBounds
./bar.go:4:6: Found IsInBounds
```

## Example 2

All the bound checks in the slice element indexing and subslice operations shown in the following example are eliminated.

```go
// example2.go
package main

func f2a(s []int) {
	for i := range s {
		_ = s[i]
		_ = s[i:len(s)]
		_ = s[:i+1]
	}
}

func f2b(s []int) {
	for i := 0; i < len(s); i++ {
		_ = s[i]
		_ = s[i:len(s)]
		_ = s[:i+1]
	}
}

func f2c(s []int) {
	for i := len(s) - 1; i >= 0; i-- {
		_ = s[i]
		_ = s[i:len(s)]
		_ = s[:i+1]
	}
}

func f2d(s []int) {
	for i := len(s); i > 0; {
		i--
		_ = s[i]
		_ = s[i:len(s)]
		_ = s[:i+1]
	}
}

func f2e(s []int) {
	for i := 0; i < len(s) - 1; i += 2 {
		_ = s[i]
		_ = s[i:len(s)]
		_ = s[:i+1]
	}
}

func main() {}
```

Run it, we will find that nothing is outputted. Yes, the official standard Go compiler is so clever that it finds all bound checks may be removed in the above example code.

```
$ go run -gcflags="-d=ssa/check_bce" example2.go
```

Note: prior to v1.24, the standard Go compiler failed to remove the bound checks in the following two loops.

```go
func f2g(s []int) {
	for i := len(s) - 1; i >= 0; i-- {
		_ = s[:i+1]
	}
}

func f2h(s []int) {
	for i := 0; i <= len(s) - 1; i++ {
		_ = s[:i+1]
	}
}
```

## Example 3

We should try to evaluate the element indexing or subslice operation with the largest index as earlier as possible to reduce the number of bound checks.

In the following example, if the expression `s[3]` is evaluated without panicking, then the bound checks for `s[0]`, `s[1]` and `s[2]` could be eliminated.

```go
// example3.go
package main

func f3a(s []int32) int32 {
	return s[0] | // Found IsInBounds (line 5)
		s[1] | // Found IsInBounds
		s[2] | // Found IsInBounds
		s[3]   // Found IsInBounds
}

func f3b(s []int32) int32 {
	return s[3] | // Found IsInBounds (line 12)
		s[0] |
		s[1] |
		s[2]
}

func main() {
}
```

Run it, we get:

```
./example3.go:5:10: Found IsInBounds
./example3.go:6:4: Found IsInBounds
./example3.go:7:4: Found IsInBounds
./example3.go:8:4: Found IsInBounds
./example3.go:12:10: Found IsInBounds
```

From the output, we could learn that there are 4 bound checks in the `f3a` function, but only one in the `f3b` function.

## Example 4

Since Go toolchain v1.19, the bould check in the `f5a` function is successfully removed,

```go
func f5a(isa []int, isb []int) {
	if len(isa) > 0xFFF {
		for _, n := range isb {
			_ = isa[n & 0xFFF]
		}
	}
}
```

However, before Go toolchain v1.19, the check is not removed. The compilers before version 1.19 need a hint to be removed, as shown in the `f5b` function:

```go
func f5b(isa []int, isb []int) {
	if len(isa) > 0xFFF {
		// A successful hint (for v1.18- compilers)
		isa = isa[:0xFFF+1]
		for _, n := range isb {
			_ = isa[n & 0xFFF] // BCEed!
		}
	}
}

func f5c(isa []int, isb []int) {
	if len(isa) > 0xFFF {
		// A not-workable hint (for v1.18- compilers)
		_ = isa[:0xFFF+1]
		for _, n := range isb {
			_ = isa[n & 0xFFF] // Found IsInBounds
		}
	}
}

func f5d(isa []int, isb []int) {
	if len(isa) > 0xFFF {
		// A not-workable hint (for v1.18- compilers)
		_ = isa[0xFFF]
		for _, n := range isb {
			_ = isa[n & 0xFFF] // Found IsInBounds
		}
	}
}
```

The next section shows more cases which need compiler hints to avoid some unnecessary bound checks.

## Example 5

Prior to Go toolchain v1.24, there are some unnecessary bound checks in the following code:

```go
func fz(s, x, y []byte) {
	n := copy(s, x)
	copy(s[n:], y) // Found IsSliceInBounds (1.24-)
	_ = x[n:]      // Found IsSliceInBounds (1.24-)
}

func fy(a, b []byte) {
    for i := range min(len(a), len(b)) {
        _ = a[i] // Found IsInBounds (1.24-)
        _ = b[i] // Found IsInBounds (1.24-)
    }
}

func fx(a [256]byte) {
	for i := 0; i < 128; i++ {
		_ = a[2*i] // Found IsInBounds (1.24-)
	}
}

func f4a(is []int, bs []byte) {
	if len(is) >= 256 {
		for _, n := range bs {
			_ = is[n] // Found IsInBounds (1.24-)
		}
	}
}
```

Since version 1.24, all of them are removed.

Before version 1.24, we have to add a hint line to remove the bound check in the `f4a` function:

```go
func f4a(is []int, bs []byte) {
	if len(is) >= 256 {
		is = is[:256] // a successful hint
		for _, n := range bs {
			_ = is[n] // BCEed!
		}
	}
}
```

Since version 1.24, the hint line becomes unnecessary.

## Sometimes, the compiler needs some hints to remove some bound checks

The official standard Go compiler is still not smart enough to remove all unnecessary bound checks. Sometimes, the compiler needs to be given some hints to remove some bound checks.

In the following example, by adding a redundant `if` code block in the function `NumSameBytes_2`, all bound checks in the loop are eliminated.

```go
type T = string

func NumSameBytes_1(x, y T) int {
	if len(x) > len(y) {
		x, y = y, x
	}
	for i := 0; i < len(x); i++ {
		if x[i] != 
			y[i] { // Found IsInBounds
			return i
		}
	}
	return len(x)
}

func NumSameBytes_2(x, y T) int {
	if len(x) > len(y) {
		x, y = y, x
	}
	
	// a successful hint
	if len(x) > len(y) {
		panic("unreachable")
	}
	
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] { // BCEed!
			return i
		}
	}
	return len(x)
}
```

The above hint works when `T` is either a string type or a slice type, whereas each of the following two hints only works for one case (as of Go toolchain v1.24.n).

```go
func NumSameBytes_3(x, y T) int {
	if len(x) > len(y) {
		x, y = y, x
	}
	
	y = y[:len(x)] // a hint, only works if T is slice
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			return i
		}
	}
	return len(x)
}

func NumSameBytes_4(x, y T) int {
	if len(x) > len(y) {
		x, y = y, x
	}
	
	_ = y[:len(x)] // a hint, only works if T is string
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			return i
		}
	}
	return len(x)
}
```

Please note that, the future versions of the standard official Go compiler will become smarter so that the above hints will become unnecessary later.

## Write code in BCE-friendly ways

In the following example, the `f7b` and `f7c` functions makes 3 less bound checks than `f7a`.

```go
func f7a(s []byte, i int) {
	_ = s[i+3] // Found IsInBounds
	_ = s[i+2] // Found IsInBounds
	_ = s[i+1] // Found IsInBounds
	_ = s[i]   // Found IsInBounds
}

func f7b(s []byte, i int) {
	s = s[i:i+4] // Found IsSliceInBounds
	_ = s[3]
	_ = s[2]
	_ = s[1]
	_ = s[0]
}

func f7c(s []byte, i int) {
	s = s[i:i+4:i+4] // Found IsSliceInBounds
	_ = s[3]
	_ = s[2]
	_ = s[1]
	_ = s[0]
}
```

However, please note that, there might be [some other factors](3-array-and-slice.html#specify-capacity) which will affect program performances. On my machine (Intel i5-4210U CPU @ 1.70GHz, Linux/amd64), among the above 3 functions, the function `f7b` is actually the least performant one.

```go
Benchmark_f7a-4  3861 ns/op
Benchmark_f7b-4  4223 ns/op
Benchmark_f7c-4  3477 ns/op
```

In practice, it is encouraged to use the three-index subslice form (`f7c`).

In the following example, benchmark results show that

*   the `f8z` function is the most performant one (in line with expectation)
*   but the `f8y` function is as performant as the `f8x` function (unexpected).

```go
func f8x(s []byte) {
	var n = len(s)
	s = s[:n]
	for i := 0; i <= n - 4; i += 4 {
		_ = s[i+3] // Found IsInBounds
		_ = s[i+2] // Found IsInBounds
		_ = s[i+1] // Found IsInBounds
		_ = s[i]
	}
}

func f8y(s []byte) {
	for i := 0; i <= len(s) - 4; i += 4 {
		s2 := s[i:]
		_ = s2[3] // Found IsInBounds
		_ = s2[2]
		_ = s2[1]
		_ = s2[0]
	}
}

func f8z(s []byte) {
	for i := 0; len(s) >= 4; i += 4 {
		_ = s[3]
		_ = s[2]
		_ = s[1]
		_ = s[0]
		s = s[4:]
	}
}
```

In fact, benchmark results also show the following `f8y3` function is as performant as the `f8z` function and the performance of the `f8y2` function is on par with the `f8y` function. So it is encouraged to use three-index subslice forms for such situations in practice.

```go
func f8y2(s []byte) {
	for i := 0; i < len(s) - 3; i += 4 {
		s2 := s[i:i+4] // Found IsInBounds
		_ = s2[3]
		_ = s2[2]
		_ = s2[1]
		_ = s2[0]
	}
}

func f8y3(s []byte) {
	for i := 0; i < len(s) - 3; i += 4 {
		s2 := s[i:i+4:i+4] // Found IsInBounds
		_ = s2[3]
		_ = s2[2]
		_ = s2[1]
		_ = s2[0]
	}
}
```

In the following example, there are no bound checks in the `f9b` and `f9c` functions, but there is one in the `f9a` function.

```go
func f9a(n int) []int {
	buf := make([]int, n+1)
	k := 0
	for i := 0; i <= n; i++ {
		buf[i] = k // Found IsInBounds
		k++
	}
	return buf
}


func f9b(n int) []int {
	buf := make([]int, n+1)
	k := 0
	for i := 0; i < len(buf); i++ {
		buf[i] = k
		k++
	}
	return buf
}

func f9c(n int) []int {
	buf := make([]int, n+1)
	k := 0
	for i := 0; i < n+1; i++ {
		buf[i] = k
		k++
	}
	return buf
}
```

In the following code, the function `f6b` is more performant than `f6a`, but both of them are much less performant than `f6c`.

```go
const N = 3

func f6a(s []byte) {
	for i := 0; i < len(s)-(N-1); i += N {
		_ = s[i+N-1] // Found IsInBounds
	}
}

func f6b(s []byte) {
	for i := N-1; i < len(s); i += N {
		_ = s[i] // Found IsInBounds
	}
}

func f6c(s []byte) {
	for i := uint(N-1); i < uint(len(s)); i += N {
		_ = s[i]
	}
}
```

Global (package-level) slices are often unfriendly to BCE, so we should try to assign them to local ones to eliminate some unnecessary bound checks. For example, in the following code, the `fa0` function does one more bound check than the `fa1` and `fa2` functions, so the function calls `fa1()` and `fa2(s)` are both more performant than `fa0()`.

```go
var s = make([]int, 5)

func fa0() {
	for i := range s {
		s[i] = i // Found IsInBounds
	}
}

func fa1() {
	s := s
	for i := range s {
		s[i] = i
	}
}

func fa2(x []int) {
	for i := range x {
		x[i] = i
	}
}
```

Arrays are often more BCE-friendly than slices. In the following code, the array version functions (`fb2` and `fc2`) don't need bound checks.

```go
var s = make([]int, 256)
var a = [256]int{}

func fb1() int {
    return s[100] // Found IsInBounds
}

func fb2() int {
    return a[100]
}

func fc1(n byte) int {
    return s[n] // Found IsInBounds
}

func fc2(n byte) int {
    return a[n]
}
```

Prior Go toolchain v1.24, the function `f0b` in the following code is much more performant than `f0a`, because there are some unnecessary bound checks in the `f0a` function. Since Go toolchain v1.24, the unnecessary bound checks in the `f0a` function are all removed, so the performance of the `f0a` function is improved much (though it is still some slower than `f0b`).

```go
func f0a(x [16]byte) (r [4]byte){
	for i := 0; i < 4; i++ {
		r[i] =
			x[i*4+3] ^
			x[i*4+2] ^
			x[i*4+1] ^
			x[i*4]     
	}
	return
}

func f0b(x [16]byte) (r [4]byte){
	r[0] = x[3] ^ x[2] ^ x[1] ^ x[0]
	r[1] = x[7] ^ x[6] ^ x[5] ^ x[4]
	r[2] = x[11] ^ x[10] ^ x[9] ^ x[8]
	r[3] = x[15] ^ x[14] ^ x[13] ^ x[12]
	return
}
```

Please note that, the future versions of the standard official Go compiler will become smarter so that more BCE-unfriendly code might become BCE-friendly later.

## The current official standard Go compiler fails to eliminate some unnecessary bound checks

As of Go toolchain v1.24.n, the official standard Go compiler doesn't eliminate the following unnecessary bound checks.

```go
func fd(data []int, check func(int) bool) []int {
	var k = 0
	for _, v := range data {
		if check(v) {
			data[k] = v // Found IsInBounds
			k++
		}
	}
	return data[:k] // Found IsSliceInBounds
}


// For the only bound check in the following function,
// * if N == 1, it will be always removed.
// * if N is a power of 2, Go toolchain 1.19+ can remove it.
// * for other cases, Go toolchain fails to remove it.
func fe(s []byte) {
	const N = 3
	if len(s) >= N {
		r := len(s) % N
		_ = s[r] // Found IsInBounds
	}
}

func ff(s []byte) {
	for i := 0; i < len(s); i++ {
		_ = s[i/2] // Found IsInBounds
		_ = s[i/3] // Found IsInBounds
	}
}

func fg(src, dst []byte) {
	dst = dst[:len(src)]
	for len(src) >= 4 {
		dst[1] = // Found IsInBounds
			src[0]
		dst[0] = src[1]
		src = src[4:]
		dst = dst[4:] // Found IsSliceInBounds
	}
}
```

The future versions of the standard official Go compiler will become smarter so that the above unnecessary bound checks will be eliminated later.

- - -

[(more articles ↡)](#i-5-bce.html)

- - -

The **_Go 101_** project is hosted on [GitHub](https://github.com/go101/go101). Welcome to improve **_Go 101_** articles by submitting corrections for all kinds of mistakes, such as typos, grammar errors, wording inaccuracies, description flaws, code bugs and broken links.

The digital versions of this book are available at the following places:

*   [Leanpub store](https://leanpub.com/go-optimizations-101), _$7.99+_ (Or buy this book from [this](https://leanpub.com/b/go-optimizations-details-generics) or [this](https://leanpub.com/b/go-optimizations-details) book bundle).
*   [Apple Books store](https://books.apple.com/book/id1609924340), _$7.99_.
*   [Amazon Kindle store](https://www.amazon.com/dp/B09NT2HJCM), _$7.99_.

Tapir, the author of Go 101, has been on writing the Go 101 series books since 2016 July. New contents will be continually added to the books (and the go101.org website) from time to time. Tapir is also an indie game developer. You can also support Go 101 by playing [Tapir's games](https://www.tapirgames.com):

*   [Color Infection](https://www.tapirgames.com/App/Color-Infection) (★★★★★), a physics based original casual puzzle game. 140+ levels.
*   [Rectangle Pushers](https://www.tapirgames.com/App/Rectangle-Pushers) (★★★★★), an original casual puzzle game. Two modes, 104+ levels.
*   [Let's Play With Particles](https://www.tapirgames.com/App/Let-Us-Play-With-Particles), a casual action original game. Three mini games are included.

Individual donations [via PayPal](https://paypal.me/tapirliu) are also welcome.

- - -

Articles in this book:

*   [Acknowledgments](0.0-acknowledgements.html)
*   [About Go Optimizations 101](0.1-introduction.html)
*   Value Parts and Value Sizes (available in [the paid ebooks](#ebooks))

*   value/type sizes
*   memory alignments
*   struct padding
*   avoid larger copy costs

*   [Memory Allocations](0.3-memory-allocations.html)
*   Stack and Escape Analysis (available in [the paid ebooks](#ebooks))

*   escape analysis
*   how to control value allocation places
*   stacks growth and shrinkage
*   how to reduce stack grow times

*   Garbage Collection (available in [the paid ebooks](#ebooks))

*   GC pacer
*   how to reduce GC pressure
*   control GC frequency

*   [Pointers](1-pointer.html)
*   Structs (available in [the paid ebooks](#ebooks))

*   3 facts/suggestions

*   Arrays and Slices (available in [the paid ebooks](#ebooks))

*   10+ facts/suggestions

*   String and Byte Slices (available in [the paid ebooks](#ebooks))

*   10+ facts/suggestions

*   **BCE (Bound Check Elimination)**

*   the cases BCE works for
*   the cases BCE doesn't work for
*   the cases BCE works for when given hints

*   [Maps](6-map.html)
*   Channels (available in [the paid ebooks](#ebooks))

*   3 facts/suggestions

*   Functions (available in [the paid ebooks](#ebooks))

*   how to make a function inline-able
*   how to make a function not inline-able
*   pointer parameters/results vs. non-pointer ones
*   named results vs. anonymous ones
*   10+ facts/suggestions

*   Interfaces (available in [the paid ebooks](#ebooks))

*   value boxing costs
*   3+ facts/suggestions
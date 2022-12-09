# go-conversions

A utility for displaying which of Go's primitive types can be natively converted between each other as reported by the Go compiler itself.

A small section of output would look like this:

```text
INFO[0000] ---------- converting int64 values ---------- 
INFO[0000]      int64 -> bool       ❌                   
INFO[0000]      int64 -> uint8      ✅                   
INFO[0000]      int64 -> uint16     ✅                   
INFO[0000]      int64 -> uint32     ✅                   
INFO[0000]      int64 -> uint64     ✅                   
INFO[0000]      int64 -> int8       ✅                   
INFO[0000]      int64 -> int16      ✅                   
INFO[0000]      int64 -> int32      ✅                   
INFO[0000]      int64 -> int64      ✅                   
INFO[0000]      int64 -> float32    ✅                   
INFO[0000]      int64 -> float64    ✅                   
INFO[0000]      int64 -> complex64  ❌                   
INFO[0000]      int64 -> complex128 ❌                   
INFO[0000]      int64 -> string     ✅                   
INFO[0000]      int64 -> int        ✅                   
INFO[0000]      int64 -> uint       ✅                   
INFO[0000]      int64 -> uintptr    ✅                   
INFO[0000]      int64 -> byte       ✅                   
INFO[0000]      int64 -> rune       ✅                   
```

This lists all the possible primitive types you can or can not convert an `int64` to.

A section like this will be listed for each of the 19 primitive types as defined by Go's [`builtin`](https://pkg.go.dev/builtin) package.

### FAQ

> Why did you make this?

I was writing out some utility wrapper functions for handling instantiation of Google's [nullable types](https://github.com/golang/protobuf/blob/master/ptypes/wrappers/wrappers.pb.go#L15-L23) for protobuf without having to futz with the actual values, i.e. a function which accepts a primitive type and returns a nullable type wrapper for it. An example could be:

```go
package main

import "github.com/golang/protobuf/ptypes/wrappers"

// Int32AsInt64 converts an int32, v, into a *wrappers.Int64Value
func Int32AsInt64(v int32) *wrappers.Int64Value {
	var w wrappers.Int64Value

	w.Value = int64(v)

	return &w
}
```

One thing to note about this function is that it has to do an explicit type conversion on `v`, an `int32` to an `int64` for it to be valid to use within a `wrappers.Int64Value`. This sent me down a bit of a rabbit hole, because I realized you can convert any `int`-class type to an `int64`, not just `int32`, in fact you can convert an `int64` to any other primitive _except_ a `bool`, `complex64`, or `complex128`, as the chart above shows. And this isn't unique to only `int64`, a lot of primitives can be converted to a lot more primitives than I originally realized.

Now why is this relevant? Well, because I wanted to create a utility wrapper function for _every possible primitive_ to its corresponding nullable type, as determined by the underlying primitive used in that nullable type. Let's look at the `wrappers.Int64Value` to determine that underlying primitive type. It is defined in the source code by Google as:

```go
package wrapperspb

import protoimpl "google.golang.org/protobuf/runtime/protoimpl"

// Wrapper message for `int64`.
//
// The JSON representation for `Int64Value` is JSON string.
type Int64Value struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The int64 value.
	Value int64 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}
```

And if we look at the `Value` field, it's primitive is `int64`. This means to be a _true_ completionist, I would need to define a utility wrapper function that maps all primitives which can validly be converted to `int64`, which would mean defining the following set of functions, according to the chart above for `int64` conversions:

```go
func Uint8AsInt64(uint8) *wrappers.Int64Value
func Uint16AsInt64(uint16) *wrappers.Int64Value
func Uint32AsInt64(uint32) *wrappers.Int64Value
func Uint64AsInt64(uint64) *wrappers.Int64Value
func Int8AsInt64(int8) *wrappers.Int64Value
func Int16AsInt64(int16) *wrappers.Int64Value
func Int32AsInt64(int32) *wrappers.Int64Value
func Int64AsInt64(int64) *wrappers.Int64Value
func Float32AsInt64(float32) *wrappers.Int64Value
func Float64AsInt64(float64) *wrappers.Int64Value
func StringAsInt64(string) *wrappers.Int64Value
func IntAsInt64(int) *wrappers.Int64Value
func UintAsInt64(uint) *wrappers.Int64Value
func UintptrAsInt64(uintptr) *wrappers.Int64Value
func ByteAsInt64(byte) *wrappers.Int64Value
func RuneAsInt64(rune) *wrappers.Int64Value
```

But remember, this is only for `int64`. There are 18 other types, whose ability to be converted to any of the other types remains in question.

So long story short, I tried to find an explicit list of all primitives that can and can't be converted between each other in Go but could not find anything satisfactory. So that is why I made this.

> How did you do it?

In addition to trying to explicitly define that conversion information, I also took this as an opportunity to learn about templating in Go, because the way to determine if two primitives can be converted between each other is to do something like:

```go
var v int
_ = string(v)
```

And considering there are currently 19 primitives, and I need to check each one against each other one, that is 19 * 19 = 361 checks. So instead of writing them all out and checking which are valid and which fail, it seemed more prudent to simply list out the 19 primitive types and have Go's templating engine create these checks for me.

But this is only part of the picture, because the checks written out are just code. You can look at the code in an IDE and see which ones are marked as errors, but all of this info is still just in your head. It would be nice if that data could be written out explicitly and in an organized fashion so you could look up a type you had in mind quickly. This is where the next step comes in: Compiling the generated code with the intention of compilation failing and reporting all the conversion errors to stderr.

This phase was completed by invoking `go build -gcflags=-e -o /dev/null <generated-code>` via an `exec.Cmd`, so, having a Go binary attempt to create a new Go binary, which I thought was pretty cool! The `-gcflags=-e` section was included to remove the compilation error quantity limitation that `go build` imposes by default to only show something like the first 10 errors before cutting the output off. Obviously we need more than that if we want to have an exhaustive list. The `-o /dev/null` is just to reroute the final binary to `/dev/null` to be deleted since we don't actually want a real binary, not to mention that compilation should always fail anyway meaning there won't even be a binary generated, but it was included as an added feature for good measure.

So then, once we have those compilation errors, we do some string and regex processing to extract the types that could not be converted between and finally we present that data to `stdout`.

> Why go about it this way?

For fun and learning mostly. It was a good opportunity to learn about type conversions, templating, and executing commands in Go. There very well may have been an easier way to do this, but I like this approach simply because of the novelty of it.

> Can I see the output without cloning and running this program?

Yes, I have attached a copy of the output to the end of this README file.

> How can I run this?

Clone the repo, run `go mod vendor`, then execute the program from the project root.

Alternatively, if you use IntelliJ, there is a run-configuration checked into this repository called `go-conversions:run` which you can execute to run the application.

> What about `[]byte -> string` and vice versa? Those are also valid conversions, you know.

Yes, they are, but they're not from one _primitive_ to another. `[]byte` is a slice of a primitive, `byte`. For the sake of simplicity, I only went with conversions between singular primitives.

> What about conversions involving `interface{}` and `struct{}`?

Those were left off for simplicity and pragmatism. I don't think it's fair to consider a `struct{}` or `interface{}` as a "primitive" in Go, plus the addition of `{}` to those "type" names makes templating much harder. Additionally, any type can be natively converted to `interface{}` and `interface{}` cannot be natively converted to any other type, so it doesn't add much value.

> What about pointers?

Same reason: that's not exactly a primitive. Its a pointer _to_ an underlying primitive. So the same logic applies, that if, say, an `int` can be converted to a `int64`, then a `*int` can be converted to a `*int64` WITH the proper dereferencing and whatnot.

> What about `error`?

Still not a primitive. `error` is an `interface`.

> Okay smart guy, since you are such a purist about primitives, why do you allow `byte` which is a type alias for `uint8` and `rune` which is a type alias for `int32`? Wouldn't that make these not primitives as well?

Man idk, I guess? Since it's an alias its just some extra information since it won't change anything. Effectively, anything a `unit8` can convert to a `byte` would also be convertible to, similarly for `int32` and `rune`. I consider it wise to keep these in as their own primitives despite this, because `byte` and `rune` are heavily used types, more so than the types they are aliased over, despite their status as an alias for some other primitive.

> Why show conversions both ways, i.e. `A -> B` and _also_ `B -> A`? If `A` can or can't be converted to `B`, isn't it the case that converting `B -> A` will have the same result?

No. This is generally true, but there are cases where one type can be converted to another, but not the other way around. For example, the conversion of `rune -> string` is valid, but the conversion of `string -> rune` is _not_.

> Why do you show conversions from one type back to itself? This is always valid.

Laziness/completeness. I could have made the code a bit more sophisticated to skip checking conversions between a type and itself, but it isn't really necessary, and I suppose it keeps the ordering and indexes of the output the same between all sections. On second thought, let's just go with completeness.

> You are using `logrus` for logging the output, which prepends the output with `INFO[0000]`, which is annoying to me. Can this be removed?

No, deal with it.

> This information is already well known and available at ...

Cool, but I don't really care. For one, I tried to find an exhaustive list like this online and came up blank, perhaps the language spec outlined the rules but I didn't see a list anywhere. But on top of that, this project was more than just the end result of valid and invalid conversions, it was also an exploration into Go's templating engine, executing commands via Go, and programmatically interpreting Go compiler errors.

> Why are you being so absurdly thorough with this README for something so... trivial I guess?

Bro, just leave me alone, this is cathartic for me alright, lol.

### Results

Below are the full results from running this program as of **12/8/2022**. They are being recorded to save anyone from having to run this program on their own machine if all they care about is seeing what primitives can be converted in Go.

```text
INFO[0000] ---------- converting bool values ---------- 
INFO[0000]       bool -> bool       ✅                   
INFO[0000]       bool -> uint8      ❌                   
INFO[0000]       bool -> uint16     ❌                   
INFO[0000]       bool -> uint32     ❌                   
INFO[0000]       bool -> uint64     ❌                   
INFO[0000]       bool -> int8       ❌                   
INFO[0000]       bool -> int16      ❌                   
INFO[0000]       bool -> int32      ❌                   
INFO[0000]       bool -> int64      ❌                   
INFO[0000]       bool -> float32    ❌                   
INFO[0000]       bool -> float64    ❌                   
INFO[0000]       bool -> complex64  ❌                   
INFO[0000]       bool -> complex128 ❌                   
INFO[0000]       bool -> string     ❌                   
INFO[0000]       bool -> int        ❌                   
INFO[0000]       bool -> uint       ❌                   
INFO[0000]       bool -> uintptr    ❌                   
INFO[0000]       bool -> byte       ❌                   
INFO[0000]       bool -> rune       ❌                   
INFO[0000] ---------- converting uint8 values ---------- 
INFO[0000]      uint8 -> bool       ❌                   
INFO[0000]      uint8 -> uint8      ✅                   
INFO[0000]      uint8 -> uint16     ✅                   
INFO[0000]      uint8 -> uint32     ✅                   
INFO[0000]      uint8 -> uint64     ✅                   
INFO[0000]      uint8 -> int8       ✅                   
INFO[0000]      uint8 -> int16      ✅                   
INFO[0000]      uint8 -> int32      ✅                   
INFO[0000]      uint8 -> int64      ✅                   
INFO[0000]      uint8 -> float32    ✅                   
INFO[0000]      uint8 -> float64    ✅                   
INFO[0000]      uint8 -> complex64  ❌                   
INFO[0000]      uint8 -> complex128 ❌                   
INFO[0000]      uint8 -> string     ✅                   
INFO[0000]      uint8 -> int        ✅                   
INFO[0000]      uint8 -> uint       ✅                   
INFO[0000]      uint8 -> uintptr    ✅                   
INFO[0000]      uint8 -> byte       ✅                   
INFO[0000]      uint8 -> rune       ✅                   
INFO[0000] ---------- converting uint16 values ---------- 
INFO[0000]     uint16 -> bool       ❌                   
INFO[0000]     uint16 -> uint8      ✅                   
INFO[0000]     uint16 -> uint16     ✅                   
INFO[0000]     uint16 -> uint32     ✅                   
INFO[0000]     uint16 -> uint64     ✅                   
INFO[0000]     uint16 -> int8       ✅                   
INFO[0000]     uint16 -> int16      ✅                   
INFO[0000]     uint16 -> int32      ✅                   
INFO[0000]     uint16 -> int64      ✅                   
INFO[0000]     uint16 -> float32    ✅                   
INFO[0000]     uint16 -> float64    ✅                   
INFO[0000]     uint16 -> complex64  ❌                   
INFO[0000]     uint16 -> complex128 ❌                   
INFO[0000]     uint16 -> string     ✅                   
INFO[0000]     uint16 -> int        ✅                   
INFO[0000]     uint16 -> uint       ✅                   
INFO[0000]     uint16 -> uintptr    ✅                   
INFO[0000]     uint16 -> byte       ✅                   
INFO[0000]     uint16 -> rune       ✅                   
INFO[0000] ---------- converting uint32 values ---------- 
INFO[0000]     uint32 -> bool       ❌                   
INFO[0000]     uint32 -> uint8      ✅                   
INFO[0000]     uint32 -> uint16     ✅                   
INFO[0000]     uint32 -> uint32     ✅                   
INFO[0000]     uint32 -> uint64     ✅                   
INFO[0000]     uint32 -> int8       ✅                   
INFO[0000]     uint32 -> int16      ✅                   
INFO[0000]     uint32 -> int32      ✅                   
INFO[0000]     uint32 -> int64      ✅                   
INFO[0000]     uint32 -> float32    ✅                   
INFO[0000]     uint32 -> float64    ✅                   
INFO[0000]     uint32 -> complex64  ❌                   
INFO[0000]     uint32 -> complex128 ❌                   
INFO[0000]     uint32 -> string     ✅                   
INFO[0000]     uint32 -> int        ✅                   
INFO[0000]     uint32 -> uint       ✅                   
INFO[0000]     uint32 -> uintptr    ✅                   
INFO[0000]     uint32 -> byte       ✅                   
INFO[0000]     uint32 -> rune       ✅                   
INFO[0000] ---------- converting uint64 values ---------- 
INFO[0000]     uint64 -> bool       ❌                   
INFO[0000]     uint64 -> uint8      ✅                   
INFO[0000]     uint64 -> uint16     ✅                   
INFO[0000]     uint64 -> uint32     ✅                   
INFO[0000]     uint64 -> uint64     ✅                   
INFO[0000]     uint64 -> int8       ✅                   
INFO[0000]     uint64 -> int16      ✅                   
INFO[0000]     uint64 -> int32      ✅                   
INFO[0000]     uint64 -> int64      ✅                   
INFO[0000]     uint64 -> float32    ✅                   
INFO[0000]     uint64 -> float64    ✅                   
INFO[0000]     uint64 -> complex64  ❌                   
INFO[0000]     uint64 -> complex128 ❌                   
INFO[0000]     uint64 -> string     ✅                   
INFO[0000]     uint64 -> int        ✅                   
INFO[0000]     uint64 -> uint       ✅                   
INFO[0000]     uint64 -> uintptr    ✅                   
INFO[0000]     uint64 -> byte       ✅                   
INFO[0000]     uint64 -> rune       ✅                   
INFO[0000] ---------- converting int8 values ---------- 
INFO[0000]       int8 -> bool       ❌                   
INFO[0000]       int8 -> uint8      ✅                   
INFO[0000]       int8 -> uint16     ✅                   
INFO[0000]       int8 -> uint32     ✅                   
INFO[0000]       int8 -> uint64     ✅                   
INFO[0000]       int8 -> int8       ✅                   
INFO[0000]       int8 -> int16      ✅                   
INFO[0000]       int8 -> int32      ✅                   
INFO[0000]       int8 -> int64      ✅                   
INFO[0000]       int8 -> float32    ✅                   
INFO[0000]       int8 -> float64    ✅                   
INFO[0000]       int8 -> complex64  ❌                   
INFO[0000]       int8 -> complex128 ❌                   
INFO[0000]       int8 -> string     ✅                   
INFO[0000]       int8 -> int        ✅                   
INFO[0000]       int8 -> uint       ✅                   
INFO[0000]       int8 -> uintptr    ✅                   
INFO[0000]       int8 -> byte       ✅                   
INFO[0000]       int8 -> rune       ✅                   
INFO[0000] ---------- converting int16 values ---------- 
INFO[0000]      int16 -> bool       ❌                   
INFO[0000]      int16 -> uint8      ✅                   
INFO[0000]      int16 -> uint16     ✅                   
INFO[0000]      int16 -> uint32     ✅                   
INFO[0000]      int16 -> uint64     ✅                   
INFO[0000]      int16 -> int8       ✅                   
INFO[0000]      int16 -> int16      ✅                   
INFO[0000]      int16 -> int32      ✅                   
INFO[0000]      int16 -> int64      ✅                   
INFO[0000]      int16 -> float32    ✅                   
INFO[0000]      int16 -> float64    ✅                   
INFO[0000]      int16 -> complex64  ❌                   
INFO[0000]      int16 -> complex128 ❌                   
INFO[0000]      int16 -> string     ✅                   
INFO[0000]      int16 -> int        ✅                   
INFO[0000]      int16 -> uint       ✅                   
INFO[0000]      int16 -> uintptr    ✅                   
INFO[0000]      int16 -> byte       ✅                   
INFO[0000]      int16 -> rune       ✅                   
INFO[0000] ---------- converting int32 values ---------- 
INFO[0000]      int32 -> bool       ❌                   
INFO[0000]      int32 -> uint8      ✅                   
INFO[0000]      int32 -> uint16     ✅                   
INFO[0000]      int32 -> uint32     ✅                   
INFO[0000]      int32 -> uint64     ✅                   
INFO[0000]      int32 -> int8       ✅                   
INFO[0000]      int32 -> int16      ✅                   
INFO[0000]      int32 -> int32      ✅                   
INFO[0000]      int32 -> int64      ✅                   
INFO[0000]      int32 -> float32    ✅                   
INFO[0000]      int32 -> float64    ✅                   
INFO[0000]      int32 -> complex64  ❌                   
INFO[0000]      int32 -> complex128 ❌                   
INFO[0000]      int32 -> string     ✅                   
INFO[0000]      int32 -> int        ✅                   
INFO[0000]      int32 -> uint       ✅                   
INFO[0000]      int32 -> uintptr    ✅                   
INFO[0000]      int32 -> byte       ✅                   
INFO[0000]      int32 -> rune       ✅                   
INFO[0000] ---------- converting int64 values ---------- 
INFO[0000]      int64 -> bool       ❌                   
INFO[0000]      int64 -> uint8      ✅                   
INFO[0000]      int64 -> uint16     ✅                   
INFO[0000]      int64 -> uint32     ✅                   
INFO[0000]      int64 -> uint64     ✅                   
INFO[0000]      int64 -> int8       ✅                   
INFO[0000]      int64 -> int16      ✅                   
INFO[0000]      int64 -> int32      ✅                   
INFO[0000]      int64 -> int64      ✅                   
INFO[0000]      int64 -> float32    ✅                   
INFO[0000]      int64 -> float64    ✅                   
INFO[0000]      int64 -> complex64  ❌                   
INFO[0000]      int64 -> complex128 ❌                   
INFO[0000]      int64 -> string     ✅                   
INFO[0000]      int64 -> int        ✅                   
INFO[0000]      int64 -> uint       ✅                   
INFO[0000]      int64 -> uintptr    ✅                   
INFO[0000]      int64 -> byte       ✅                   
INFO[0000]      int64 -> rune       ✅                   
INFO[0000] ---------- converting float32 values ---------- 
INFO[0000]    float32 -> bool       ❌                   
INFO[0000]    float32 -> uint8      ✅                   
INFO[0000]    float32 -> uint16     ✅                   
INFO[0000]    float32 -> uint32     ✅                   
INFO[0000]    float32 -> uint64     ✅                   
INFO[0000]    float32 -> int8       ✅                   
INFO[0000]    float32 -> int16      ✅                   
INFO[0000]    float32 -> int32      ✅                   
INFO[0000]    float32 -> int64      ✅                   
INFO[0000]    float32 -> float32    ✅                   
INFO[0000]    float32 -> float64    ✅                   
INFO[0000]    float32 -> complex64  ❌                   
INFO[0000]    float32 -> complex128 ❌                   
INFO[0000]    float32 -> string     ❌                   
INFO[0000]    float32 -> int        ✅                   
INFO[0000]    float32 -> uint       ✅                   
INFO[0000]    float32 -> uintptr    ✅                   
INFO[0000]    float32 -> byte       ✅                   
INFO[0000]    float32 -> rune       ✅                   
INFO[0000] ---------- converting float64 values ---------- 
INFO[0000]    float64 -> bool       ❌                   
INFO[0000]    float64 -> uint8      ✅                   
INFO[0000]    float64 -> uint16     ✅                   
INFO[0000]    float64 -> uint32     ✅                   
INFO[0000]    float64 -> uint64     ✅                   
INFO[0000]    float64 -> int8       ✅                   
INFO[0000]    float64 -> int16      ✅                   
INFO[0000]    float64 -> int32      ✅                   
INFO[0000]    float64 -> int64      ✅                   
INFO[0000]    float64 -> float32    ✅                   
INFO[0000]    float64 -> float64    ✅                   
INFO[0000]    float64 -> complex64  ❌                   
INFO[0000]    float64 -> complex128 ❌                   
INFO[0000]    float64 -> string     ❌                   
INFO[0000]    float64 -> int        ✅                   
INFO[0000]    float64 -> uint       ✅                   
INFO[0000]    float64 -> uintptr    ✅                   
INFO[0000]    float64 -> byte       ✅                   
INFO[0000]    float64 -> rune       ✅                   
INFO[0000] ---------- converting complex64 values ---------- 
INFO[0000]  complex64 -> bool       ❌                   
INFO[0000]  complex64 -> uint8      ❌                   
INFO[0000]  complex64 -> uint16     ❌                   
INFO[0000]  complex64 -> uint32     ❌                   
INFO[0000]  complex64 -> uint64     ❌                   
INFO[0000]  complex64 -> int8       ❌                   
INFO[0000]  complex64 -> int16      ❌                   
INFO[0000]  complex64 -> int32      ❌                   
INFO[0000]  complex64 -> int64      ❌                   
INFO[0000]  complex64 -> float32    ❌                   
INFO[0000]  complex64 -> float64    ❌                   
INFO[0000]  complex64 -> complex64  ✅                   
INFO[0000]  complex64 -> complex128 ✅                   
INFO[0000]  complex64 -> string     ❌                   
INFO[0000]  complex64 -> int        ❌                   
INFO[0000]  complex64 -> uint       ❌                   
INFO[0000]  complex64 -> uintptr    ❌                   
INFO[0000]  complex64 -> byte       ❌                   
INFO[0000]  complex64 -> rune       ❌                   
INFO[0000] ---------- converting complex128 values ---------- 
INFO[0000] complex128 -> bool       ❌                   
INFO[0000] complex128 -> uint8      ❌                   
INFO[0000] complex128 -> uint16     ❌                   
INFO[0000] complex128 -> uint32     ❌                   
INFO[0000] complex128 -> uint64     ❌                   
INFO[0000] complex128 -> int8       ❌                   
INFO[0000] complex128 -> int16      ❌                   
INFO[0000] complex128 -> int32      ❌                   
INFO[0000] complex128 -> int64      ❌                   
INFO[0000] complex128 -> float32    ❌                   
INFO[0000] complex128 -> float64    ❌                   
INFO[0000] complex128 -> complex64  ✅                   
INFO[0000] complex128 -> complex128 ✅                   
INFO[0000] complex128 -> string     ❌                   
INFO[0000] complex128 -> int        ❌                   
INFO[0000] complex128 -> uint       ❌                   
INFO[0000] complex128 -> uintptr    ❌                   
INFO[0000] complex128 -> byte       ❌                   
INFO[0000] complex128 -> rune       ❌                   
INFO[0000] ---------- converting string values ---------- 
INFO[0000]     string -> bool       ❌                   
INFO[0000]     string -> uint8      ❌                   
INFO[0000]     string -> uint16     ❌                   
INFO[0000]     string -> uint32     ❌                   
INFO[0000]     string -> uint64     ❌                   
INFO[0000]     string -> int8       ❌                   
INFO[0000]     string -> int16      ❌                   
INFO[0000]     string -> int32      ❌                   
INFO[0000]     string -> int64      ❌                   
INFO[0000]     string -> float32    ❌                   
INFO[0000]     string -> float64    ❌                   
INFO[0000]     string -> complex64  ❌                   
INFO[0000]     string -> complex128 ❌                   
INFO[0000]     string -> string     ✅                   
INFO[0000]     string -> int        ❌                   
INFO[0000]     string -> uint       ❌                   
INFO[0000]     string -> uintptr    ❌                   
INFO[0000]     string -> byte       ❌                   
INFO[0000]     string -> rune       ❌                   
INFO[0000] ---------- converting int values ----------  
INFO[0000]        int -> bool       ❌                   
INFO[0000]        int -> uint8      ✅                   
INFO[0000]        int -> uint16     ✅                   
INFO[0000]        int -> uint32     ✅                   
INFO[0000]        int -> uint64     ✅                   
INFO[0000]        int -> int8       ✅                   
INFO[0000]        int -> int16      ✅                   
INFO[0000]        int -> int32      ✅                   
INFO[0000]        int -> int64      ✅                   
INFO[0000]        int -> float32    ✅                   
INFO[0000]        int -> float64    ✅                   
INFO[0000]        int -> complex64  ❌                   
INFO[0000]        int -> complex128 ❌                   
INFO[0000]        int -> string     ✅                   
INFO[0000]        int -> int        ✅                   
INFO[0000]        int -> uint       ✅                   
INFO[0000]        int -> uintptr    ✅                   
INFO[0000]        int -> byte       ✅                   
INFO[0000]        int -> rune       ✅                   
INFO[0000] ---------- converting uint values ---------- 
INFO[0000]       uint -> bool       ❌                   
INFO[0000]       uint -> uint8      ✅                   
INFO[0000]       uint -> uint16     ✅                   
INFO[0000]       uint -> uint32     ✅                   
INFO[0000]       uint -> uint64     ✅                   
INFO[0000]       uint -> int8       ✅                   
INFO[0000]       uint -> int16      ✅                   
INFO[0000]       uint -> int32      ✅                   
INFO[0000]       uint -> int64      ✅                   
INFO[0000]       uint -> float32    ✅                   
INFO[0000]       uint -> float64    ✅                   
INFO[0000]       uint -> complex64  ❌                   
INFO[0000]       uint -> complex128 ❌                   
INFO[0000]       uint -> string     ✅                   
INFO[0000]       uint -> int        ✅                   
INFO[0000]       uint -> uint       ✅                   
INFO[0000]       uint -> uintptr    ✅                   
INFO[0000]       uint -> byte       ✅                   
INFO[0000]       uint -> rune       ✅                   
INFO[0000] ---------- converting uintptr values ---------- 
INFO[0000]    uintptr -> bool       ❌                   
INFO[0000]    uintptr -> uint8      ✅                   
INFO[0000]    uintptr -> uint16     ✅                   
INFO[0000]    uintptr -> uint32     ✅                   
INFO[0000]    uintptr -> uint64     ✅                   
INFO[0000]    uintptr -> int8       ✅                   
INFO[0000]    uintptr -> int16      ✅                   
INFO[0000]    uintptr -> int32      ✅                   
INFO[0000]    uintptr -> int64      ✅                   
INFO[0000]    uintptr -> float32    ✅                   
INFO[0000]    uintptr -> float64    ✅                   
INFO[0000]    uintptr -> complex64  ❌                   
INFO[0000]    uintptr -> complex128 ❌                   
INFO[0000]    uintptr -> string     ✅                   
INFO[0000]    uintptr -> int        ✅                   
INFO[0000]    uintptr -> uint       ✅                   
INFO[0000]    uintptr -> uintptr    ✅                   
INFO[0000]    uintptr -> byte       ✅                   
INFO[0000]    uintptr -> rune       ✅                   
INFO[0000] ---------- converting byte values ---------- 
INFO[0000]       byte -> bool       ❌                   
INFO[0000]       byte -> uint8      ✅                   
INFO[0000]       byte -> uint16     ✅                   
INFO[0000]       byte -> uint32     ✅                   
INFO[0000]       byte -> uint64     ✅                   
INFO[0000]       byte -> int8       ✅                   
INFO[0000]       byte -> int16      ✅                   
INFO[0000]       byte -> int32      ✅                   
INFO[0000]       byte -> int64      ✅                   
INFO[0000]       byte -> float32    ✅                   
INFO[0000]       byte -> float64    ✅                   
INFO[0000]       byte -> complex64  ❌                   
INFO[0000]       byte -> complex128 ❌                   
INFO[0000]       byte -> string     ✅                   
INFO[0000]       byte -> int        ✅                   
INFO[0000]       byte -> uint       ✅                   
INFO[0000]       byte -> uintptr    ✅                   
INFO[0000]       byte -> byte       ✅                   
INFO[0000]       byte -> rune       ✅                   
INFO[0000] ---------- converting rune values ---------- 
INFO[0000]       rune -> bool       ❌                   
INFO[0000]       rune -> uint8      ✅                   
INFO[0000]       rune -> uint16     ✅                   
INFO[0000]       rune -> uint32     ✅                   
INFO[0000]       rune -> uint64     ✅                   
INFO[0000]       rune -> int8       ✅                   
INFO[0000]       rune -> int16      ✅                   
INFO[0000]       rune -> int32      ✅                   
INFO[0000]       rune -> int64      ✅                   
INFO[0000]       rune -> float32    ✅                   
INFO[0000]       rune -> float64    ✅                   
INFO[0000]       rune -> complex64  ❌                   
INFO[0000]       rune -> complex128 ❌                   
INFO[0000]       rune -> string     ✅                   
INFO[0000]       rune -> int        ✅                   
INFO[0000]       rune -> uint       ✅                   
INFO[0000]       rune -> uintptr    ✅                   
INFO[0000]       rune -> byte       ✅                   
INFO[0000]       rune -> rune       ✅                   
```

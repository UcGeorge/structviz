# structviz

A CLI tool that visualizes the memory layout of Go structs, shows wasted bytes from padding, and automatically outputs an optimally packed version using the greedy sort-by-alignment algorithm.

## What it does

1. Parses any Go struct definition you paste into it
2. Renders a color-coded byte grid of the **current** layout
   - Field bytes are colored (one color per field)
   - Internal (avoidable) padding is **red**
   - Trailing (unavoidable) padding is **gray**
3. Applies the optimal field ordering (largest alignment first) and renders the **optimized** layout
4. Prints the ready-to-use optimized struct definition

## Why this matters

Go (like C) inserts padding bytes between struct fields to keep each field aligned to its natural boundary. The order of your fields determines how much padding is inserted. Sorting fields from largest to smallest alignment eliminates all internal padding — this is provably optimal because all Go alignments are powers of two.

## Requirements

- Go 1.21 or later
- A terminal with ANSI color support (any modern terminal)

## Install

```sh
go install github.com/UcGeorge/structviz@latest
```

Or clone and run locally:

```sh
git clone https://github.com/UcGeorge/structviz
cd structviz
go build -o structviz .
```

## Usage

Run the tool, paste your struct, then press **Ctrl+D** (macOS/Linux) or **Ctrl+Z** (Windows):

```sh
structviz
```

Or pipe directly:

```sh
cat mystruct.go | structviz
```

```sh
echo 'type Request struct {
    Active bool
    ID     int64
    Code   int32
    Name   string
    Flag   bool
    Count  int16
}' | structviz
```

## Supported input formats

All three forms are accepted:

```go
// Full type declaration (recommended)
type MyStruct struct {
    ...
}

// Bare struct keyword
struct {
    ...
}

// Just the body
{
    ...
}
```

## Supported field types

All built-in Go types:

| Types | Size | Alignment |
|---|---|---|
| `bool`, `byte`, `int8`, `uint8` | 1 | 1 |
| `int16`, `uint16` | 2 | 2 |
| `int32`, `uint32`, `float32`, `rune` | 4 | 4 |
| `int`, `uint`, `uintptr`, `int64`, `uint64`, `float64`, `complex64` | 8 | 8 |
| `string`, `complex128`, `error` | 16 | 8 |
| `[]T` (slice header) | 24 | 8 |
| `*T`, `map[K]V`, `chan T`, `func(...)` | 8 | 8 |
| `interface{}` | 16 | 8 |
| `[N]T` (array) | N × sizeof(T) | align(T) |
| anonymous `struct{...}` | computed recursively | — |

External package types (e.g. `time.Time`, `sync.Mutex`) are not supported — use only built-in types or inline the fields manually.

## How the grid works

Each row of the grid is exactly **alignment** bytes wide, where alignment is the largest field alignment in the struct. This mirrors how memory is actually laid out relative to the struct's own alignment requirement.

```
       0     1     2     3     4     5     6     7
       ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐
    0: │Name │     │     │     │     │     │     │     │
       ├─────┼─────┼─────┼─────┼─────┼─────┼─────┼─────┤
    8: │Name │     │     │     │     │     │     │     │
       ├─────┼─────┼─────┼─────┼─────┼─────┼─────┼─────┤
   16: │ID   │     │     │     │     │     │     │     │
       └─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘
```

- **Colored cells** — one field, one color; the first byte shows the field name
- **Red cells (▒)** — internal padding; bytes wasted because of misaligned field order
- **Gray cells (░)** — trailing padding; unavoidable, determined by the largest alignment

## The optimization algorithm

Sorting fields by alignment descending is always optimal (not just heuristic) because:

- All Go alignment values are powers of 2
- After placing a field with alignment A, the current offset is always already a multiple of any smaller alignment B (since A is a multiple of B)
- This means no internal padding ever occurs
- The only remaining padding is the trailing alignment padding, which is a fixed lower bound regardless of field order

## Example output

```
ORIGINAL: Request
────────────────────────────────────────────────────────────
  size: 48 bytes   alignment: 8   internal padding: 12 bytes   trailing padding: 4 bytes   total waste: 16 bytes

OPTIMIZED: Request
────────────────────────────────────────────────────────────
  size: 32 bytes   alignment: 8   internal padding: 0 bytes   trailing padding: 0 bytes   total waste: 0 bytes
  Saved 16 bytes (33% reduction)

type Request struct {
    Name   string
    ID     int64
    Code   int32
    Count  int16
    Active bool
    Flag   bool
}
```

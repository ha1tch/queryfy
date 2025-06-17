# Baseline (default: atoms + simd + array + pooling)
export PROGRAM="superjsonic_multi.go"

go run $PROGRAM

# Test with no optimizations
go run $PROGRAM -none

# Test only SIMD
go run $PROGRAM -none -simd

# Test only atoms
go run $PROGRAM -none -atoms

# Test SIMD + atoms (no pooling/array)
go run $PROGRAM -none -simd -atoms

# Test all optimizations
go run $PROGRAM -all

# Test without atoms (everything else enabled)
go run $PROGRAM -atoms=false

# Test impact of stringview
go run $PROGRAM -stringview

# Test 16-byte SIMD
go run $PROGRAM -simd16

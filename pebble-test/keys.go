package main

import (
    "encoding/binary"
    "math"
)

// keyLSS composes the Latest State Store key: s/<userKey>
func keyLSS(user []byte) []byte {
    out := make([]byte, 0, 2+len(user))
    out = append(out, 's', '/')
    out = append(out, user...)
    return out
}

// keyHSS composes the Historical State Store key: h/<heightBE(8B)>/<userKey>
func keyHSS(height uint64, user []byte) []byte {
    out := make([]byte, 0, 2+8+1+len(user))
    out = append(out, 'h', '/')
    var hb [8]byte
    binary.BigEndian.PutUint64(hb[:], height)
    out = append(out, hb[:]...)
    out = append(out, '/')
    out = append(out, user...)
    return out
}

// boundsForHeight returns LowerBound/UpperBound for a given historical height partition.
func boundsForHeight(height uint64) (lb, ub []byte) {
    lb = make([]byte, 0, 2+8+1)
    lb = append(lb, 'h', '/')
    var hb [8]byte
    binary.BigEndian.PutUint64(hb[:], height)
    lb = append(lb, hb[:]...)
    lb = append(lb, '/')

    // UpperBound is exclusive; cover the entire prefix by using next height's prefix.
    next := height + 1
    if next != 0 { // overflow ⇒ no UB
        var nb [8]byte
        binary.BigEndian.PutUint64(nb[:], next)
        ub = make([]byte, 0, 2+8+1)
        ub = append(ub, 'h', '/')
        ub = append(ub, nb[:]...)
        ub = append(ub, '/')
    } else {
        ub = nil
    }
    return
}

// boundsForHeightRange returns LowerBound/UpperBound for a contiguous height window [min,max].
func boundsForHeightRange(min, max uint64) (lb, ub []byte) {
    // Reject invalid windows to avoid accidental wide scans
    if min > max {
        return nil, nil
    }

    // lb = h/<min>/
    lb = make([]byte, 0, 2+8+1)
    lb = append(lb, 'h', '/')
    var lbb [8]byte
    binary.BigEndian.PutUint64(lbb[:], min)
    lb = append(lb, lbb[:]...)
    lb = append(lb, '/')

    // Build exclusive upper bound using next uint64 prefix
    if max != math.MaxUint64 {
        // Use next height's prefix: h/<max+1>/
        next := max + 1
        ub = make([]byte, 0, 2+8+1)
        ub = append(ub, 'h', '/')
        var ubb [8]byte
        binary.BigEndian.PutUint64(ubb[:], next)
        ub = append(ub, ubb[:]...)
        ub = append(ub, '/')
    } else {
        // Overflow case: fall back to sentinel ub h/<max>/0xFF
        ub = make([]byte, 0, 2+8+2)
        ub = append(ub, 'h', '/')
        var ubb [8]byte
        binary.BigEndian.PutUint64(ubb[:], max)
        ub = append(ub, ubb[:]...)
        ub = append(ub, '/')
        ub = append(ub, 0xFF)
    }
    return
}

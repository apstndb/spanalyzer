"""Reject one precise FarmHash naming family, not all unknown naming functions.

For the three 11-byte signatures, any common 17-byte prefix produces 28-byte
inputs with common first 20 bytes. Google's public HashLen17to32 therefore
reduces to HashLen16(A + ROR(c,30), B + c, k2 + 56), with common uint64 A/B.
Allowing arbitrary A/B over-approximates the states reachable from a prefix.
UNSAT rejects this entire prefix family; SAT/unknown would not reject it.
See the Research Note for the pinned public source and observed target names.
"""
import json
import random

import farmhash
import z3

MASK = (1 << 64) - 1
K1 = 0xB492B66FBE98F273
K2 = 0x9AE16A3B2F90404F
MUL = (K2 + 56) & MASK
PAIRS = [
    ("IDX_T_X_Y_U", 0x026F836781A5FCAF),
    ("IDX_U_A_B_N", 0x8FE1FF22505F93E5),
    ("IDX_A_B_C_N", 0x99E878FAF8057D7D),
]


def rotate(value, bits):
    return ((value >> bits) | (value << (64 - bits))) & MASK


def model(a, b, signature):
    c = (int.from_bytes(signature.encode()[-8:], "little") * MUL) & MASK
    left, right = a + rotate(c, 30), b + c
    first = (left ^ right) * MUL
    first ^= z3.LShR(first, 47)
    second = (right ^ first) * MUL
    second ^= z3.LShR(second, 47)
    return second * MUL


def main():
    # Check the model against an independent native implementation for binary
    # prefixes, not only the particular printable Emulator seed.
    rng = random.Random(20261113)
    for _ in range(100):
        prefix = bytes(rng.randrange(256) for _ in range(17))
        for signature, _ in PAIRS:
            data = prefix + signature.encode()
            assert len(data) == 28
            first = (int.from_bytes(data[:8], "little") * K1) & MASK
            second = int.from_bytes(data[8:16], "little")
            fourth = (int.from_bytes(data[12:20], "little") * K2) & MASK
            a = (rotate((first + second) & MASK, 43) + fourth) & MASK
            b = (first + rotate((second + K2) & MASK, 18)) & MASK
            got = z3.simplify(model(z3.BitVecVal(a, 64), z3.BitVecVal(b, 64), signature)).as_long()
            assert got == farmhash.Fingerprint64(data)
    a, b = z3.BitVecs("a b", 64)
    results = []
    for scope, pairs in [("retained-managed-and-omni", PAIRS[:2]), ("omni-with-new-collision-control", PAIRS)]:
        solver = z3.Solver()
        solver.set(timeout=30000)
        for signature, target in pairs:
            solver.add(model(a, b, signature) == target)
        result = solver.check()
        results.append({"scope": scope, "status": str(result), "pairs": [(s, f"{h:016X}") for s, h in pairs]})
        assert result == z3.unsat, "This run does not establish rejection of the modeled family"
    print(json.dumps({"family": "common17-byte prefix plus complete11-byte signature under FarmHash Fingerprint64", "native_controls": 300, "timeout_ms": 30000, "results": results}))


if __name__ == "__main__":
    main()

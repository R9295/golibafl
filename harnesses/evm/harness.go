package main

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -levm
#include <stdint.h>
#include <stdlib.h>

typedef struct {
    uint8_t *data;
    size_t   len;
} EvmByteBuffer;

EvmByteBuffer arbitrary_state_test(const uint8_t *data, size_t len, uint64_t gas);
void           evm_byte_buffer_free(EvmByteBuffer buffer);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto/keccak"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests"
)

const fuzzGas = 100_000

var (
	lastStateRoot common.Hash
	lastLogsRoot  common.Hash
)

func harness(data []byte) {
	if len(data) == 0 {
		return
	}
	buf := C.arbitrary_state_test(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		C.uint64_t(fuzzGas),
	)
	if buf.data == nil || buf.len == 0 {
		return
	}
	defer C.evm_byte_buffer_free(buf)

	jsonBytes := C.GoBytes(unsafe.Pointer(buf.data), C.int(buf.len))

	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &wrapper); err != nil {
		return
	}

	for _, body := range wrapper {
		var st tests.StateTest
		if err := st.UnmarshalJSON(body); err != nil {
			continue
		}
		for _, sub := range st.Subtests() {
			runSubtest(&st, sub)
		}
	}
}

func runSubtest(st *tests.StateTest, sub tests.StateSubtest) {
	state, root, _, err := st.RunNoVerify(sub, vm.Config{}, false, rawdb.HashScheme)
	defer state.Close()
	if err != nil {
		return
	}
	lastStateRoot = root
	lastLogsRoot = rlpHash(state.StateDB.Logs())
	fmt.Println(state, lastLogsRoot)
}

func rlpHash(x interface{}) common.Hash {
	var h common.Hash
	hw := keccak.NewLegacyKeccak256()
	_ = rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

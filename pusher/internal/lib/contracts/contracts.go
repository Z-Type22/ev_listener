package contracts

import (
	"bytes"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func LoadABI(path string) (abi.ABI, error) {

	file, err := os.ReadFile(path)
	if err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(bytes.NewReader(file))
}

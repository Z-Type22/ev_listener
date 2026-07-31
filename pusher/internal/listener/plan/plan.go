package plan

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type PlanData struct {
	Hash      [32]byte
	Merchant  common.Address
	Price     *big.Int
	Period    uint32
	Token     common.Address
	Status    uint8
	CreatedAt *big.Int
	UpdatedAt *big.Int
	Uri       string
}

type PlanManager struct {
	Address     common.Address
	ContractABI abi.ABI
	Client      *ethclient.Client
}

func (p *PlanManager) GetPlan(ctx context.Context, hash common.Hash) (*PlanData, error) {
	data, err := p.ContractABI.Pack("getPlan", hash)
	if err != nil {
		return nil, err
	}

	message := ethereum.CallMsg{To: &p.Address, Data: data}

	result, err := p.Client.CallContract(ctx, message, nil)
	if err != nil {
		return nil, err
	}

	var output struct{ Plan PlanData }

	err = p.ContractABI.UnpackIntoInterface(&output, "getPlan", result)
	if err != nil {
		return nil, err
	}

	return &output.Plan, nil
}

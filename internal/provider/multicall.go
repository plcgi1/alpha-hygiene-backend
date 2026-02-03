package provider

import (
	"alpha-hygiene-backend/config"
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

// MulticallClient - Клиент для работы с Multicall контрактом
type MulticallClient struct {
	client        *ethclient.Client
	multicallAddr common.Address
	contractABI   abi.ABI
	log           *logrus.Entry
}

// NewMulticallClient - Создает новый клиент для Multicall контракта
func NewMulticallClient(cfg *config.Config, log *logrus.Entry) (*MulticallClient, error) {
	// Use RPC URL from config or default to Alchemy
	rpcURL := cfg.Alchemy.URL
	if rpcURL == "" {
		rpcURL = "https://eth-mainnet.g.alchemy.com/v2/" + cfg.Alchemy.ApiKey
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	// Multicall контакт (MakerDAO) на Ethereum Mainnet
	multicallAddr := common.HexToAddress("0x5e22962C96d7fC8b4e8a01D48D9ae490D309e2A7")

	// ABI для Multicall контракта
	multicallABI, err := abi.JSON(strings.NewReader(`[
		{
			"constant": true,
			"inputs": [
				{
					"components": [
						{
							"name": "target",
							"type": "address"
						},
						{
							"name": "callData",
							"type": "bytes"
						}
					],
					"name": "calls",
					"type": "tuple[]"
				}
			],
			"name": "aggregate",
			"outputs": [
				{
					"name": "blockNumber",
					"type": "uint256"
				},
				{
					"name": "returnData",
					"type": "bytes[]"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		}
	]`))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	logger := log.WithFields(logrus.Fields{"component": "multicall"})
	logger.Info("Successfully connected to Multicall contract")

	return &MulticallClient{
		client:        client,
		multicallAddr: multicallAddr,
		contractABI:   multicallABI,
		log:           logger,
	}, nil
}

// Call - Структура для одиночного вызова в batch
type Call struct {
	Target   common.Address `json:"target"`
	CallData []byte         `json:"callData"`
}

// CallResult - Результат вызова контракта
type CallResult struct {
	Success bool
	Data    []byte
}

// Aggregate - Выполняет batch вызовы контрактов
func (c *MulticallClient) Aggregate(ctx context.Context, calls []Call) ([]CallResult, error) {
	// Подготавливаем параметры для вызова
	var args []interface{} = []interface{}{calls}

	// Кодируем данные для вызова контракта
	data, err := c.contractABI.Pack("aggregate", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode call data: %w", err)
	}

	// Выполняем статический вызов контракта
	result, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.multicallAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// Декодируем результат
	var decodedResult struct {
		BlockNumber *big.Int
		ReturnData  [][]byte
	}
	err = c.contractABI.UnpackIntoInterface(&decodedResult, "aggregate", result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode contract call result: %w", err)
	}

	// Преобразуем в CallResult
	callResults := make([]CallResult, len(decodedResult.ReturnData))
	for i, data := range decodedResult.ReturnData {
		// Проверка на успешность вызова
		// Для простоты считаем что все вызовы успешные, если они вернули данные
		// В реальной реализации нужно добавить проверку на ошибки
		success := true
		if len(data) == 0 {
			success = false
		}

		callResults[i] = CallResult{
			Success: success,
			Data:    data,
		}
	}

	c.log.Debugf("Multicall executed %d calls", len(calls))
	return callResults, nil
}

// GetAllowances - Получает allowances для нескольких токенов за один запрос
func (c *MulticallClient) GetAllowances(ctx context.Context, owner common.Address, spender common.Address, tokenAddresses []common.Address) (map[string]*big.Int, error) {
	// ABI для метода allowance на ERC20 токене
	const erc20AllowanceMethod = "allowance(address,address)"
	const erc20AllowanceMethodID = "0xdd62ed3e"

	// Подготавливаем вызовы для каждого токена
	var calls []Call
	for _, tokenAddr := range tokenAddresses {
		// Кодируем параметры вызова allowance для каждого токена
		data := []byte(erc20AllowanceMethodID)
		// Кодируем owner и spender как 32-byte адреса
		data = append(data, common.LeftPadBytes(owner.Bytes(), 32)...)
		data = append(data, common.LeftPadBytes(spender.Bytes(), 32)...)

		calls = append(calls, Call{
			Target:   tokenAddr,
			CallData: data,
		})
	}

	// Выполняем batch вызов
	results, err := c.Aggregate(ctx, calls)
	if err != nil {
		return nil, fmt.Errorf("failed to execute batch call: %w", err)
	}

	// Декодируем результаты
	allowances := make(map[string]*big.Int)
	for i, tokenAddr := range tokenAddresses {
		if !results[i].Success {
			c.log.Warnf("Call for token %s failed", tokenAddr.Hex())
			continue
		}

		// Результат вызова должен быть uint256 (32 байта)
		if len(results[i].Data) != 32 {
			c.log.Warnf("Unexpected data length for token %s: %d bytes", tokenAddr.Hex(), len(results[i].Data))
			continue
		}

		allowance := new(big.Int).SetBytes(results[i].Data)
		allowances[tokenAddr.Hex()] = allowance
	}

	c.log.Debugf("Successfully retrieved allowances for %d tokens", len(allowances))
	return allowances, nil
}

// Close - Закрывает подключение к RPC
func (c *MulticallClient) Close() error {
	c.client.Close()
	c.log.Info("Multicall RPC connection closed")
	return nil
}

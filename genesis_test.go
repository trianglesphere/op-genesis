package opgenesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type TestCase struct {
	name         string
	path         string
	chainID      uint64
	gethOverride func(*params.ChainConfig)
	nodeOverride func(*rollup.Config)
}

func u64p(x uint64) *uint64 {
	return &x
}

var (
	MainnetProtocolVersionsAddress = common.HexToAddress("0x8062AbC286f5e7D9428a0Ccb9AbD71e50d93b935")
	SepoliaProtocolVersionsAddress = common.HexToAddress("0x79ADD5713B383DAa0a138d3C4780C7A1804a8090")
	GoerliProtocolVersionsAddress  = common.HexToAddress("0x0C24F5098774aA366827D667494e9F889f7cFc08")

	MainnetCanyonTime = u64p(1704992401)
	SepoliaCanyonTime = u64p(1699981200)
	GoerliCanyonTime  = u64p(1699981200)

	// MainnetDeltaTime = u64p()
	SepoliaDeltaTime = u64p(1703203200)
	GoerliDeltaTime  = u64p(1703116800)
)

var mainnetGethOverride = func(cfg *params.ChainConfig) {
	cfg.ShanghaiTime = MainnetCanyonTime
	cfg.CanyonTime = MainnetCanyonTime
	cfg.Optimism.EIP1559DenominatorCanyon = 250
}

var mainnetNodeOverride = func(cfg *rollup.Config) {
	cfg.CanyonTime = MainnetCanyonTime
	cfg.ProtocolVersionsAddress = MainnetProtocolVersionsAddress
}

var sepoliaGethOverride = func(cfg *params.ChainConfig) {
	cfg.ShanghaiTime = SepoliaCanyonTime
	cfg.CanyonTime = SepoliaCanyonTime
	cfg.Optimism.EIP1559DenominatorCanyon = 250
}

var sepoliaNodeOverride = func(cfg *rollup.Config) {
	cfg.CanyonTime = SepoliaCanyonTime
	cfg.DeltaTime = SepoliaDeltaTime
	cfg.ProtocolVersionsAddress = SepoliaProtocolVersionsAddress
}

func TestConfigs(t *testing.T) {
	tests := []TestCase{
		// Mainnet
		{
			name:         "Base Mainnet",
			path:         "data/mainnet/base",
			chainID:      8453,
			gethOverride: mainnetGethOverride,
			nodeOverride: mainnetNodeOverride,
		},
		{
			name:         "PGN Mainnet",
			path:         "data/mainnet/pgn",
			chainID:      424,
			gethOverride: mainnetGethOverride,
			nodeOverride: mainnetNodeOverride,
		},
		{
			name:         "Zora Mainnet",
			path:         "data/mainnet/zora",
			chainID:      7777777,
			gethOverride: mainnetGethOverride,
			nodeOverride: mainnetNodeOverride,
		},
		// Sepolia
		{
			name:         "Base Sepolia",
			path:         "data/sepolia/base",
			chainID:      84532,
			gethOverride: sepoliaGethOverride,
			nodeOverride: sepoliaNodeOverride,
		},
		{
			name:         "PGN Sepolia",
			path:         "data/sepolia/pgn",
			chainID:      58008,
			gethOverride: sepoliaGethOverride,
			nodeOverride: sepoliaNodeOverride,
		},
		{
			name:         "Zora Sepolia",
			path:         "data/sepolia/zora",
			chainID:      999999999,
			gethOverride: sepoliaGethOverride,
			nodeOverride: sepoliaNodeOverride,
		},
		// Goerli
		{
			name:    "Base Goerli",
			path:    "data/goerli/base",
			chainID: 84531,
			gethOverride: func(cfg *params.ChainConfig) {
				cfg.RegolithTime = u64p(1683219600) // Not set in Base Genesis but set in base rollup.json
				cfg.ShanghaiTime = GoerliCanyonTime
				cfg.CanyonTime = GoerliCanyonTime
				cfg.Optimism.EIP1559DenominatorCanyon = 250
			},
			nodeOverride: func(cfg *rollup.Config) {
				cfg.CanyonTime = GoerliCanyonTime
				cfg.DeltaTime = GoerliDeltaTime
				cfg.ProtocolVersionsAddress = GoerliProtocolVersionsAddress
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, test.Run)
	}
}

func (tc *TestCase) Run(t *testing.T) {
	rollupPath := fmt.Sprintf("%s/rollup.json", tc.path)
	genesisPath := fmt.Sprintf("%s/genesis.json", tc.path)
	testRollupConfig(t, rollupPath, tc.chainID, tc.nodeOverride)
	testGenesisConfig(t, genesisPath, tc.chainID, tc.gethOverride)
	testGenesisHash(t, genesisPath, tc.chainID)
}

func testRollupConfig(t *testing.T, path string, chainID uint64, override func(*rollup.Config)) {
	var config rollup.Config
	err := readJson(path, &config)
	require.NoError(t, err)

	config2, err := rollup.LoadOPStackRollupConfig(chainID)
	require.NoError(t, err)

	// Apply overrides & assert that the override is necessary (to prevent stale overrides)
	if override != nil {
		require.NotEqual(t, &config, config2, "When using overrides, the pre-overide config should not be the same as the superchain registry config")
		override(&config)
	}
	require.Equal(t, &config, config2)
}

func testGenesisConfig(t *testing.T, path string, chainID uint64, override func(*params.ChainConfig)) {
	var genesis core.Genesis
	err := readJson(path, &genesis)
	require.NoError(t, err)

	chainConfig, err := params.LoadOPStackChainConfig(chainID)
	require.NoError(t, err)

	// Apply overrides & assert that the override is necessary (to prevent stale overrides)
	if override != nil {
		require.NotEqual(t, genesis.Config, chainConfig, "When using overrides, the pre-overide config should not be the same as the superchain registry config")
		override(genesis.Config)
	}
	require.Equal(t, genesis.Config, chainConfig)
}

func testGenesisHash(t *testing.T, path string, chainID uint64) {
	var genesis core.Genesis
	err := readJson(path, &genesis)
	require.NoError(t, err)

	genesis2, err := core.LoadOPStackGenesis(chainID)
	require.NoError(t, err)

	genesisHash := genesis.ToBlock().Hash()
	genesis2Hash := genesis2.ToBlock().Hash()

	require.Equal(t, genesisHash, genesis2Hash, "Genesis block hash must match")
}

func readJson(path string, out any) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &out)
	if err != nil {
		return err
	}
	return nil
}

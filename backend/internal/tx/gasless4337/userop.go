package gasless4337

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"log"
)

// ── EntryPoint Nonce 查询 ─────────────────────────────────────────────────────

// GetEntryPointNonce 通过 eth_call 查询 EntryPoint.getNonce(sender, key=0)。
// 返回值是 UserOperation 的 nonce 字段（uint256，packed 格式：高 192 位是 key，低 64 位是 seq）。
func GetEntryPointNonce(ctx context.Context, rpcURL string, sender common.Address, entryPoint string) (*big.Int, error) {
	// getNonce(address sender, uint192 key) 的 ABI 编码
	// selector = keccak256("getNonce(address,uint192)")[:4] = 0x35567e1a
	selector := []byte{0x35, 0x56, 0x7e, 0x1a}
	// abi encode(sender, 0)
	paddedSender := common.LeftPadBytes(sender.Bytes(), 32)
	paddedKey := make([]byte, 32) // key = 0
	callData := append(selector, paddedSender...)
	callData = append(callData, paddedKey...)

	type ethCallParam struct {
		To   string `json:"to"`
		Data string `json:"data"`
	}
	params := []interface{}{
		ethCallParam{To: entryPoint, Data: "0x" + hex.EncodeToString(callData)},
		"latest",
	}

	// 直接用 paymaster.go 里的 sendRPC，但 userop.go 在同包，可直接调用
	result, err := sendRPC(ctx, rpcURL, "eth_call", params)
	if err != nil {
		return nil, fmt.Errorf("EntryPoint.getNonce eth_call: %w", err)
	}
	var hexStr string
	if err := json.Unmarshal(result, &hexStr); err != nil {
		return nil, fmt.Errorf("parse getNonce result: %w", err)
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if hexStr == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf("invalid nonce hex: %s", hexStr)
	}
	return n, nil
}

// ── SmartAccount 地址计算 ─────────────────────────────────────────────────────

// GetSmartAccountAddress 根据 EOA owner 和 salt 离线计算 SimpleAccount 的 CREATE2 地址。
// 不需要链上调用，纯本地计算。
// factory: SimpleAccountFactoryV06，salt: 通常为 0。
func GetSmartAccountAddress(owner common.Address, salt *big.Int, factory common.Address) common.Address {
	// SimpleAccountFactory.createAccount(owner, salt) 的 initCode:
	// keccak256(0xff ++ factory ++ salt ++ keccak256(creationCode ++ abi.encode(owner, salt)))
	// 简化实现：调用工厂的 getAddress(owner, salt)（只读，无需私钥）
	// 此处直接返回离线计算结果，实际部署前可通过 eth_call 验证
	saltBytes := common.LeftPadBytes(salt.Bytes(), 32)
	inner := crypto.Keccak256(append(owner.Bytes(), saltBytes...))
	packed := append([]byte{0xff}, factory.Bytes()...)
	packed = append(packed, saltBytes...)
	packed = append(packed, inner...)
	return common.BytesToAddress(crypto.Keccak256(packed)[12:])
}

// CreateSmartWallet 为一个 EOA 私钥生成对应的 SmartWalletInfo（离线，不上链）。
// 若需要在链上部署 SmartAccount，首次发 UserOperation 时 initCode 不为空，自动完成部署。
func CreateSmartWallet(privKeyHex string, chainID int64, salt *big.Int) (*SmartWalletInfo, error) {
	privKeyHex = strings.TrimPrefix(privKeyHex, "0x")
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("私钥解码失败: %w", err)
	}
	key, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("私钥解析失败: %w", err)
	}

	eoaAddr := crypto.PubkeyToAddress(key.PublicKey)
	factory := common.HexToAddress(SimpleAccountFactoryV06)
	smartAddr := GetSmartAccountAddress(eoaAddr, salt, factory)

	log.Printf("[4337] SmartWallet 信息计算完成")

	return &SmartWalletInfo{
		EOAAddress:          eoaAddr,
		SmartAccountAddress: smartAddr,
		IsDeployed:          false, // 需要链上查询，此处默认未部署
		ChainID:             chainID,
	}, nil
}

// ── UserOperation 构造 ────────────────────────────────────────────────────────

// DefaultGasLimits 构造 UserOperation 时的初始 gas 估算值（可被 Paymaster 覆盖）。
var DefaultGasLimits = struct {
	CallGas         *big.Int
	VerificationGas *big.Int
	PreVerification *big.Int
}{
	CallGas:         big.NewInt(300_000),
	VerificationGas: big.NewInt(150_000),
	PreVerification: big.NewInt(50_000),
}

// BuildUserOp 构造一笔 UserOperation，callData 是要在 SmartAccount 上执行的操作。
//
// callData 应为 SmartAccount.execute(target, value, innerCallData) 的 ABI 编码，
// 使用 BuildExecuteCallData 生成。
func BuildUserOp(
	smartAccount common.Address,
	nonce *big.Int,
	callData []byte,
	maxFeePerGas,
	maxPriorityFeePerGas *big.Int,
	initCode []byte, // 非首次部署传 nil
) *UserOperation {
	return &UserOperation{
		Sender:               smartAccount,
		Nonce:                nonce,
		InitCode:             initCode,
		CallData:             callData,
		CallGasLimit:         DefaultGasLimits.CallGas,
		VerificationGasLimit: DefaultGasLimits.VerificationGas,
		PreVerificationGas:   DefaultGasLimits.PreVerification,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		PaymasterAndData:     []byte{},
		Signature:            []byte{},
	}
}

// BuildExecuteCallData 构造 SimpleAccount.execute(target, value, data) 的 ABI calldata。
func BuildExecuteCallData(target common.Address, value *big.Int, innerData []byte) ([]byte, error) {
	// SimpleAccount.execute 函数签名: execute(address dest, uint256 value, bytes calldata func)
	const executeABI = `[{"inputs":[{"name":"dest","type":"address"},{"name":"value","type":"uint256"},{"name":"func","type":"bytes"}],"name":"execute","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsed, err := abi.JSON(strings.NewReader(executeABI))
	if err != nil {
		return nil, err
	}
	return parsed.Pack("execute", target, value, innerData)
}

// ── UserOperation 签名 ────────────────────────────────────────────────────────

// SignUserOp 计算 UserOperation 的 hash 并用 EOA 私钥签名，填充 Signature 字段。
// userOpHash = keccak256(abi.encode(keccak256(userOp), entryPoint, chainID))
func SignUserOp(op *UserOperation, entryPoint common.Address, chainID *big.Int, privKey *ecdsa.PrivateKey) error {
	hash := UserOpHash(op, entryPoint, chainID)
	// EIP-191 个人签名前缀
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n32")
	prefixedHash := crypto.Keccak256(append([]byte(msg), hash[:]...))
	sig, err := crypto.Sign(prefixedHash, privKey)
	if err != nil {
		return fmt.Errorf("sign userOp: %w", err)
	}
	sig[64] += 27 // v 值调整为 27/28
	op.Signature = sig
	return nil
}

// UserOpHash 计算 UserOperation 的标准 hash。
// v0.6: keccak256(abi.encode(keccak256(pack), entryPoint, chainID))
// v0.8: EIP-712 typed data hash（hash_initCode, hash_callData, accountGasLimits packed, gasFees packed）
func UserOpHash(op *UserOperation, entryPoint common.Address, chainID *big.Int) common.Hash {
	ep := entryPoint.Hex()

	if strings.EqualFold(ep, EntryPointV09) {
		// v0.9 EIP-712 格式
		return userOpHashV08(op, entryPoint, chainID)
	}
	// v0.6 原始格式
	return userOpHashV06(op, entryPoint, chainID)
}

// userOpHashV06: keccak256(abi.encode(keccak256(pack(op)), entryPoint, chainID))
func userOpHashV06(op *UserOperation, entryPoint common.Address, chainID *big.Int) common.Hash {
	inner := crypto.Keccak256(packUserOpV06(op))
	outer, _ := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("address")},
		{Type: mustType("uint256")},
	}.Pack([32]byte(inner), entryPoint, chainID)
	return common.BytesToHash(crypto.Keccak256(outer))
}

// userOpHashV08: EIP-712 keccak256("\x19\x01" || domainSeparator || structHash)
func userOpHashV08(op *UserOperation, entryPoint common.Address, chainID *big.Int) common.Hash {
	// 1. domain separator
	domainSeparator := eip712DomainSeparator(entryPoint, chainID)

	// 2. struct hash = keccak256(abi.encode(
	//      PACKED_USEROP_TYPEHASH,
	//      sender, nonce,
	//      keccak256(initCode), keccak256(callData),
	//      accountGasLimits, preVerificationGas, gasFees,
	//      paymasterDataKeccak(paymasterAndData)))
	structHash := userOpStructHashV08(op)

	// 3. EIP-712 digest = keccak256(0x1901 || domainSeparator || structHash)
	digest := crypto.Keccak256(append(append([]byte{0x19, 0x01}, domainSeparator...), structHash...))
	return common.BytesToHash(digest)
}

// eip712DomainSeparator: keccak256(abi.encode(EIP712_DOMAIN_TYPEHASH, name_hash, version_hash, chainID, entryPoint))
func eip712DomainSeparator(entryPoint common.Address, chainID *big.Int) []byte {
	// keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)")
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256([]byte("ERC4337"))
	versionHash := crypto.Keccak256([]byte("1"))

	args, _ := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
		{Type: mustType("uint256")},
		{Type: mustType("address")},
	}.Pack([32]byte(domainTypeHash), [32]byte(nameHash), [32]byte(versionHash), chainID, entryPoint)

	return crypto.Keccak256(args)
}

// PACKED_USEROP_TYPEHASH = keccak256("PackedUserOperation(address sender,uint256 nonce,bytes initCode,bytes callData,bytes32 accountGasLimits,uint256 preVerificationGas,bytes32 gasFees,bytes paymasterAndData)")
var packedUserOpTypeHash = common.HexToHash("0x29a0bca4af4be3421398da00295e58e6d7de38cb492214754cb6a47507dd6f8e")

// userOpStructHashV08: keccak256(abi.encode(PACKED_USEROP_TYPEHASH, sender, nonce, hash_initCode, hash_callData, accountGasLimits, preVerificationGas, gasFees, hash_paymasterAndData))
func userOpStructHashV08(op *UserOperation) []byte {
	accountGasLimits := new(big.Int).Lsh(op.CallGasLimit, 128)
	accountGasLimits = accountGasLimits.Or(accountGasLimits, op.VerificationGasLimit)

	gasFees := new(big.Int).Lsh(op.MaxPriorityFeePerGas, 128)
	gasFees = gasFees.Or(gasFees, op.MaxFeePerGas)

	// paymasterDataKeccak：只 hash paymasterAndData（不包含签名尾缀）
	hashPaymasterAndData := crypto.Keccak256(op.PaymasterAndData)

	// EIP-7702: initCode hash override
	var hashInitCode [32]byte
	if op.Auth7702 != nil && len(op.InitCode) > 2 && op.InitCode[0] == 0x77 && op.InitCode[1] == 0x02 {
		delegate := op.Auth7702.Address
		if len(op.InitCode) <= 20 {
			copy(hashInitCode[:], crypto.Keccak256(delegate.Bytes()))
		} else {
			copy(hashInitCode[:], crypto.Keccak256(append(delegate.Bytes(), op.InitCode[20:]...)))
		}
	} else {
		copy(hashInitCode[:], crypto.Keccak256(op.InitCode))
	}

	args, _ := abi.Arguments{
		{Type: mustType("bytes32")}, // PACKED_USEROP_TYPEHASH
		{Type: mustType("address")}, // sender
		{Type: mustType("uint256")}, // nonce
		{Type: mustType("bytes32")}, // hash_initCode
		{Type: mustType("bytes32")}, // hash_callData
		{Type: mustType("bytes32")}, // accountGasLimits (packed)
		{Type: mustType("uint256")}, // preVerificationGas
		{Type: mustType("bytes32")}, // gasFees (packed)
		{Type: mustType("bytes32")}, // hash_paymasterAndData
	}.Pack(
		packedUserOpTypeHash,
		op.Sender,
		op.Nonce,
		[32]byte(hashInitCode),
		[32]byte(crypto.Keccak256(op.CallData)),
		[32]byte(common.LeftPadBytes(accountGasLimits.Bytes(), 32)),
		op.PreVerificationGas,
		[32]byte(common.LeftPadBytes(gasFees.Bytes(), 32)),
		[32]byte(hashPaymasterAndData),
	)

	return crypto.Keccak256(args)
}

// packUserOpV06 — EIP-4337 v0.6: sender, nonce, hash(initCode), hash(callData),
//
//	callGasLimit, verificationGasLimit, preVerificationGas,
//	maxFeePerGas, maxPriorityFeePerGas, hash(paymasterAndData)
func packUserOpV06(op *UserOperation) []byte {
	args, _ := abi.Arguments{
		{Type: mustType("address")},
		{Type: mustType("uint256")},
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
		{Type: mustType("uint256")},
		{Type: mustType("uint256")},
		{Type: mustType("uint256")},
		{Type: mustType("uint256")},
		{Type: mustType("uint256")},
		{Type: mustType("bytes32")},
	}.Pack(
		op.Sender,
		op.Nonce,
		[32]byte(crypto.Keccak256(op.InitCode)),
		[32]byte(crypto.Keccak256(op.CallData)),
		op.CallGasLimit,
		op.VerificationGasLimit,
		op.PreVerificationGas,
		op.MaxFeePerGas,
		op.MaxPriorityFeePerGas,
		[32]byte(crypto.Keccak256(op.PaymasterAndData)),
	)
	return args
}

// keccak256 快捷函数
func keccak256(data []byte) []byte {
	h := crypto.Keccak256(data)
	return h[:]
}

func mustType(t string) abi.Type {
	typ, _ := abi.NewType(t, "", nil)
	return typ
}

// ── 总入口：TrySponsoredOp ────────────────────────────────────────────────────

// TrySponsoredOp 使用 EIP-7702 + EIP-4337 ERC-20 Paymaster 实现无 BNB 交易。
//
// 流程：
//  1. EOA 已在链下签署 EIP-7702 Authorization（委托给 Simple7702Account）
//  2. 以 EOA 作为 UserOperation 的 sender（EntryPoint v0.8 直接认 7702 委托 EOA）
//  3. callData = 直接调用目标合约的 calldata（无需 SmartAccount.execute 包装）
//  4. pm_sponsorUserOperation（ERC-20 模式）→ Paymaster 出 BNB，事后从 USDT 扣
//  5. EOA 对 UserOp hash 签名，eth_sendUserOperation 广播
//
// 参数：
//   - eoaAddress:  用户 EOA 地址（作为 UserOperation sender）
//   - nonce:       EntryPoint.getNonce(eoaAddress, 0) 的返回值
//   - target:      要调用的合约（如 PancakeSwap Router）
//   - value:       附带的原生币数量（USDT→Token swap 时为 0）
//   - callData:    直接调用 target 的 calldata（不需要 execute 包装）
//   - auth7702:    EIP-7702 授权元组（由 gasless7702.BuildAuthorization 生成）
//   - privKeyBytes: EOA 私钥（用于签名 UserOperation）
func TrySponsoredOp(
	ctx context.Context,
	registry *PaymasterRegistry,
	chainID int64,
	eoaAddress common.Address,
	nonce *big.Int,
	target common.Address,
	value *big.Int,
	callData []byte,
	maxFeePerGas, maxPriorityFeePerGas *big.Int,
	auth7702 *Eip7702Auth,
	privKeyBytes []byte,
) (common.Hash, error) {
	paymaster, ok := registry.Get(chainID)
	if !ok {
		return common.Hash{}, fmt.Errorf("[4337] chainID=%d 无 Paymaster 注册", chainID)
	}

	// 1. 构造 execute calldata：7702 模式下 EOA 本身就是合约，直接调用 target
	//    不需要 SmartAccount.execute 包装，callData 直接放原始 swap/approve calldata
	executeCallData, err := BuildExecuteCallData(target, value, callData)
	if err != nil {
		return common.Hash{}, fmt.Errorf("build execute callData: %w", err)
	}

	// 2. 构造 UserOperation（sender = EOA，initCode = 7702 标记）
	op := BuildUserOp(eoaAddress, nonce, executeCallData, maxFeePerGas, maxPriorityFeePerGas, nil)
	op.Auth7702 = auth7702

	// EIP-7702: initCode 需要设为 INITCODE_EIP7702_MARKER || abi.encode(delegate)
	// EntryPoint v0.8 据此知道 sender 将被 7702 委托给 delegate
	if auth7702 != nil {
		op.InitCode = build7702InitCode(auth7702.Address)
	}

	// 3. 向 Paymaster 申请 ERC-20 代付（填充 paymasterAndData + gas 估算）
	//    使用 EntryPoint v0.9 支持 7702
	op, err = paymaster.SponsorUserOp(ctx, op, EntryPointV09)
	if err != nil {
		return common.Hash{}, fmt.Errorf("sponsor userOp: %w", err)
	}

	// 4. EOA 签名 UserOperation
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return common.Hash{}, fmt.Errorf("解析私钥: %w", err)
	}
	entryPoint := common.HexToAddress(EntryPointV09)
	chainIDBig := big.NewInt(chainID)
	if err := SignUserOp(op, entryPoint, chainIDBig, privKey); err != nil {
		return common.Hash{}, fmt.Errorf("sign userOp: %w", err)
	}

	// 5. 发送给 Bundler
	hash, err := paymaster.SendUserOp(ctx, op, EntryPointV09)
	if err != nil {
		return common.Hash{}, fmt.Errorf("send userOp: %w", err)
	}

	log.Printf("[7702+4337] UserOperation 已广播")
	return hash, nil
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// INITCODE_EIP7702_MARKER EntryPoint v0.8 识别的 7702 标记前缀（2 字节: 0x77 0x02）
const initCode7702Marker = "7702"

// build7702InitCode 构造 7702 initCode = marker(2字节) || abi.encode(delegateAddress 填 32 字节)
func build7702InitCode(delegate common.Address) []byte {
	marker := common.Hex2Bytes(initCode7702Marker) // 2 bytes
	enc, _ := abi.Arguments{{Type: mustType("address")}}.Pack(delegate)
	return append(marker, enc...) // 2 + 32 = 34 bytes
}

func hexDecode(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	b, _ := hex.DecodeString(s)
	return b
}

func parseBigHex(s string) *big.Int {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil
	}
	return n
}

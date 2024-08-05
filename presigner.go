package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum-optimism/presigner/pkg/shell"
	"github.com/ethereum/go-ethereum/common"
)

type TxSignature struct {
	Signer    string `json:"signer"`
	Signature string `json:"signature"`
}

type TxState struct {
	ChainId    string `json:"chain_id"`
	RpcUrl     string `json:"rpc_url"`
	CreatedAt  string `json:"created_at"`
	SafeAddr   string `json:"safe_addr"`
	SafeNonce  string `json:"safe_nonce"`
	TargetAddr string `json:"target_addr"`
	ScriptName string `json:"script_name"`

	// populated by sign
	Data       string `json:"data"`
	Signatures []TxSignature `json:"signatures,omitempty"`

	// populated by simulate
	Calldata string `json:"calldata,omitempty"`
}

func main() {
	// global flags
	var jsonFile string
	var workdir string
	var scriptName string

	flag.StringVar(&jsonFile, "json-file", "", "JSON file")
	flag.StringVar(&workdir, "workdir", ".", "Directory in which to run the subprocess")
	flag.StringVar(&scriptName, "script-name", "CallPause", "Script name")

	// create flags
	var chainId string
	var rpcUrl string
	var safeAddr string
	var safeNonce string
	var targetAddr string

	flag.StringVar(&chainId, "chain", "1", "Chain ID")
	flag.StringVar(&rpcUrl, "rpc-url", "", "RPC URL (default to \"https://eth.llamarpc.com)\"")
	flag.StringVar(&safeAddr, "safe-addr", "", "Safe address")
	flag.StringVar(&safeNonce, "safe-nonce", "", "Safe nonce")
	flag.StringVar(&targetAddr, "target-addr", "", "Target address")

	// sign flags
	var privateKey string
	var ledger bool
	var mnemonic string
	var hdPath string
	var senderAddr string
	flag.StringVar(&privateKey, "private-key", "", "Private key to use for signing or executing")
	flag.BoolVar(&ledger, "ledger", false, "Use ledger device for signing or executing")
	flag.StringVar(&mnemonic, "mnemonic", "", "Mnemonic to use for signing or executing")
	flag.StringVar(&hdPath, "hd-paths", "m/44'/60'/0'/0/0", "Hierarchical deterministic derivation path for mnemonic or ledger, for signing or executing")
	flag.StringVar(&senderAddr, "sender", "", "Address of the --sender to pass to forge")


	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		log.Println("no command specified, use one of: create, nonce, threshold, owners, sign, merge, verify, simulate, execute")
		flag.PrintDefaults()
		os.Exit(1)
	}
	cmd := args[0]

	if cmd == "nonce" {
		if safeAddr == "" {
			log.Println("missing one of the required nonce parameter: safe-addr")
			flag.PrintDefaults()
			os.Exit(1)
		}

		if rpcUrl == "" {
			rpcUrl = "https://eth.llamarpc.com"
		}
		if chainId == "" {
			chainId = "1"
		}

		env := []string{
			"SAFE_ADDR=" + safeAddr,
		}
		outBuffer, _, err := shell.Run(workdir, "cast", env, "", true,
			"call",
			safeAddr,
			"nonce()",
			"--rpc-url", rpcUrl)
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}

		outBuffer, _, err = shell.Run(workdir, "cast", env, string(outBuffer), true,
			"--to-dec")
		fmt.Println(strings.TrimSpace(string(outBuffer)))

	} else if cmd == "threshold" {
		if safeAddr == "" {
			log.Println("missing one of the required nonce parameter: safe-addr")
			flag.PrintDefaults()
			os.Exit(1)
		}

		if rpcUrl == "" {
			rpcUrl = "https://eth.llamarpc.com"
		}
		if chainId == "" {
			chainId = "1"
		}

		env := []string{
			"SAFE_ADDR=" + safeAddr,
		}
		outBuffer, _, err := shell.Run(workdir, "cast", env, "", true,
			"call",
			safeAddr,
			"getThreshold()",
			"--rpc-url", rpcUrl)
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}

		outBuffer, _, err = shell.Run(workdir, "cast", env, string(outBuffer), true,
			"--to-dec")
		fmt.Println(strings.TrimSpace(string(outBuffer)))
	} else if cmd == "owners" {

		if safeAddr == "" {
			log.Println("missing one of the required nonce parameter: safe-addr")
			flag.PrintDefaults()
			os.Exit(1)
		}

		if rpcUrl == "" {
			rpcUrl = "https://eth.llamarpc.com"
		}
		if chainId == "" {
			chainId = "1"
		}

		env := []string{
			"SAFE_ADDR=" + safeAddr,
		}
		outBuffer, _, err := shell.Run(workdir, "cast", env, "", true,
			"call",
			safeAddr,
			"getOwners()",
			"--rpc-url", rpcUrl)
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}

		out := strings.TrimSpace(string(outBuffer))

		if !strings.HasPrefix(out, "0x") {
			log.Println("error running forge: result has invalid format")
			os.Exit(1)
		}

		hex := out[2:]
		if len(hex)%64 != 0 {
			log.Println("error running forge: result has invalid format")
			os.Exit(1)
		}

		// skip first two 64-byte chunks
		for i := 2 * 64; i < len(hex); i += 64 {
			addr := common.HexToAddress(hex[i : i+64])
			fmt.Println(strings.ToLower(addr.String()))
		}

	} else if cmd == "create" {
		if safeAddr == "" || targetAddr == "" {
			log.Println("missing one of the required create parameter: safe-addr, target-addr")
			flag.PrintDefaults()
			os.Exit(1)
		}

		if rpcUrl == "" {
			rpcUrl = "https://eth.llamarpc.com"
		}
		if chainId == "" {
			chainId = "1"
		}

		env := []string{
			"SAFE_ADDR=" + safeAddr,
			"SAFE_NONCE=" + safeNonce,
			"TARGET_ADDR=" + targetAddr,
		}

		outBuffer, _, err := shell.Run(workdir, "forge", env, "", false,
			"script",
			scriptName,
			"--sig", "sign()",
			"--rpc-url", rpcUrl,
			"--chain-id", chainId,
			"--via-ir")
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}
		if safeNonce == "" {
			safeNonce, err = extractNonce(outBuffer)
			if err != nil {
				log.Printf("error extracting nonce: %v\n", err)
				os.Exit(1)
			}
		}
		tx := &TxState{
			ChainId:    chainId,
			RpcUrl:     rpcUrl,
			CreatedAt:  time.Now().Format(time.RFC3339),
			SafeAddr:   safeAddr,
			SafeNonce:  safeNonce,
			TargetAddr: targetAddr,
			ScriptName: scriptName,
			Signatures: nil,
		}

		if jsonFile == "" {
			jsonFile = fmt.Sprintf("tx/draft-%s.json", safeNonce)
		}
		writeTxState(jsonFile, tx)
	} else if cmd == "sign" {
		options := 0
		if privateKey != "" {
			options++
		}
		if ledger {
			options++
		}
		if mnemonic != "" {
			options++
		}
		if options != 1 {
			log.Printf("one (and only one) of --private-key, --ledger, --mnemonic must be set")
			os.Exit(1)
		}

		tx := readTxState(jsonFile)

		var signingFlags []string
		if ledger {
			signingFlags = append(signingFlags, "--ledger")
			signingFlags = append(signingFlags, "--hd-paths", hdPath)
		}
		if mnemonic != "" {
			signingFlags = append(signingFlags, "--mnemonic", mnemonic)
		}
		if privateKey != "" {
			signingFlags = append(signingFlags, "--private-key", privateKey)
		}
		signingFlags = append(signingFlags, "--workdir", workdir)

		signingFlagsAddress := append(signingFlags, "--address")

		// read wallet address from ledger
		outBuffer, _, err := shell.Run(workdir, "eip712sign", []string{}, "", false, signingFlagsAddress...)
		if err != nil {
			log.Printf("error running eip712sign: %v\n", err)
			os.Exit(1)
		}
		signer := senderAddr
		if signer == "" {
			var err error
			signer, err = extractSigner(outBuffer)
			if err != nil {
				log.Printf("error running eip712sign: %v\n", err)
				os.Exit(1)
			}
		}

		log.Println("running simulation")

		useRpcUrl := tx.RpcUrl
		if rpcUrl != "" {
			useRpcUrl = rpcUrl
		}

		env := []string{
			"SAFE_ADDR=" + tx.SafeAddr,
			"SAFE_NONCE=" + tx.SafeNonce,
			"TARGET_ADDR=" + tx.TargetAddr,
		}

		outBuffer, _, err = shell.Run(workdir, "forge", env, "", false,
			"script",
			tx.ScriptName,
			"--sig", "sign()",
			"--rpc-url", useRpcUrl,
			"--chain-id", tx.ChainId,
			"--sender", signer,
			"--via-ir")

		tx.Data = extractData(outBuffer)

		// sign the payload
		outBuffer, _, err = shell.Run(workdir, "eip712sign", []string{}, tx.Data+"\n", false, signingFlags...)
		if err != nil {
			log.Printf("error running eip712sign: %v\n", err)
			os.Exit(1)
		}

		_, sig, err := extractSignatures(outBuffer)
		if err != nil {
			log.Printf("error extracting signatures: %v\n", err)
			os.Exit(1)
		}

		var found bool
		for _, s := range tx.Signatures {
			if s.Signer == signer {
				log.Printf("signature for %s already exists, overwriting\n", signer)
				s.Signature = sig
				found = true
				break
			}
		}
		if !found {
			tx.Signatures = append(tx.Signatures, TxSignature{
				Signer:    signer,
				Signature: sig,
			})
			log.Printf("added signature for %s\n", signer)
		}
		if jsonFile == "" || (strings.HasPrefix(path.Base(jsonFile), "draft-") && strings.HasSuffix(jsonFile, ".json")) {
			newName, err := extractFilename(jsonFile, "draft", signer)
			if err != nil {
				log.Printf("error generating filename: %v\n", err)
				os.Exit(1)
			}
			jsonFile = newName
		}
		writeTxState(jsonFile, tx)
	} else if cmd == "verify" {
		tx := readTxState(jsonFile)
		if len(tx.Signatures) == 0 {
			log.Printf("no signatures found\n")
			os.Exit(1)
		}
		signatures := ""
		for _, s := range tx.Signatures {
			signatures = signatures + s.Signature
		}
		env := []string{
			"SAFE_ADDR=" + tx.SafeAddr,
			"SAFE_NONCE=" + tx.SafeNonce,
			"TARGET_ADDR=" + tx.TargetAddr,
		}
		useRpcUrl := tx.RpcUrl
		if rpcUrl != "" {
			useRpcUrl = rpcUrl
		}
		outBuffer, _, err := shell.Run(workdir, "forge", env, "", false,
			"script",
			tx.ScriptName,
			"--sig", "verify(bytes)", signatures,
			"--rpc-url", useRpcUrl,
			"--chain", tx.ChainId,
			"--via-ir")
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}

		if strings.Contains(string(outBuffer), "Script ran successfully.") {
			log.Printf("signatures are valid and tx is ready to be executed\n")
		} else {
			os.Exit(255) // succeeded but signatures are invalid
		}
	} else if cmd == "merge" {
		tx := readTxState(jsonFile)

		signatures := make(map[string]string, len(tx.Signatures))
		for _, s := range tx.Signatures {
			signatures[s.Signer] = s.Signature
		}

		for _, otherFile := range args[1:] {
			otherTx := readTxState(otherFile)
			if otherTx.SafeAddr != tx.SafeAddr {
				log.Printf("safe addr mismatch for file: %s\n", otherFile)
				log.Printf("   %s != %s\n", otherTx.SafeAddr, tx.SafeAddr)
				os.Exit(1)
			}
			if otherTx.TargetAddr != tx.TargetAddr {
				log.Printf("target addr mismatch for file: %s\n", otherFile)
				log.Printf("   %s != %s\n", otherTx.TargetAddr, tx.TargetAddr)
				os.Exit(1)
			}
			if otherTx.Data != tx.Data {
				if tx.Data == "" {
					tx.Data = otherTx.Data
				} else {
					log.Printf("data mismatch for file: %s\n", otherFile)
					log.Printf("   %s != %s\n", otherTx.Data, tx.Data)
					os.Exit(1)
				}
			}
			if otherTx.SafeNonce != tx.SafeNonce {
				log.Printf("nonce mismatch for file: %s\n", otherFile)
				log.Printf("   %s != %s\n", otherTx.SafeNonce, tx.SafeNonce)
				os.Exit(1)
			}

			for _, s := range otherTx.Signatures {
				signatures[s.Signer] = s.Signature
			}
		}

		newSigs := make([]TxSignature, 0, len(signatures))
		for signer, sig := range signatures {
			newSigs = append(newSigs, TxSignature{signer, sig})
		}
		tx.Signatures = newSigs

		writeTxState(jsonFile, tx)
	} else if cmd == "execute" || cmd == "simulate" {
		tx := readTxState(jsonFile)

		if cmd == "execute" {
			if len(tx.Signatures) == 0 {
				log.Printf("no signatures found\n")
				os.Exit(1)
			}

			options := 0
			if privateKey != "" {
				options++
			}
			if ledger {
				options++
			}
			if options != 1 {
				log.Printf("one (and only one) of --private-key, --ledger must be set for execution")
				os.Exit(1)
			}
		}

		signatures := ""
		for _, s := range tx.Signatures {
			signatures = signatures + s.Signature
		}
		env := []string{
			"SAFE_ADDR=" + tx.SafeAddr,
			"SAFE_NONCE=" + tx.SafeNonce,
			"TARGET_ADDR=" + tx.TargetAddr,
		}
		useRpcUrl := tx.RpcUrl
		if rpcUrl != "" {
			useRpcUrl = rpcUrl
		}
		var optFlags []string
		var signingFlags []string

		if ledger {
			signingFlags = append(signingFlags, "--ledger")
		}
		if privateKey != "" {
			signingFlags = append(signingFlags, "--private-key", privateKey)
		}
		if cmd == "execute" {
			optFlags = append(optFlags,
				"--broadcast",
				"--sig", "run(bytes)", signatures)
		} else if cmd == "simulate" {
			optFlags = append(optFlags,
				"--sig", "simulateSigned(bytes)", signatures)
		}
		optFlags = append(optFlags, signingFlags...)

		execFlags := []string{
			"script",
			tx.ScriptName,

			"--rpc-url", useRpcUrl,
			"--chain", tx.ChainId,
			"--via-ir"}
		execFlags = append(execFlags, optFlags...)

		outBuffer, _, err := shell.Run(workdir, "forge", env, "", false, execFlags...)
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}

		if strings.Contains(string(outBuffer), "Script ran successfully.") {
			log.Printf("simulation succeeded\n")
		} else {
			os.Exit(255) // simulation failed
		}

		if cmd == "simulate" {
			if jsonFile == "" || (strings.HasPrefix(path.Base(jsonFile), "draft-") && strings.HasSuffix(jsonFile, ".json")) {
				newName, err := extractFilename(jsonFile, "ready", "")
				if err != nil {
					log.Printf("error generating filename: %v\n", err)
					os.Exit(1)
				}
				jsonFile = newName
			}

			calldata, err := extractCalldata(outBuffer)
			if err != nil {
				log.Printf("error extracting calldata: %v\n", err)
				os.Exit(1)
			}
			tx.Calldata = calldata
			log.Printf("added calldata\n")
			writeTxState(jsonFile, tx)

			printExecuteInstructions(jsonFile, tx, useRpcUrl)

			onelinerName := strings.ReplaceAll(jsonFile, ".json", ".sh.b64")
			createOneLiner(onelinerName, tx)

			oneliner := fmt.Sprintf("/bin/bash <(base64 -d -i %s) --rpc-url %s", onelinerName, useRpcUrl)

			log.Printf(`

to run oneliner:
    %s

`, shell.Highlight(oneliner))
		}
	} else {
		log.Println("unknown command, use one of: create, nonce, threshold, owners, sign, merge, verify, simulate, execute")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func printExecuteInstructions(jsonFile string, tx *TxState, useRpcUrl string) {
	presignerCmd := fmt.Sprintf(`go run presigner.go \
    --json-file %s \
    --private-key $EXECUTORKEY \
    execute`, jsonFile)
	castCmd := fmt.Sprintf(
		`SAFE_ADDR=%s
CALLDATA=%s
EXECUTORKEY=********
cast send \
    --rpc-url %s \
    --chain %s \
    --private-key $EXECUTORKEY \
    $SAFE_ADDR \
    $CALLDATA`,
		tx.SafeAddr, tx.Calldata, useRpcUrl, tx.ChainId)

	log.Printf(`

transaction now can be sent to network with:

- - 8< - -

%s

- - or - -

%s

- - 8< - -
`,
		shell.Highlight(presignerCmd), shell.Highlight(castCmd))
}

func createOneLiner(onelinerName string, tx *TxState) {
	contents := fmt.Sprintf(`
echo -n "checking for rust... "
RUST_VERSION=$(rustc -V 2> /dev/null || echo none)
echo $RUST_VERSION
if [ "$x" = "none" ]; then
  echo "install rust with:"
  echo "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
  exit 1
fi

echo -n "checking for cast... "
CAST_VERSION=$(cast -V 2> /dev/null || echo none)
echo $CAST_VERSION
if [ "$x" = "none" ]; then
  echo "install cast with:"
  echo "curl -L https://foundry.paradigm.xyz | bash && foundryup"
  exit 1
fi


SAFE_ADDR=%s
CALLDATA=%s
CHAIN_ID=%s

CAST_CMD="cast send --chain $CHAIN_ID $SAFE_ADDR $CALLDATA $*"

echo calling: $CAST_CMD
echo "- - - press ENTER to continue or CTRL-C to abort - - -"
read

echo sending transaction...
$CAST_CMD
`, tx.SafeAddr, tx.Calldata, tx.ChainId)

	base64Encoded := make([]byte, base64.StdEncoding.EncodedLen(len(contents)))
	base64.StdEncoding.Encode(base64Encoded, []byte(contents))
	shell.WriteFile(onelinerName, base64Encoded)
}

func writeTxState(file string, tx *TxState) {
	jsonContents, err := json.Marshal(tx)
	if err != nil {
		log.Println("error marshalling tx state")
		os.Exit(1)
	}
	shell.WriteFile(file, jsonContents)
}

func readTxState(file string) *TxState {
	var tx TxState
	jsonContents, err := os.ReadFile(file)
	if err != nil {
		log.Printf("error reading tx state: %v\n", err)
		os.Exit(1)
	}
	err = json.Unmarshal(jsonContents, &tx)
	if err != nil {
		log.Printf("error unmarshalling tx state: %v\n", err)
		os.Exit(1)
	}
	return &tx
}

func extractFilename(filename, newState string, signer string) (string, error) {
	dir := path.Dir(filename)
	base := path.Base(filename)
	exp := regexp.MustCompile("(ready|draft)-(.*?)-?(\\d+).json")
	matches := exp.FindStringSubmatch(base)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid filename pattern")
	}
	if len(matches[2]) > 0 {
		newState = newState + "-"
	}
	if len(signer) > 0 {
		signer = ".signer-" + signer
	}
	newName := newState + matches[2] + "-" + matches[3] + signer + ".json"
	return path.Join(dir, newName), nil
}

func extractNonce(buffer []byte) (string, error) {
	exp := regexp.MustCompile(".*?\n  Safe current nonce: (.*?)\n")
	matches := exp.FindStringSubmatch(string(buffer))
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid output from forge")
	}
	if matches[1] == "" {
		return "", fmt.Errorf("invalid output from forge")
	}
	return matches[1], nil
}

func extractCalldata(buffer []byte) (string, error) {
	exp := regexp.MustCompile(".*&rawFunctionInput=(.*?)\n")
	matches := exp.FindStringSubmatch(string(buffer))
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid output from forge")
	}
	if matches[1] == "" {
		return "", fmt.Errorf("invalid output from forge")
	}
	return matches[1], nil
}

func extractSigner(buffer []byte) (string, error) {
	exp := regexp.MustCompile(".*?\n?Signer: (.*?)\n")
	matches := exp.FindStringSubmatch(string(buffer))
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid output from eip712sign")
	}
	if matches[1] == "" {
		return "", fmt.Errorf("invalid output from eip712sign")
	}
	return matches[1], nil
}

func extractSignatures(buffer []byte) (string, string, error) {
	exp := regexp.MustCompile(".*?\nData: (.*?)\nSigner: (.*?)\nSignature: (.*?)\n")
	matches := exp.FindStringSubmatch(string(buffer))
	if len(matches) != 4 {
		return "", "", fmt.Errorf("invalid output from eip712sign")
	}
	if matches[2] == "" || matches[3] == "" {
		return "", "", fmt.Errorf("invalid output from eip712sign")
	}
	return matches[2], matches[3], nil
}

func extractData(input []byte) string {
	prefix := "vvvvvvvv"
	suffix := "^^^^^^^^"
	if index := strings.Index(string(input), prefix); prefix != "" && index >= 0 {
		input = input[index+len(prefix):]
	}
	if index := strings.Index(string(input), suffix); suffix != "" && index >= 0 {
		input = input[:index]
	}
	return strings.TrimSpace(string(input))
}

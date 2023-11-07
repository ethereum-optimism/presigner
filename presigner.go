package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type TxSignatures struct {
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
	Data       string `json:"data"`

	// populated by sign
	Signatures []TxSignatures `json:"signatures,omitempty"`

	// populated by simulate
	Calldata string `json:"calldata,omitempty"`
}

func main() {
	// global flags
	var jsonFile string
	var workdir string
	var scriptName string

	flag.StringVar(&jsonFile, "json-file", "tx/presigner.json", "Json file")
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
	flag.StringVar(&privateKey, "private-key", "", "Private key to use for signing or executing")
	flag.BoolVar(&ledger, "ledger", false, "Use ledger device for signing or executing")
	flag.StringVar(&mnemonic, "mnemonic", "", "Mnemonic to use for signing or executing")
	flag.StringVar(&hdPath, "hd-paths", "m/44'/60'/0'/0/0", "Hierarchical deterministic derivation path for mnemonic or ledger, for signing or executing")

	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		log.Println("no command specified, use one of: create, nonce, sign, verify, simulate, execute")
		flag.PrintDefaults()
		os.Exit(1)
	}
	cmd := args[0]

	if workdir == "" || jsonFile == "" || scriptName == "" {
		log.Println("missing one of the required global parameter: workdir, json-file, script-name")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if cmd == "nonce" {
		if safeAddr == "" {
			log.Println("missing one of the required create parameter: safe-addr")
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

		_, _, err := run(workdir, "forge", env, "", "script",
			scriptName,
			"--sig", "nonce()",
			"--rpc-url", rpcUrl,
			"--chain-id", chainId,
			"--via-ir")

		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
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

		outBuffer, _, err := run(workdir, "forge", env, "", "script",
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
			Data:       extractData(outBuffer),
			Signatures: nil,
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
			log.Printf("one (and only one) of -private-key, -ledger, -mnemonic must be set")
			os.Exit(1)
		}

		tx := readTxState(jsonFile)

		var signingFlags []string
		if ledger {
			signingFlags = append(signingFlags, "-ledger")
		}
		if mnemonic != "" {
			signingFlags = append(signingFlags, "-mnemonic", mnemonic)
		}
		if privateKey != "" {
			signingFlags = append(signingFlags, "-private-key", privateKey)
		}
		signingFlags = append(signingFlags, "-hd-paths", hdPath)
		signingFlags = append(signingFlags, "-workdir", workdir)

		outBuffer, _, err := run(workdir, "eip712sign", []string{}, tx.Data+"\n", signingFlags...)

		if err != nil {
			log.Printf("error running eip712sign: %v\n", err)
			os.Exit(1)
		}

		signer, sig, err := extractSignatures(outBuffer)
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
			tx.Signatures = append(tx.Signatures, TxSignatures{
				Signer:    signer,
				Signature: sig,
			})
			log.Printf("added signature for %s\n", signer)
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
		outBuffer, _, err := run(workdir, "forge", env, "", "script",
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
	} else if cmd == "execute" || cmd == "simulate" {
		tx := readTxState(jsonFile)
		if len(tx.Signatures) == 0 {
			log.Printf("no signatures found\n")
			os.Exit(1)
		}

		if cmd == "execute" {
			options := 0
			if privateKey != "" {
				options++
			}
			if ledger {
				options++
			}
			if options != 1 {
				log.Printf("one (and only one) of -private-key, -ledger must be set for execution")
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
		execArgs := []string{"script",
			tx.ScriptName,
			"--sig", "run(bytes)", signatures,
			"--rpc-url", useRpcUrl,
			"--chain", tx.ChainId,
			"--via-ir"}

		if cmd == "execute" {
			execArgs = append(execArgs, "--broadcast")
			if ledger {
				execArgs = append(execArgs, "--ledger")
			}
			if privateKey != "" {
				execArgs = append(execArgs, "--private-key", privateKey)
			}
		}

		outBuffer, _, err := run(workdir, "forge", env, "", execArgs...)
		if err != nil {
			log.Printf("error running forge: %v\n", err)
			os.Exit(1)
		}
		calldata, err := extractCalldata(outBuffer)
		if err != nil {
			log.Printf("error extracting calldata: %v\n", err)
			os.Exit(1)
		}
		tx.Calldata = calldata
		log.Printf("added calldata\n")
		writeTxState(jsonFile, tx)

		presignerCmd := fmt.Sprintf(`go run presigner.go \
    -json-file %s \
    -private-key $EXECUTORKEY \
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
			tx.SafeAddr, calldata, useRpcUrl, tx.ChainId)

		log.Printf(`

transaction now can be sent to network with:

- - 8< - -

%s

- - or - -

%s

- - 8< - - 
`, highlight(presignerCmd), highlight(castCmd))
	} else {
		log.Println("unknown command, use one of: create, nonce, sign, verify, simulate, execute")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func writeTxState(file string, tx *TxState) {
	exists := existFile(file)
	if exists {
		log.Printf("file %s already exists, overwriting\n", file)
	}
	jsonContents, err := json.Marshal(tx)
	if err != nil {
		log.Println("error marshalling tx state")
		os.Exit(1)
	}
	os.WriteFile(file, jsonContents, 0600)
	log.Printf("saved: %s\n", file)
}

func existFile(file string) bool {
	if _, err := os.Stat("/path/to/whatever"); err == nil {
		return true
	}
	return false
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

func run(workdir, name string, env []string, in string, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	cmd.Env = env

	var outBuffer, errBuffer bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuffer)
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuffer)

	var stdinpipe io.WriteCloser
	if in != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		defer stdin.Close()
		stdinpipe = stdin
	}

	fmt.Println("running:", obfuscateCmdString(cmd.String()))
	err := cmd.Start()

	if in != "" {
		io.WriteString(stdinpipe, in)
		stdinpipe.Close()
	}

	cmd.Wait()

	return outBuffer.Bytes(), errBuffer.Bytes(), err
}

func obfuscateCmdString(s string) string {
	output := ""
	words := strings.Split(s, " ")
	lastWord := ""
	for _, w := range words {
		if strings.HasSuffix(lastWord, "-private-key") ||
			strings.HasSuffix(lastWord, "-mnemonic") ||
			strings.HasSuffix(lastWord, "-hd-paths") {
			w = "********"
		}
		if output != "" {
			output += " "
		}
		output += w
		lastWord = w
	}
	return output
}

func highlight(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 36, s)
}

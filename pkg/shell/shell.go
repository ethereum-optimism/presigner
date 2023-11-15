package shell

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Run(workdir, name string, env []string, in string, silent bool, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir

	if len(env) > 0 {
		cmd.Env = append(cmd.Env, env...)
	}

	var outBuffer, errBuffer bytes.Buffer
	if silent {
		cmd.Stdout = bufio.NewWriter(&outBuffer)
		cmd.Stderr = bufio.NewWriter(&errBuffer)
	} else {
		cmd.Stdout = io.MultiWriter(os.Stdout, &outBuffer)
		cmd.Stderr = io.MultiWriter(os.Stderr, &errBuffer)
	}

	var stdinpipe io.WriteCloser
	if in != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		defer stdin.Close()
		stdinpipe = stdin
	}

	if !silent {
		fmt.Println("running:", ObfuscateCmdString(cmd.String()))
	}
	err := cmd.Start()

	if in != "" {
		io.WriteString(stdinpipe, in)
		stdinpipe.Close()
	}

	cmd.Wait()

	return outBuffer.Bytes(), errBuffer.Bytes(), err
}

func ObfuscateCmdString(s string) string {
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

func Highlight(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 36, s)
}

func WriteFile(file string, s []byte) {
	exists := ExistFile(file)
	if exists {
		log.Printf("file %s already exists, overwriting\n", file)
	}

	os.WriteFile(file, s, 0600)
	log.Printf("saved: %s\n", file)
}

func ExistFile(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func Base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum-optimism/presigner/pkg/shell"
)

func main() {
	var workdir string
	var path string
	var account string
	var vault string

	flag.StringVar(&workdir, "workdir", ".", "Workdir")
	flag.StringVar(&path, "path", "tx/", "Path to files to be pushed or pulled")
	flag.StringVar(&account, "account", "oplabs.1password.com", "1Password account")
	flag.StringVar(&vault, "vault", "Pre-signed Pause", "1Password vault")

	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		log.Println("no command specified, use one of: list, pull, push")
		flag.PrintDefaults()
		os.Exit(1)
	}
	cmd := args[0]

	if cmd == "list" {
		outBuffer, _, err := shell.Run(workdir, "op", []string{}, "", true,
			"--format", "json",
			"--account", account,
			"--vault", vault,
			"item",
			"list")
		if err != nil {
			log.Printf("error running op: %v\n", err)
			os.Exit(1)
		}
		var j []map[string]interface{}
		json.Unmarshal(outBuffer, &j)
		for _, jitem := range j {
			fmt.Println(jitem["title"])
		}
	} else if cmd == "pull" {
		if len(args) != 2 {
			log.Println("use: pull <item>")
			flag.PrintDefaults()
			os.Exit(1)
		}
		item := args[1]
		outBuffer, _, err := shell.Run(workdir, "op", []string{}, "", true,
			"--account", account,
			"read",
			fmt.Sprintf("op://%s/%s/text", vault, item))
		if err != nil {
			log.Printf("error running op: %v\n", err)
			os.Exit(1)
		}
		decoded, err := base64.StdEncoding.DecodeString(string(outBuffer))
		if err != nil {
			log.Printf("error decoding base64: %v\n", err)
			os.Exit(1)
		}
		shell.WriteFile(fmt.Sprintf("%s/%s", path, item), decoded)
	} else if cmd == "push" {
		if len(args) != 2 {
			log.Println("use: push <item>")
			flag.PrintDefaults()
			os.Exit(1)
		}
		item := args[1]

		// check for existence
		_, errBuffer, err := shell.Run(workdir, "op", []string{}, "", true,
			"--account", account,
			"read",
			fmt.Sprintf("op://%s/%s/text", vault, item))
		if err != nil {
			log.Printf("error running op: %v\n", err)
			os.Exit(1)
		}
		if !strings.Contains(string(errBuffer), "isn't an item") {
			fmt.Println("item already exists, exiting")
			os.Exit(255)
		}

		contents, err := os.ReadFile(fmt.Sprintf("%s/%s", path, item))
		if err != nil {
			log.Printf("error reading file: %v\n", err)
			os.Exit(1)
		}
		b64 := base64.StdEncoding.EncodeToString(contents)
		_, _, err = shell.Run(workdir, "op", []string{}, "", true,
			"--account", account,
			"--vault", vault,
			"item",
			"create",
			"--title", item,
			"--category", "Login",
			fmt.Sprintf("text=%s", b64))
		if err != nil {
			log.Printf("error running op: %v\n", err)
			os.Exit(1)
		}
	} else {
		log.Println("unknown command, use one of: list, pull, push")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Songmu/prompter"
	"gopkg.in/ini.v1"
)

var (
	defaultProfile string
	mfaProfile     string
)

func init() {
	flag.StringVar(&defaultProfile, "default profile name", "default", "")
	flag.StringVar(&mfaProfile, "mfa profile name", "mfa", "")
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var credFile string
	if cf := os.Getenv("AWS_CONFIG_FILE"); cf != "" {
		credFile = cf
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		credFile = filepath.Join(home, ".aws", "credentials")
	}
	cred, err := ini.Load(credFile)
	if err != nil {
		return err
	}
	// create mfa section if not exists
	if _, err := cred.GetSection(mfaProfile); err != nil {
		if _, err := cred.NewSection(mfaProfile); err != nil {
			return err
		}
		if err := cred.SaveTo(credFile); err != nil {
			return err
		}
		if err := cred.Reload(); err != nil {
			return err
		}
	}
	expiration := cred.Section(mfaProfile).Key("expiration").MustTimeFormat(time.RFC3339)
	if time.Now().Before(expiration) {
		log.Println("OK: There is no need to update the session token.")
		return nil
	}

	serialNumber := cred.Section(defaultProfile).Key("aws_mfa_device").String()
	if serialNumber == "" {
		return errors.New("empty mfa device serial number in default profile")
	}
	deviceCode := prompter.Prompt("DEVICE CODE", "")
	if deviceCode == "" {
		return errors.New("empty device code")
	}

	var stderr bytes.Buffer
	cmds := []string{"aws", "sts", "get-session-token", "--serial-number", serialNumber, "--token-code", deviceCode, "--profile", defaultProfile}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("aws cli failed %s %s, : %w", cmds, strings.TrimSpace(stderr.String()), err)
	}
	var r struct {
		Credentials struct {
			AccessKeyID     string    `json:"AccessKeyId"`
			SecretAccessKey string    `json:"SecretAccessKey"`
			SessionToken    string    `json:"SessionToken"`
			Expiration      time.Time `json:"Expiration"`
		} `json:"Credentials"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}

	cred.Section(mfaProfile).Key("aws_access_key_id").SetValue(r.Credentials.AccessKeyID)
	cred.Section(mfaProfile).Key("aws_secret_access_key").SetValue(r.Credentials.SecretAccessKey)
	cred.Section(mfaProfile).Key("aws_session_token").SetValue(r.Credentials.SessionToken)
	cred.Section(mfaProfile).Key("expiration").SetValue(r.Credentials.Expiration.Format(time.RFC3339))
	if err := cred.SaveTo(credFile); err != nil {
		return err
	}
	log.Println("OK: Successfully update the session token.")
	return nil
}

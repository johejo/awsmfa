package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	credPath := filepath.Join(home, ".aws", "credentials")
	cred, err := ini.Load(credPath)
	if err != nil {
		return err
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
	b, err := exec.Command("aws", "sts", "get-session-token", "--serial-number", serialNumber, "--token-code", deviceCode).Output()
	if err != nil {
		return err
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
	if err := cred.SaveTo(credPath); err != nil {
		return err
	}
	log.Println("OK: Successfully update the session token.")
	return nil
}

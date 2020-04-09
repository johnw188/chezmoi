package chezmoi

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

// GPG interfaces with gpg.
type GPG struct {
	Recipient string
	Symmetric bool
}

// Decrypt decrypts ciphertext. filename is used as a hint for naming temporary
// files.
func (g *GPG) Decrypt(filename string, ciphertext []byte) ([]byte, error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-decrypt")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	outputFilename := path.Join(tempDir, path.Base(filename))
	inputFilename := outputFilename + ".gpg"
	if err := ioutil.WriteFile(inputFilename, ciphertext, 0o600); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"gpg",
		"--output", outputFilename,
		"--quiet",
		"--decrypt", inputFilename,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(outputFilename)
}

// Encrypt encrypts plaintext for ts's recipient. filename is used as a hint for
// naming temporary files.
func (g *GPG) Encrypt(filename string, plaintext []byte) ([]byte, error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-encrypt")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	inputFilename := path.Join(tempDir, path.Base(filename))
	if err := ioutil.WriteFile(inputFilename, plaintext, 0o600); err != nil {
		return nil, err
	}
	outputFilename := inputFilename + ".gpg"

	args := []string{
		"--armor",
		"--output", outputFilename,
		"--quiet",
	}
	if g.Symmetric {
		args = append(args, "--symmetric")
	} else {
		if g.Recipient != "" {
			args = append(args, "--recipient", g.Recipient)
		}
		args = append(args, "--encrypt")
	}
	args = append(args, filename)
	cmd := exec.Command("gpg", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(outputFilename)
}

package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/howeyc/gopass"
	"github.com/nmcclain/totp"
	"github.com/skratchdot/open-golang/open"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"sort"
	"strings"
	"time"
)

var version = "0.1"

var usage = `mytotp: TOTP client for the command line

Usage:
  mytotp [options]
  mytotp add <name> <secret>
  mytotp qr <filename>
  mytotp dump
  mytotp import <filename>
  mytotp -h --help
  mytotp --version

Options:
  -s, --secrets <file>        TOTP secrets file - KEEP PRIVATE [default: ~/.totp]. 
  -c, --continuous            Print codes continuously.
  -w, --web                   Run a simple web interface.
  -l, --listen <ip:port>      Local IP and port for web interface [default: localhost:8000].
  -h, --help                  Show this screen.
  --version                   Show version.

Optionally set the MYTOTP_PASSPHRASE environment variable to avoid the initial passphrase prompt.
`

type codesets struct {
	Codes []codeset
	Valid float64
}
type codeset struct {
	Name string
	Code string
	Id   int
}
type secret struct {
	id     int
	name   string
	secret string
	url    string
}

func fatal(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func main() {
	// handle cli arguments
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		fatal(err)
	}

	// substitute ~ for homedir
	secretsFileName := args["--secrets"].(string)
	if secretsFileName[:2] == "~/" {
		usr, err := user.Current()
		if err != nil {
			fatal(err)
		}
		secretsFileName = strings.Replace(secretsFileName, "~/", usr.HomeDir+"/", 1)
	}

	// prompt for encryption password if not in env var
	pass := []byte(os.Getenv("MYTOTP_PASSPHRASE"))
	if len(pass) < 1 {
		fmt.Print("mytotp passphrase: ")
		pass = []byte(gopass.GetPasswd())
	}
	salt := []byte("salt is so salty") // TODO: randomly generate this and store it @ the beginning of the totp creds file.
	// "The recommended parameters for interactive logins as of 2009 are N=16384, r=8, p=1"
	key, err := scrypt.Key(pass, salt, 2*16384, 8, 1, 32)
	if err != nil {
		fatal(err)
	}

	// enforce reasonable secrets file security
	fileInfo, err := os.Stat(secretsFileName)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			if args["add"].(bool) || args["qr"].(bool) {
				// create a new file
				err := saveSecrets(secretsFileName, key, make(map[string]secret))
				if err != nil {
					fatal(err)
				}
			} else {
				fatal(fmt.Errorf("Your mytotp secrets file was not found at: %s\n Try running 'mytotp add <name> <secret>' or 'mytotp qr <filename>' to create it,\n or use the -c flag to specify a different location.", secretsFileName))
			}
		} else { // some other error w/that file, besides not existing.
			fatal(err)
		}
	} else {
		if !strings.HasSuffix(fileInfo.Mode().String(), "------") {
			fatal(fmt.Errorf("Permissions on secrets file are too loose - try a 'chmod 600 %s'", secretsFileName))
		}
	}

	// read the secrets file
	cipherText, err := ioutil.ReadFile(secretsFileName)
	if err != nil {
		fatal(err)
	}
	// decrypt secrets file
	plainText, err := decrypt(key, cipherText)
	if err != nil {
		fatal(err)
	}
	// parse the secrets file
	secrets, err := parseSecrets(plainText)
	if err != nil {
		fatal(err)
	}

	// handle misc commands
	if args["dump"].(bool) {
		// slice of sorted map keys
		keys := []string{}
		for k := range secrets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// print out the URLs
		for _, k := range keys {
			s := secrets[k]
			fmt.Println(s.url)
		}
		return
	} else if args["import"].(bool) {
		// import from urls from file
		importFile, err := ioutil.ReadFile(args["<filename>"].(string))
		if err != nil {
			fatal(err)
		}
		newSecrets := 0
		updatedSecrets := 0
		for _, c := range strings.Split(strings.TrimRight(string(importFile), "\r\n"), "\n") {
			s, err := parseUrl(c)
			if err != nil {
				log.Printf("Skipping unrecognized URL %s: %s", c, err.Error())
				continue
			}
			if _, exists := secrets[s.name]; exists {
				log.Printf("Updating existing secret: %s", s.name)
				updatedSecrets++
			} else {
				log.Printf("Importing new secret: %s", s.name)
				newSecrets++
			}
			secrets[s.name] = s
		}
		err = saveSecrets(secretsFileName, key, secrets)
		if err != nil {
			fatal(err)
		}
		log.Printf("Imported %d new secrets, updated %d secrets", newSecrets, updatedSecrets)
		return
	}

	// add a new secret
	if args["add"].(bool) || args["qr"].(bool) {
		var s secret
		if args["add"].(bool) {
			s.name = args["<name>"].(string)
			s.url = fmt.Sprintf("otpauth://totp/%s?secret=%s\n", s.name, args["<secret>"].(string))
		} else if args["qr"].(bool) {
			// add a new secret from a QR code image, then exit
			s, err = parseQr(args["<filename>"].(string))
			if err != nil {
				fatal(err)
			}
		}
		if _, exists := secrets[s.name]; exists {
			fatal(fmt.Errorf("Secret with name %s already exists!", s.name))
		}
		_, err := base32.StdEncoding.DecodeString(s.secret)
		if err != nil {
			fatal(err)
		}
		secrets[s.name] = s
		err = saveSecrets(secretsFileName, key, secrets)
		if err != nil {
			fatal(err)
		}
		log.Printf("Secret '%s' successfully added!", s.name)
	}

	// start a web server
	if args["--web"].(bool) {
		doWeb(args["--listen"].(string), secrets)
		open.Start("http://" + args["--listen"].(string))
	}

	// calculate and output the user codes
	for {
		// slice of sorted map keys
		keys := []string{}
		for k := range secrets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// print out the codes
		for _, name := range keys {
			s := secrets[name]
			code, err := totp.GetUserCode([]byte(s.secret), nil)
			if err != nil {
				fatal(err)
			}
			fmt.Printf("%s: %s\n", code, name)
		}
		secs := 30 - math.Mod(float64(time.Now().Unix()), 30)
		fmt.Printf("\tGood for %f seconds.\n", secs)
		if !args["--continuous"].(bool) && !args["--web"].(bool) {
			break
		}
		time.Sleep(time.Second * time.Duration(secs))
	}
}

// saveSecrets overwrites the secrets file with TOTP secrets from memory
func saveSecrets(secretsFileName string, key []byte, secrets map[string]secret) error {
	contents := ""
	for _, secret := range secrets {
		contents += secret.url + "\n"
	}
	// encrypt secrets
	cipherText, err := encrypt(key, []byte(contents))
	if err != nil {
		return err
	}
	// write out the file
	err = ioutil.WriteFile(secretsFileName, cipherText, 0600)
	if err != nil {
		return err
	}
	return nil
}

// parseSecrets parses the TOTP secrets file
func parseSecrets(credFileRaw []byte) (map[string]secret, error) {
	credFile := strings.TrimRight(string(credFileRaw), "\r\n")
	secrets := make(map[string]secret)
	for id, c := range strings.Split(credFile, "\n") {
		if len(c) < 1 {
			log.Printf("Skipping blank line")
			continue
		}
		u, err := parseUrl(c)
		if err != nil {
			log.Printf("Skipping unrecognized line: %s", err.Error())
			continue
		}
		u.id = id
		secrets[u.name] = u
	}
	return secrets, nil
}

// parseUrl parses a otpauth: URL into its parts
func parseUrl(c string) (secret, error) {
	u, err := url.Parse(c)
	if err != nil {
		return secret{}, err
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		log.Printf("Skipping unrecognized secret: %s", c)
		return secret{}, err
	}
	v := u.Query()
	u.Path = strings.TrimLeft(u.Path, "/")
	secretBase32 := strings.ToUpper(v.Get("secret"))
	s, err := base32.StdEncoding.DecodeString(secretBase32)
	if err != nil {
		return secret{}, err
	}
	return secret{name: u.Path, secret: string(s), url: c}, nil
}

// webStaticHandler serves embedded static web files (js&css)
func webStaticHandler(w http.ResponseWriter, r *http.Request) {
	assetPath := r.URL.Path[1:]
	staticAsset, err := Asset(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	headers := w.Header()
	if strings.HasSuffix(assetPath, ".js") {
		headers["Content-Type"] = []string{"application/javascript"}
	} else if strings.HasSuffix(assetPath, ".css") {
		headers["Content-Type"] = []string{"text/css"}
	}
	io.Copy(w, bytes.NewReader(staticAsset))
}

// doWeb configures and runs the web server
func doWeb(listen string, secrets map[string]secret) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		page, err := Asset("assets/index.html")
		if err != nil {
			fatal(fmt.Errorf("Error with HTTP server template asset: %s", err.Error()))
		}
		fmt.Fprintf(w, string(page))
	})
	http.HandleFunc("/assets/", webStaticHandler)
	http.HandleFunc("/codes/", func(w http.ResponseWriter, r *http.Request) {
		resp := &codesets{}
		resp.Codes = []codeset{}
		// slice of sorted map keys
		keys := []string{}
		for k := range secrets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// return the codes in json format
		for _, name := range keys {
			c := secrets[name]
			code, err := totp.GetUserCode([]byte(c.secret), nil)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			resp.Codes = append(resp.Codes, codeset{name, code, c.id})
		}
		resp.Valid = 30 - math.Mod(float64(time.Now().Unix()), 30)
		out, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	})
	go func() {
		err := http.ListenAndServe(listen, nil)
		if err != nil {
			fatal(fmt.Errorf("Error starting web server: %s", err.Error()))
		}
	}()
}

// parseQr processes TOTP QR code
func parseQr(f string) (secret, error) {
	s := secret{}
	if _, err := exec.LookPath("zbarimg"); err != nil {
		return s, fmt.Errorf("Couldn't find the zbarimg command in your path.")
	}
	if _, err := os.Stat(f); err != nil {
		return s, err
	}
	out, err := exec.Command("zbarimg", "-q", f).Output()
	if err != nil {
		return s, err
	}
	if !strings.HasPrefix(string(out), "QR-Code:") {
		return s, fmt.Errorf("Unable to parse QR code.")
	}
	entry := strings.TrimPrefix(string(out), "QR-Code:")
	u, err := url.Parse(entry)
	if err != nil {
		return s, err
	}
	s.url = u.String()
	s.name = strings.TrimLeft(u.Path, "/")
	return s, nil
}

// encrypt does basic AES encryption using a random IV
func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

// decrypt does basic AES decryption
func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, fmt.Errorf("Encrypted secrets file is too short - possibly corrupted?")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, fmt.Errorf("Error decrypting secrets file - maybe your passphrase is wrong?")
	}
	return data, nil
}

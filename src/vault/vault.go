package vault

import (
	"html/template"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

type VaultApiInterface interface {
	Get(string) VaultData
	SetClient(*api.Client)
	Client() *api.Client
}

type Vault struct {
	api             VaultApiInterface
	policyTmpl      *template.Template
	secretsPathTmpl *template.Template
	authTmpl        *template.Template
	addr            string
	kubeTokenPath   string
}

func (v *Vault) Addr() string {
	return v.addr
}

func (v *Vault) KubeAddr() string {
	return v.kubeTokenPath
}

type VaultData map[string]interface{}

func New(addr, policyTmpl, secretsPathTmpl, authTmpl string) *Vault {
	v := new(Vault)
	v.addr = addr
	v.api = new(VaultApi)
	policyT, err := template.New("policy").Parse(policyTmpl)
	if err != nil {
		log.Fatalf("Policy template parser error:%s", err)
	}
	v.policyTmpl = policyT
	authT, err := template.New("auth").Parse(authTmpl)
	if err != nil {
		log.Fatalf("Auth template parser error:%s", err)
	}
	v.authTmpl = authT
	secretsPathT, err := template.New("secretspath").Parse(secretsPathTmpl)
	if err != nil {
		log.Fatalf("Auth template parser error:%s", err)
	}
	v.secretsPathTmpl = secretsPathT
	return v
}

func (v *Vault) SetToken(unwrap bool, vaultToken string) {
	var token string
	if len(vaultToken) > 0 {
		token = vaultToken
	} else {
		token = os.Getenv("VAULT_TOKEN")
		if len(token) > 0 {
			log.Debugf("Using token from environment variable VAULT_TOKEN")
		} else {
			log.Errorf("No token provided")
			os.Exit(1)
		}
	}
	if !unwrap {
		v.api.Client().SetToken(token)
		return
	}
	re, err := v.api.Client().Logical().Unwrap(token)
	if err != nil {
		log.Errorf("Can't unwrap token:%s, error:%s", token, err)
		os.Exit(1)
	}
	v.api.Client().SetToken(re.Auth.ClientToken)
}

func (v *Vault) Connect() *Vault {
	log.Debugf("Connecting to vault addr:%s", v.addr)
	c, err := api.NewClient(&api.Config{Address: v.addr})
	if err != nil {
		log.Errorf("Failed to connect to vault addr:%s, error:%s", v.addr, err)
		os.Exit(1)
	}
	v.api.SetClient(c)
	return v
}

func parseAuthPath(kubeAuth string) (role string, path string) {
	re := regexp.MustCompile(`^(.+?)@(.+)$`).FindStringSubmatch(kubeAuth)
	if len(re) == 3 {
		return re[1], re[2]
	} else {
		return "default", kubeAuth
	}
}

func (v *Vault) KubeAuth(kubeTokenPath, kubeAuth string) string {
	role, path := parseAuthPath(kubeAuth)
	jwt, err := ioutil.ReadFile(kubeTokenPath)
	if err != nil {
		log.Errorf("Can't read jwt token at %s", v.kubeTokenPath)
		os.Exit(1)
	}
	re, err := v.api.Client().Logical().Write("auth/"+path+"/login", map[string]interface{}{"role": role, "jwt": string(jwt)})
	if err != nil {
		log.Errorf("Can't authenticate jwt token path:%s, role:%s, error:%s", path, role, err)
		os.Exit(1)
	}
	return re.Auth.ClientToken
}

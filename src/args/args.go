package args

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

type Args struct {
	VerboseLevel     string
	AuthPath         string
	KubeTokenPath    string
	VaultAddr        string
	VaultToken       string
	Cluster          string
	ServiceAccount   string
	KubeAddr         string
	VaultPolicyT     string
	VaultAuthT       string
	VaultPolicyPathT string
	Unwrap           bool
	Args             []string
}

func New() *Args {
	return new(Args).Parse()
}

func (a *Args) Parse() *Args {
	flag.StringVar(&a.VerboseLevel, "verbose", "info", "Set verbosity level")
	flag.StringVar(&a.AuthPath, "authpath", "", "Authenticate with kubernetes, format: role@authengine")
	flag.StringVar(&a.KubeTokenPath, "kubetokenpath", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Kubernetes service account token path")
	flag.StringVar(&a.VaultAddr, "addr", os.Getenv("VAULT_ADDR"), "Vault address")
	flag.StringVar(&a.VaultToken, "token", "", "Vault token")
	flag.StringVar(&a.Cluster, "cluster", "", "Cluster name")
	flag.StringVar(&a.ServiceAccount, "serviceaccount", "default", "Service account")
	flag.StringVar(&a.KubeAddr, "kubeaddr", "", "Kubernetes api address")
	flag.StringVar(&a.VaultPolicyT, "vaultpolicy", "k8s/{{ .Cluster }}/{{ .Namespace }}", "Vault policy name template")
	flag.StringVar(&a.VaultPolicyPathT, "vaultpolicypath", "team/{{ .Namespace }}", "Vault policy path template")
	flag.StringVar(&a.VaultAuthT, "vaultauth", "k8s/{{ .Cluster }}/{{ .Namespace }}", "Vault auth path template")
	flag.BoolVar(&a.Unwrap, "unwrap", false, "Unwrap token")
	flag.Parse()
	a.Args = flag.Args()
	return a
}

func (a *Args) LogLevel() *Args {
	switch a.VerboseLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	return a
}

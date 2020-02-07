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

func env(name, def string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return def
}

func (a *Args) Parse() *Args {
	flag.StringVar(&a.VerboseLevel, "verbose", env("VERBOSE", "info"), "Set verbosity level")
	flag.StringVar(&a.AuthPath, "authPath", env("AUTH_PATH", ""), "Authenticate with kubernetes, format: role@authengine")
	flag.StringVar(&a.Cluster, "clusterName", env("CLUSTER_NAME", ""), "Cluster name")
	flag.StringVar(&a.ServiceAccount, "serviceaccount", env("SERVICE_ACCOUNT", "default"), "Service account")
	flag.StringVar(&a.KubeAddr, "kubeApiAddr", env("KUBE_API_ADDR", ""), "Kubernetes api address")
	flag.StringVar(&a.KubeTokenPath, "kubeTokenPath", env("KUBE_TOKEN_PATH", "/var/run/secrets/kubernetes.io/serviceaccount/token"), "Kubernetes service account token path")
	flag.StringVar(&a.VaultAddr, "vaultAddr", env("VAULT_ADDR", ""), "Vault address")
	flag.StringVar(&a.VaultToken, "vaultToken", env("VAULT_TOKEN", ""), "Vault token")
	flag.StringVar(&a.VaultPolicyT, "vaultPolicyName", env("VAULT_POLICY_NAME", "k8s/{{ .Cluster }}/{{ .Namespace }}"), "Vault policy name template")
	flag.StringVar(&a.VaultPolicyPathT, "vaultSecretsPath", env("VAULT_SECRETS_PATH", "k8s/{{ .Cluster }}/{{ .Namespace }}/*"), "Vault policy path template")
	flag.StringVar(&a.VaultAuthT, "vaultAuth", env("VAULT_AUTH", "k8s/{{ .Cluster }}/{{ .Namespace }}"), "Vault auth path template")
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

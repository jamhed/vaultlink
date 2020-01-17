package app

import (
	"os"
	"time"

	"vaultlink/args"
	"vaultlink/vault"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type App struct {
	vault     *vault.Vault
	args      *args.Args
	clientset *kubernetes.Clientset
}

type AppInterface interface {
	ClientSet() *kubernetes.Clientset
	Args() *args.Args
	Vault() *vault.Vault
}

func New() *App {
	a := new(App)
	a.args = args.New().LogLevel()
	a.vault = vault.New(a.Args().VaultAddr, a.Args().VaultPolicyT, a.Args().VaultPolicyPathT, a.Args().VaultAuthT).Connect()
	a.SetToken()
	return a
}

func (a *App) ClientSet() *kubernetes.Clientset {
	return a.clientset
}

func (a *App) Vault() *vault.Vault {
	return a.vault
}

func (a *App) Args() *args.Args {
	return a.args
}

func (a *App) SetToken() *App {
	var token string
	if len(a.args.KubeAuth) > 0 {
		token = a.vault.KubeAuth(a.args.KubeTokenPath, a.args.KubeAuth)
	} else {
		token = a.args.VaultToken
	}
	a.vault.SetToken(a.args.Unwrap, token)
	return a
}

func (a *App) Connect() *App {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("In-cluster config error:%s", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Clientset error:%s", err)
		os.Exit(1)
	}
	a.clientset = clientset
	return a
}

func (a *App) Control() {
	informerFactory := informers.NewSharedInformerFactory(a.ClientSet(), time.Second*30)

	informerFactory.Core().V1().Secrets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if secret, ok := obj.(*corev1.Secret); ok {
				a.onCreateSecret(secret)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if secret, ok := obj.(*corev1.Secret); ok {
				a.onDeleteSecret(secret)
			}
		},
	})

	informerFactory.Core().V1().Namespaces().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			if newNs, ok := new.(*corev1.Namespace); ok {
				if oldNs, ok := old.(*corev1.Namespace); ok {
					if newNs.GetResourceVersion() != oldNs.GetResourceVersion() {
						if newNs.Status.Phase == "Active" {
							a.onUpdateNamespace(oldNs, newNs)
						}
					}
				}
			}
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	informerFactory.Start(stop)
	for {
		time.Sleep(time.Second)
	}
}

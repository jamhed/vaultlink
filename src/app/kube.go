package app

import (
	"fmt"

	"vaultlink/vault"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ensureMap(m map[string]string) map[string]string {
	if m == nil {
		m = make(map[string]string)
	}
	return m
}

func ignoreErr(b []byte, err error) []byte {
	if err != nil {
		log.Errorf("ignored error:%s", err)
	}
	return b
}

func (a *App) bindVault(ns *corev1.Namespace) {
	namespace := ns.GetName()
	saName := a.Args().ServiceAccount
	sa, err := a.ClientSet().CoreV1().ServiceAccounts(namespace).Get(saName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get service account:%s namespace:%s %s", saName, namespace, err)
		return
	}
	secret, err := a.ClientSet().CoreV1().Secrets(namespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get 1secret:%s namespace:%s %s", sa.Secrets[0].Name, namespace, err)
		return
	}
	info := a.Vault().Bind(a.Args().Cluster, namespace, sa.GetName(), a.Args().KubeAddr, secret.Data["token"], secret.Data["ca.crt"])
	a.createReviewRole(namespace, sa.GetName())
	a.setNs(ns, info)
}

func (a *App) unbindVault(ns *corev1.Namespace) {
	namespace := ns.GetName()
	saName := a.Args().ServiceAccount
	a.Vault().Unbind(a.Args().Cluster, namespace, saName)
	a.deleteReviewRole(namespace, saName)
	a.unsetNs(ns)
}

func (a *App) setNs(ns *corev1.Namespace, info *vault.BindInfo) {
	if ns.Status.Phase != "Active" {
		return
	}

	ann := ensureMap(ns.GetAnnotations())
	ann["vault-link/bind"] = "true"
	ann["vault-link/vault"] = a.Vault().Addr()
	ann["vault-link/vault.auth"] = info.Auth
	ann["vault-link/vault.policy"] = info.Policy
	ann["vault-link/vault.policy-path"] = info.Policypath

	ns.SetAnnotations(ann)
	if _, err := a.ClientSet().CoreV1().Namespaces().Update(ns); err != nil {
		log.Errorf("Updating namespace:%s, error:%s", ns.GetName(), err)
	}
}

func (a *App) unsetNs(ns *corev1.Namespace) {
	if ns.Status.Phase != "Active" {
		return
	}
	ann := ensureMap(ns.GetAnnotations())
	delete(ann, "vault-link/bind")
	delete(ann, "vault-link/vault")
	delete(ann, "vault-link/vault.auth")
	delete(ann, "vault-link/vault.policy")
	delete(ann, "vault-link/vault.policy-path")

	ns.SetAnnotations(ann)
	if _, err := a.ClientSet().CoreV1().Namespaces().Update(ns); err != nil {
		log.Errorf("Updating namespace:%s, error:%s", ns.GetName(), err)
	}
}

func (a *App) onCreateSecret(secret *corev1.Secret) {
	namespace := secret.GetNamespace()
	nsClient := a.ClientSet().CoreV1().Namespaces()
	ns, err := nsClient.Get(namespace, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get namespace:%s %s", namespace, err)
		return
	}
	ann := ensureMap(ns.GetAnnotations())
	saName := ensureMap(secret.GetAnnotations())["kubernetes.io/service-account.name"]
	if !(saName == a.Args().ServiceAccount && ann["vault-link/bind"] == "true") {
		log.Debugf("Skip bind namespace:%s sa:%s annotation:%s", namespace, saName, ann["vault-link/bind"])
		return
	}
	a.bindVault(ns)
}

func (a *App) onDeleteSecret(secret *corev1.Secret) {
	namespace := secret.GetNamespace()
	nsClient := a.ClientSet().CoreV1().Namespaces()
	ns, err := nsClient.Get(namespace, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get namespace:%s %s", namespace, err)
		return
	}
	ann := ensureMap(ns.GetAnnotations())
	if ann["vault-link/bind"] != "true" {
		return
	}
	a.unbindVault(ns)
}

func (a *App) onUpdateNamespace(old, new *corev1.Namespace) {
	namespace := new.GetName()
	newAnn := ensureMap(new.GetAnnotations())
	oldAnn := ensureMap(old.GetAnnotations())
	log.Debugf("Update namespace:%s annotation new:%s old:%s", namespace, newAnn["vault-link/bind"], oldAnn["vault-link/bind"])
	if newAnn["vault-link/bind"] == "true" && oldAnn["vault-link/bind"] != "true" {
		a.bindVault(new)
	}
	if newAnn["vault-link/bind"] != "true" && oldAnn["vault-link/bind"] == "true" {
		a.unbindVault(new)
	}
}

func (a *App) createReviewRole(namespace, sa string) {
	name := fmt.Sprintf("%s-%s-tokenreview-binding", namespace, sa)
	log.Debugf("Create review role:%s", name)
	roleClient := a.ClientSet().RbacV1().ClusterRoleBindings()
	_, err := roleClient.Create(
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "system:auth-delegator",
			},
			Subjects: []rbacv1.Subject{rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			}},
		},
	)
	if err != nil {
		log.Errorf("Create review role name:%s, error:%s", name, err)
	}
}

func (a *App) deleteReviewRole(namespace, sa string) {
	name := fmt.Sprintf("%s-%s-tokenreview-binding", namespace, sa)
	log.Debugf("Delete review role:%s", name)
	roleClient := a.ClientSet().RbacV1().ClusterRoleBindings()
	err := roleClient.Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Delete review role name:%s, error:%s", name, err)
	}
}

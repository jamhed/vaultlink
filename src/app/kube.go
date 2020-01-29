package app

import (
	"fmt"

	"vaultlink/vault"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
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

func getOktaGroup(ns *corev1.Namespace) string {
	ann := ns.GetAnnotations()
	if ann == nil {
		return ""
	}
	if group, ok := ann["vault-link/group"]; ok {
		return group
	}
	return ""
}

func (a *App) bindVault(ns *corev1.Namespace) error {
	namespace := ns.GetName()
	saName := a.Args().ServiceAccount
	sa, err := a.ClientSet().CoreV1().ServiceAccounts(namespace).Get(saName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get service account:%s namespace:%s %s", saName, namespace, err)
		return err
	}
	secret, err := a.ClientSet().CoreV1().Secrets(namespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Get secret:%s namespace:%s %s", sa.Secrets[0].Name, namespace, err)
		return err
	}
	group := getOktaGroup(ns)
	if len(group) > 0 {
		info := a.Vault().Bind(a.Args().Cluster, namespace, saName, a.Args().KubeAddr, group, secret.Data["token"], secret.Data["ca.crt"])
		a.createReviewRole(namespace, saName)
		a.setNs(ns, info)
	} else {
		log.Warnf("No group annotation for namespace:%s", ns.Name)
	}
	return nil
}

func (a *App) unbindVault(ns *corev1.Namespace) {
	namespace := ns.GetName()
	saName := a.Args().ServiceAccount
	group := getOktaGroup(ns)
	a.Vault().Unbind(a.Args().Cluster, namespace, saName, group)
	a.deleteReviewRole(namespace, saName)
	a.deleteClusterRoleBinding(namespace, saName)
	a.unsetNs(ns)
}

func (a *App) setNs(ns *corev1.Namespace, info *vault.BindInfo) {
	if ns.Status.Phase != "Active" {
		return
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		nsTmp, err := a.ClientSet().CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ann := ensureMap(nsTmp.GetAnnotations())
		ann["vault-link/bind"] = "true"
		ann["vault-link/vault"] = a.Vault().Addr()
		ann["vault-link/vault.auth"] = info.Auth
		ann["vault-link/vault.policy"] = info.Policy
		ann["vault-link/vault.policy-path"] = info.Policypath
		nsTmp.SetAnnotations(ann)
		_, err = a.ClientSet().CoreV1().Namespaces().Update(nsTmp)
		return err
	})
	if retryErr != nil {
		log.Errorf("Updating namespace:%s, error:%s", ns.GetName(), retryErr)
	}
}

func (a *App) unsetNs(ns *corev1.Namespace) {
	if ns.Status.Phase != "Active" {
		return
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		nsTmp, err := a.ClientSet().CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ann := ensureMap(nsTmp.GetAnnotations())
		delete(ann, "vault-link/bind")
		delete(ann, "vault-link/vault")
		delete(ann, "vault-link/vault.auth")
		delete(ann, "vault-link/vault.policy")
		delete(ann, "vault-link/vault.policy-path")
		nsTmp.SetAnnotations(ann)
		_, err = a.ClientSet().CoreV1().Namespaces().Update(nsTmp)
		return err
	})
	if retryErr != nil {
		log.Errorf("Updating namespace:%s, error:%s", ns.GetName(), retryErr)
	}
}

func (a *App) deleteClusterRoleBinding(namespace, saName string) error {
	name := fmt.Sprintf("%s-%s-tokenreview-binding", namespace, saName)
	return a.ClientSet().RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
}

func (a *App) onCreateNamespace(ns *corev1.Namespace) {
	a.cache[ns.Name] = true
}

func (a *App) onUpdateNamespace(old, new *corev1.Namespace) {
	namespace := new.GetName()
	newAnn := ensureMap(new.GetAnnotations())
	oldAnn := ensureMap(old.GetAnnotations())
	if newAnn["vault-link/bind"] == "true" && oldAnn["vault-link/bind"] != "true" {
		log.Debugf("Bind namespace:%s", namespace)
		a.bindVault(new)
	} else if newAnn["vault-link/bind"] == "true" && a.cache[new.Name] {
		if a.bindVault(new) == nil {
			delete(a.cache, new.Name)
		}
	} else if newAnn["vault-link/bind"] != "true" && oldAnn["vault-link/bind"] == "true" {
		log.Debugf("Unbind namespace:%s", namespace)
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

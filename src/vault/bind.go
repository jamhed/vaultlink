package vault

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

type BindInfo struct {
	Auth       string
	Policy     string
	Policypath string
}

type Tmpl struct {
	Cluster        string
	Namespace      string
	ServiceAccount string
}

func (v *Vault) makeAuthName(cluster, namespace, sa string) string {
	var buf bytes.Buffer
	v.authTmpl.Execute(&buf, Tmpl{cluster, namespace, sa})
	return buf.String()
}

func (v *Vault) makePolicyName(cluster, namespace, sa string) string {
	var buf bytes.Buffer
	v.policyTmpl.Execute(&buf, Tmpl{cluster, namespace, sa})
	return buf.String()
}

func (v *Vault) makePolicyPathName(cluster, namespace, sa string) string {
	var buf bytes.Buffer
	v.policyPathTmpl.Execute(&buf, Tmpl{cluster, namespace, sa})
	return buf.String()
}

func (v *Vault) makeOktaGroupPath(group string) string {
	return fmt.Sprintf("auth/okta/groups/%s", group)
}

func (v *Vault) Unbind(cluster, namespace, sa string) {
	name := v.makeAuthName(cluster, namespace, sa)
	log.Infof("Disabling auth path:%s", name)
	err := v.api.Client().Sys().DisableAuth(name)
	if err != nil {
		log.Errorf("Disable auth:%s %s", name, err)
	}
	policyName := v.makePolicyName(cluster, namespace, sa)
	log.Infof("Deleting policy name:%s", policyName)
	err = v.api.Client().Sys().DeletePolicy(policyName)
	if err != nil {
		log.Errorf("Delete policy:%s error:%s", policyName, err)
	}
}

func (v *Vault) Bind(cluster, namespace, sa, kubeAddr, oktaGroup string, token, ca []byte) *BindInfo {
	name := v.makeAuthName(cluster, namespace, sa)
	log.Infof("Enabling auth path:%s", name)
	err := v.api.Client().Sys().EnableAuthWithOptions(name, &api.EnableAuthOptions{Type: "kubernetes"})
	if err != nil {
		log.Errorf("Enable auth:%s error:%s", name, err)
	}

	cfgPath := fmt.Sprintf("auth/%s/config", name)
	log.Infof("Configuring auth method:%s", cfgPath)
	_, err = v.api.Client().Logical().Write(cfgPath, VaultData{
		"token_reviewer_jwt": string(token),
		"kubernetes_host":    kubeAddr,
		"kubernetes_ca_cert": string(ca),
	})
	if err != nil {
		log.Errorf("Configure auth path:%s error:%s", cfgPath, err)
	}

	rolePath := fmt.Sprintf("auth/%s/role/%s", name, sa)
	policyName := v.makePolicyName(cluster, namespace, sa)

	log.Infof("Configuring auth role:%s", rolePath)
	_, err = v.api.Client().Logical().Write(rolePath, VaultData{
		"bound_service_account_names":      sa,
		"bound_service_account_namespaces": namespace,
		"policies":                         policyName,
		"token_num_uses":                   0,
		"token_ttl":                        "24h",
	})
	if err != nil {
		log.Errorf("Configure auth role path:%s error:%s", rolePath, err)
	}

	policyPath := v.makePolicyPathName(cluster, namespace, sa)
	log.Infof("Configuring policy:%s path:%s", policyName, policyPath)
	policy := fmt.Sprintf(`path "%s" {
capabilities = ["create", "read", "update", "delete", "list"]
}`, policyPath)
	err = v.api.Client().Sys().PutPolicy(policyName, policy)
	if err != nil {
		log.Errorf("Configuring policy:%s path:%s error:%s", policyName, policyPath, err)
	}

	oktaGroupPath := v.makeOktaGroupPath(oktaGroup)
	log.Infof("Configuring okta group mapping:%s", oktaGroupPath)
	_, err = v.api.Client().Logical().Write(oktaGroupPath, VaultData{
		"policies": []string{oktaGroup},
	})
	if err != nil {
		log.Errorf("Configure auth role path:%s error:%s", rolePath, err)
	}

	return &BindInfo{name, policyName, policyPath}
}

path "auth/k8s/ak8s-nv/*" {
	capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
path "sys/auth/k8s/ak8s-nv/*" {
	capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
path "sys/policies/acl/k8s/*" {
	capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
path "auth/okta/groups/*" {
	capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

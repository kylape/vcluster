{{/*
  Render a pod or container security context for the selected vCluster
  security profile. The restricted profile deliberately leaves UID/GID and
  fsGroup allocation to the platform (for example, OpenShift SCC).
*/}}
{{- define "vcluster.securityContext" -}}
{{- $context := deepCopy (default dict .context) -}}
{{- if eq (default "legacy" .profile) "restricted" -}}
{{- $_ := unset $context "runAsUser" -}}
{{- $_ := unset $context "runAsGroup" -}}
{{- $_ := set $context "runAsNonRoot" true -}}
{{- $_ := set $context "seccompProfile" (dict "type" "RuntimeDefault") -}}
{{- if .container -}}
{{- $_ := set $context "allowPrivilegeEscalation" false -}}
{{- $capabilities := deepCopy (default dict (get $context "capabilities")) -}}
{{- $_ := set $capabilities "drop" (list "ALL") -}}
{{- $_ := set $context "capabilities" $capabilities -}}
{{- end -}}
{{- end -}}
{{- toYaml $context -}}
{{- end -}}

{{- define "vcluster.securityProfile" -}}
{{- default "legacy" .Values.controlPlane.statefulSet.security.profile -}}
{{- end -}}

{{- define "vcluster.security.validate" -}}
{{- $profile := include "vcluster.securityProfile" . -}}
{{- if not (has $profile (list "legacy" "restricted")) -}}
{{- fail (printf "controlPlane.statefulSet.security.profile must be legacy or restricted, got %q" $profile) -}}
{{- end -}}
{{- $etcdProfile := .Values.controlPlane.backingStore.etcd.deploy.statefulSet.security.profile | default $profile -}}
{{- if not (has $etcdProfile (list "legacy" "restricted")) -}}
{{- fail (printf "controlPlane.backingStore.etcd.deploy.statefulSet.security.profile must be empty, legacy, or restricted, got %q" $etcdProfile) -}}
{{- end -}}
{{- if and (eq $profile "restricted") .Values.controlPlane.advanced.kubeVip.enabled -}}
{{- fail "controlPlane.advanced.kubeVip.enabled is incompatible with the restricted security profile because kube-vip requires NET_ADMIN and NET_RAW" -}}
{{- end -}}
{{- end -}}

// Package k8ssource implements the Kubernetes Source (ARCHITECTURE.md §7).
//
// What is built today: manifest mode (--manifests <dir>) reads workload YAML or
// rendered Helm output — Deployments, StatefulSets, DaemonSets, ReplicaSets,
// Jobs, CronJobs, and Pods — and extracts image references from containers,
// initContainers, and ephemeralContainers, deduped by ref. --namespace narrows
// the scan; a document with no metadata.namespace counts as "default", where
// `kubectl apply` would put it. No cluster access, and no network.
//
// What is NOT built: live-cluster scanning. New() without a ManifestsDir reports
// that and stops. The design calls for client-go typed clients (paginated,
// deduped by ownerRefs) fanning each unique image into imagesource — serial by
// default, --parallel-images opt-in — but none of that exists yet, so
// --parallel-images has nothing to parallelize: manifest mode lists images
// rather than pulling them.
package k8ssource

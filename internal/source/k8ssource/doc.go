// Package k8ssource implements the Kubernetes Source (ARCHITECTURE.md §7):
// client-go typed clients enumerate Deployments, StatefulSets, DaemonSets,
// Jobs, CronJobs, and Pods (paginated, deduped by ownerRefs), extract image
// references from containers, initContainers, and ephemeralContainers,
// dedupe them, and fan each unique image into imagesource — serial by
// default, --parallel-images opt-in.
//
// An offline mode (--manifests <dir>) runs the same extraction code over
// manifest YAML or rendered Helm output with no cluster access, honoring the
// --offline no-network guarantee (§13).
package k8ssource

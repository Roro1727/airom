// Package infra detects AI serving-infrastructure signals in deployment
// artifacts (ARCHITECTURE.md §4, §17): Dockerfiles (AI base images such as
// ollama or vllm, model-pulling build steps), docker-compose services, and
// Kubernetes manifests — emitting KindInfra and KindService claims eligible
// for SERVED_BY and CONFIGURES edges, using MethodConfig evidence.
//
// In-source infra client usage (an Ollama, vLLM, or TGI client call site) is
// declarative surface owned by rules/infra/*.yaml under the §6.3 bright
// line; this package covers the config-file side that needs real parsing of
// Dockerfile, compose, and manifest structure.
package infra

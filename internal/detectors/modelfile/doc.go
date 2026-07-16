// Package modelfile detects local model weight files by magic bytes and
// header-only parsing — core IP (ARCHITECTURE.md §4, §17): GGUF,
// safetensors, ONNX, torch zips (with static pickle opcode walking),
// TensorFlow SavedModel, TensorRT engines, TFLite, and HDF5. Parsers extract
// architecture, parameter count, quantization, and context length from
// headers alone; the scanner never loads or executes a model (§13).
//
// Every parser here eats untrusted bytes and is fuzzed in CI: it must return
// errors — never panic, never allocate unbounded (adversarial safetensors
// header lengths are capped, test-asserted). The pickle walker statically
// flags suspicious GLOBAL opcodes (os.system, subprocess, builtins.eval, …)
// as PickleRisk without executing anything — a security differentiator, not
// just inventory.
package modelfile

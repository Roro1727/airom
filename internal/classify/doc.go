// Package classify assigns each walked file its language, MIME class, and
// binary/text disposition from the path and the shared 32 KB header sample
// (ARCHITECTURE.md §3). It owns the magic-byte registry that identifies
// binary model formats (GGUF, safetensors, ONNX, torch-zip, SavedModel,
// TensorRT, TFLite, HDF5, Parquet, Arrow, …) for selector matching, and the
// language classification that scopes rule-pack `languages:` lists and
// selects the region lexer (§6.3, §6.4).
//
// Classification runs in the walker as part of decide-before-you-read (P3):
// it never reads beyond the header sample.
package classify

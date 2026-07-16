// Package prompt detects prompt assets stored as standalone files
// (ARCHITECTURE.md §4, §17): .txt/.md/.yaml/.jinja content judged by
// template heuristics — placeholder syntax, role markers, instruction shape
// — plus prompt-suggestive path signals, emitting KindPrompt claims that can
// receive PROMPTED_BY edges.
//
// In-code prompt usage (PromptTemplate, ChatPromptTemplate, system_prompt
// call sites) is deliberately NOT here: that is declarative surface owned by
// rules/prompts/*.yaml under the §6.3 bright line. This package exists for
// what a pattern cannot express — deciding whether a whole file is a prompt
// at all.
package prompt

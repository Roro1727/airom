package prompt

import (
	"context"
	"testing"
)

// TestCodegenTemplatesWithRoleAndSystemAreNotPrompts pins the bug the first
// chatShaped shipped with: it asked only whether the words "role" and "system"
// both appeared, by substring. Every RBAC/permissions/manifest generator answers
// yes — and landed on the content-confirmed tier (0.80), scoring HIGHER than a
// real prompt filed under prompts/ (0.60).
func TestCodegenTemplatesWithRoleAndSystemAreNotPrompts(t *testing.T) {
	cases := []struct{ name, path, body string }{
		{
			"RBAC role constants",
			"templates/rbac.go.j2",
			"{% for role in roles %}\nconst Role{{ role.name }} = \"{{ role.id }}\" // system: {{ role.system }}\n{% endfor %}\n",
		},
		{
			"permission table codegen",
			"codegen/permissions.py.j2",
			"{% for p in permissions %}\n{{ p.role }} = {{ p.mask }}  # system permission\n{% endfor %}\n",
		},
		{
			"SQL schema codegen",
			"tpl/audit_schema.sql.j2",
			"CREATE TABLE {{ table }} (\n  role TEXT NOT NULL, -- system audit\n  {{ col }} TEXT\n);\n",
		},
		{
			"kubernetes RoleBinding manifest",
			"templates/k8s_rolebinding.yaml.j2",
			"kind: RoleBinding\nroleRef:\n  name: {{ name }}\nsubjects:\n  - kind: ServiceAccount\n    name: system-{{ sa }}\n",
		},
		{
			"systemd unit generator: 'systemd' is not 'system'",
			"templates/unit.service.j2",
			"# the systemd role handler for {{ svc }}\nExecStart={{ bin }} --roles={{ roles }}\n",
		},
	}
	for _, c := range cases {
		got, err := NewPrompt().DetectFile(context.Background(), file(t, c.path, c.body))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("%s (%s): reported as a prompt @ %.2f; codegen mentioning role/system is not a chat template",
				c.name, c.path, got[0].Occurrence.Confidence)
		}
	}
}

// TestChatShapedRecognizesRealChatTemplates: the gate must still admit the real
// thing, including a HuggingFace chat_template.jinja that never says "system"
// (transformers >=4.43 writes this exact filename).
func TestChatShapedRecognizesRealChatTemplates(t *testing.T) {
	cases := []struct{ name, path, body string }{
		{
			"HF chat_template.jinja iterating messages/role",
			"chat_template.jinja",
			"{% for message in messages %}{{ '<' }}{{ message['role'] }}{{ '>' }}\n{{ message['content'] }}\n{% endfor %}\n",
		},
		{
			"role compared to a chat literal",
			"templates/agent.jinja",
			"{% if role == \"system\" %}\nYou are {{ agent_name }}, an agent.\n{% endif %}\n",
		},
		{
			"yaml chat messages",
			"templates/chat.yaml.j2",
			"- role: system\n  content: {{ instructions }}\n- role: user\n  content: {{ query }}\n",
		},
		{
			"HF generation prompt machinery",
			"tpl/llama.jinja",
			"{{ bos_token }}{% if add_generation_prompt %}[INST] {{ q }} [/INST]{% endif %}\n",
		},
		{
			"json chat message array",
			"prompts/messages.json",
			"[{\"role\": \"system\", \"content\": \"You are a helpful bot.\"}]\n",
		},
	}
	for _, c := range cases {
		got, err := NewPrompt().DetectFile(context.Background(), file(t, c.path, c.body))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("%s (%s): got %d findings, want 1 — this is a real chat template", c.name, c.path, len(got))
		}
	}
}

// TestPromptsStoredAsDataAreNotRefusedAsCode: .json/.toml are data, exactly as
// the .yaml admitted beside them is.
func TestPromptsStoredAsDataAreNotRefusedAsCode(t *testing.T) {
	const body = "system: You are a helpful research assistant.\n"
	for _, p := range []string{
		"prompts/greeting.yaml", "prompts/greeting.yml",
		"prompts/greeting.json", "prompts/greeting.toml",
	} {
		got, err := NewPrompt().DetectFile(context.Background(), file(t, p, body))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("%s: got %d findings, want 1; a prompt stored as data is not source code", p, len(got))
		}
	}
}

func TestHasWord(t *testing.T) {
	cases := []struct {
		lower, tok string
		want       bool
	}{
		{"the systemd role handler", "system", false}, // the original bug
		{"roles and systems", "role", false},
		{"roles and systems", "system", false},
		{`{% if role == "system" %}`, "role", true},
		{`{% if role == "system" %}`, "system", true},
		{"{{ message['role'] }}", "role", true},
		{"subsystem", "system", false},
		{"my_system", "system", false}, // '_' continues an identifier
		{"system", "system", true},
		{"", "role", false},
		{"rol", "role", false},
	}
	for _, c := range cases {
		if got := hasWord(c.lower, c.tok); got != c.want {
			t.Errorf("hasWord(%q, %q) = %v, want %v", c.lower, c.tok, got, c.want)
		}
	}
}

import OpenAI from "openai";

const client = new OpenAI();

export async function run(prompt: string): Promise<string> {
  const completion = await client.chat.completions.create({
    model: "claude-sonnet-4-5",
    max_tokens: 512,
    messages: [{ role: "user", content: prompt }],
  });
  return completion.choices[0]?.message?.content ?? "";
}

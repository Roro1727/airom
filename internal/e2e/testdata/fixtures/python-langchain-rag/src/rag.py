"""A tiny retrieval-augmented-generation pipeline used as an e2e fixture."""

import chromadb
import openai

EMBED_MODEL = "text-embedding-3-large"


def build_store() -> "chromadb.api.ClientAPI":
    return chromadb.PersistentClient(path="./chroma-store")


def answer(client: openai.OpenAI, question: str, context: str) -> str:
    response = client.chat.completions.create(
        model="gpt-4.1",
        temperature=0.2,
        max_tokens=800,
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": f"{context}\n\n{question}"},
        ],
    )
    return response.choices[0].message.content

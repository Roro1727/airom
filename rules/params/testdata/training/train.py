"""Standalone training / fine-tuning hyperparameter fixture.

Exercises the multiline TrainingArguments(...) call shape (key=value) and a
dict-of-config shape ("key": value), so both separators are covered — the same
standalone fallback story as the aiconfig pack.
"""

from transformers import TrainingArguments

args = TrainingArguments(
    output_dir="./out",
    # airom: training/optimization
    learning_rate=2e-5,
    # airom: training/optimization
    weight_decay=0.01,
    # airom: training/optimization
    max_grad_norm=1.0,
    # airom: training/optimization
    adam_epsilon=1e-8,
    # airom: training/schedule
    num_train_epochs=3,
    # airom: training/schedule
    per_device_train_batch_size=8,
    # airom: training/schedule
    gradient_accumulation_steps=4,
    # airom: training/schedule
    warmup_ratio=0.03,
)

# OpenAI fine-tune hyperparameters, dict / yaml-json shape.
finetune = {
    "learning_rate_multiplier": 0.1,  # airom: training/optimization
    "n_epochs": 3,  # airom: training/schedule
}

# Negative cases — the keyword is present, but the anchored numeric pattern
# must reject: the name is not at line start, or the value is not a number.
# airom-ok: training/optimization
note = "set the learning_rate later"

# airom-ok: training/schedule
doc = "num_train_epochs drives the loop"

# airom-ok: training/optimization
schedule = "warmup then decay the weight_decay"

# A keyword-bearing line whose value is a string, not a number, stays silent.
# airom-ok: training/schedule
per_device_train_batch_size = "auto"

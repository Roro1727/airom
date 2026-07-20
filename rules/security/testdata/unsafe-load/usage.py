"""Fixture for the unsafe-load pack: torch.load weights_only opt-out.

Positives explicitly pass weights_only=False; negatives use the safe path or a
bare call (safe-by-default in torch >= 2.6).
"""

import torch

# airom: unsafe-load/torch-weights-only-false
model = torch.load("ckpt.pt", weights_only=False)

# One level of nested parens (loading from an opened file) still matches.
# airom: unsafe-load/torch-weights-only-false
m2 = torch.load(open("ckpt.pt", "rb"), weights_only=False)

# Negative — the safe path is not flagged.
# airom-ok: unsafe-load/torch-weights-only-false
safe = torch.load("ckpt.pt", weights_only=True)

# Negative — a bare load is safe-by-default in torch >= 2.6, so not flagged.
# airom-ok: unsafe-load/torch-weights-only-false
bare = torch.load("ckpt.pt")

# Negative — an identifier merely ending in weights_only must not match the
# arg name (the leading \b guards against not_weights_only / use_weights_only).
# airom-ok: unsafe-load/torch-weights-only-false
wrap = torch.load(f, not_weights_only=False)

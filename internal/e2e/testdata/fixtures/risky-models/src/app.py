"""Loads a checkpoint through an unsafe path — the code-level unsafe-load risk."""
import torch


def load_checkpoint(path):
    # weights_only=False opts out of PyTorch's safe deserialization path.
    return torch.load(path, weights_only=False)

"""Pytest fixtures and hooks for bridge tests."""

import pytest

import hivescope_bridge.main as bridge_main
from hivescope_bridge.store import store


@pytest.fixture(autouse=True)
def reset_store():
    store.clear()


@pytest.fixture(autouse=True)
def reset_demo_flag():
    bridge_main._demo_running = False

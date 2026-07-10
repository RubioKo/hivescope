"""Tests for the pluggy-based adapter registry."""

from hivescope_bridge.adapters.registry import hookimpl, hookspec


def test_hookspec_and_hookimpl_are_callable():
    assert callable(hookspec)
    assert callable(hookimpl)


def test_hookimpl_returns_adapter_class():
    class DummyAdapter:
        pass

    @hookimpl
    def hivescope_adapter():
        return DummyAdapter

    result = hivescope_adapter()
    assert result is DummyAdapter

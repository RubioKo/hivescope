from pluggy import HookimplMarker, HookspecMarker

hookspec = HookspecMarker("hivescope")
hookimpl = HookimplMarker("hivescope")


@hookspec
def hivescope_adapter() -> type:
    """Register an adapter plugin. Must return a BaseAdapter subclass."""

from typing import Annotated
import dagger
from dagger.server import Environment


env = Environment()


def base_image() -> dagger.Container:
    return (
        dagger.apko().wolfi(["python-3.11", "py3.11-pip"])
		.with_mounted_cache("/root/.cache/pip", dagger.cache_volume("pip_cache"))
		.with_mounted_cache("/root/.local/pipx/cache", dagger.cache_volume("pipx_cache"))
		.with_mounted_cache("/root/.cache/hatch", dagger.cache_volume("hatch_cache"))
        .with_exec(["pip", "install", "pytest", "shiv"])
        .with_mounted_directory("/src", dagger.host().directory("."))
        .with_workdir("/src/universe/_demo/client")
        .with_exec(["pip", "install", "."])
    )

@env.command
async def publish(version: Annotated[str, "The version to publish."]) -> str:
    """Publish the client"""
    if version == "nope":
        print("OH NO! Publishing the client failed!")
        raise RuntimeError("Publishing failed")

    return f"Published client version {version}"

@env.check
async def unit_test() -> str:
    """Run unit tests"""
    return await (
        base_image()
        .with_exec(["pytest", "-v"])
        .stdout()
    )

@env.artifact
async def client_image() -> dagger.Container:
    """The client app and its dependencies packaged up into a container"""
    client_app = (
        base_image()
        .with_exec(["shiv", "-e", "src.client.main:main", "-o", "/client", "/src/universe/_demo/client", "--root", "/tmp/.shiv"])
        .file("/client")
    )
    return base_image().with_file("/usr/bin/client", client_app).with_entrypoint(["/usr/bin/client"])

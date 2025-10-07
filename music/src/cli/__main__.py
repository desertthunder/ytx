"""CLI entry point for YouTube Music proxy server."""

import typer
import uvicorn

app = typer.Typer(add_completion=False)

TPort = typer.Option(8080, help="Port to run the server on")
THost = typer.Option("127.0.0.1", help="Host to bind the server to")
TReload = typer.Option(False, help="Enable auto-reload for development")


@app.command()
def serve(port: int = TPort, host: str = THost, reload: bool = TReload) -> None:
    """Start the FastAPI proxy server for YouTube Music API."""
    uvicorn.run("api.app:app", host=host, port=port, reload=reload)


if __name__ == "__main__":
    app()

"""API middleware."""

import sys
import time
from typing import Callable

from fastapi import Request, Response
from loguru import logger
from starlette.middleware.base import BaseHTTPMiddleware


def configure_logging() -> None:
    """Configure loguru with custom format that includes extra fields."""
    logger.remove()
    logger.add(
        sys.stderr,
        format=(
            "<green>{time:YYYY-MM-DD HH:mm:ss.SSS}</green> | <level>{level: <5}</level> | "
            "<cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - "
            "<level>{message}</level> {extra}"
        ),
        level="DEBUG",
    )


class LoggingMiddleware(BaseHTTPMiddleware):
    """Middleware for logging HTTP requests and responses.

    Logs request/response details including optional response body preview.

    Response bodies are captured for JSON responses under 10KB.
    """

    # 10KB
    MAX_BODY_SIZE = 10 * 1024

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """Log request and response details.

        Args:
            request: Incoming HTTP request
            call_next: Next middleware or route handler

        Returns:
            HTTP response
        """
        start_time = time.time()

        logger.info(
            f"{request.method} {request.url.path}",
            extra={
                "client": request.client.host if request.client else None,
                "method": request.method,
                "path": request.url.path,
                "query": str(request.query_params) if request.query_params else None,
            },
        )

        try:
            response = await call_next(request)
        except Exception as exc:
            duration = time.time() - start_time
            logger.error(
                f"Request failed: {request.method} {request.url.path}",
                extra={"duration_ms": round(duration * 1000, 2), "error": str(exc)},
            )
            raise

        duration = time.time() - start_time
        log_level = "error" if response.status_code >= 400 else "info"
        log_extra = {"status_code": response.status_code, "duration_ms": round(duration * 1000, 2)}

        message = (f"{request.method} {request.url.path} - {response.status_code}",)
        getattr(logger, log_level)(message, extra=log_extra)

        return response

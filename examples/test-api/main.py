import json

from fastapi import FastAPI, Request

app = FastAPI()


@app.get("/health")
async def health():
    return {"status": "server is healthy"}


@app.post("/echo")
async def echo(request: Request):
    body = await request.body()
    return {
        "method": request.method,
        "url": str(request.url),
        "headers": dict(request.headers),
        "query_params": dict(request.query_params),
        "body": body.decode() if body else None,
    }


@app.get("/echo")
async def echo_get(request: Request):
    return {
        "method": request.method,
        "url": str(request.url),
        "headers": dict(request.headers),
        "query_params": dict(request.query_params),
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)

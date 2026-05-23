def greet(name: str) -> str:
    cleaned = name.strip() or "world"
    return f"Hello, {cleaned}!"

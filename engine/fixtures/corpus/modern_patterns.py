from functools import wraps
import asyncio

def rate_limit(limit):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            return func(*args, **kwargs)
        return wrapper
    return decorator

def audit_log(action):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            return func(*args, **kwargs)
        return wrapper
    return decorator

# Nested list comprehension
matrix = [[1, 2, 3], [4, 5, 6]]
flat = [num for row in matrix for num in row if num > 0]

# Decorator chain
@rate_limit(10)
@audit_log("fetch_records")
def process_data(request):
    val = request.args.get("v", "")
    
    # Modern match-case (Python 3.10+)
    match val:
        case "A":
            mode = 1
        case "B":
            mode = 2
        case _:
            mode = 0
            
    return mode

# Async generator
async def async_counter():
    for i in range(5):
        await asyncio.sleep(0.01)
        yield i
